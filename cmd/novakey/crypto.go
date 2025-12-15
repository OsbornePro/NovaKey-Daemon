// cmd/novakey/crypto.go
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"filippo.io/mlkem768"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	// Outer transport protocol (current)
	protocolVersion = 3

	defaultDevicesFile = "devices.json"

	maxClockSkewSec = 120 // allow up to ±120 seconds clock skew
	maxMsgAgeSec    = 300 // reject messages older than 5 minutes
	replayCacheTTL  = 600 // keep seen nonces for 10 minutes

	maxRequestsPerDevicePerMin = 60 // per-device rate limit
)

// devices.json format
type deviceConfig struct {
	ID     string `json:"id"`
	KeyHex string `json:"key_hex"` // 32-byte per-device static key (hex)
}

type devicesConfigFile struct {
	Devices []deviceConfig `json:"devices"`
}

// For v3 we no longer store a long-lived AEAD per device.
// We derive a fresh AEAD key per message from:
//
//   HKDF-SHA256( ikm = kemShared, salt = deviceStaticKey, info = "NovaKey v3 AEAD key" )
//
// deviceState holds the static per-device secret.
type deviceState struct {
	id        string
	staticKey []byte // 32 bytes from devices.json
}

var devices map[string]deviceState

// replayCache: deviceID -> nonceHex -> timestamp
// rateState:  deviceID -> rateWindow
var (
	replayMu    sync.Mutex
	replayCache = make(map[string]map[string]int64)
	rateState   = make(map[string]rateWindow)
)

type rateWindow struct {
	windowStart int64
	count       int
}

// initCrypto loads server Kyber keys and per-device static keys.
func initCrypto() error {
	// 1. Load or create server Kyber keys (fills serverDecapKey / serverEncapKey).
	if err := loadOrCreateServerKeys(cfg.ServerKeysFile); err != nil {
		return fmt.Errorf("loading server Kyber keys: %w", err)
	}
	if serverDecapKey == nil {
		return fmt.Errorf("serverDecapKey is nil after loadOrCreateServerKeys")
	}

	// 2. Load per-device static keys from devices.json.
	path := cfg.DevicesFile
	if path == "" {
		path = defaultDevicesFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading devices file %q: %w", path, err)
	}

	var dc devicesConfigFile
	if err := json.Unmarshal(data, &dc); err != nil {
		return fmt.Errorf("parsing devices file %q: %w", path, err)
	}

	if len(dc.Devices) == 0 {
		return fmt.Errorf("devices file %q has no devices", path)
	}

	m := make(map[string]deviceState, len(dc.Devices))
	for _, d := range dc.Devices {
		if d.ID == "" {
			return fmt.Errorf("device with empty id in %q", path)
		}
		keyBytes, err := hex.DecodeString(d.KeyHex)
		if err != nil {
			return fmt.Errorf("device %q: invalid key_hex: %w", d.ID, err)
		}
		if len(keyBytes) != chacha20poly1305.KeySize {
			return fmt.Errorf("device %q: key must be %d bytes, got %d",
				d.ID, chacha20poly1305.KeySize, len(keyBytes))
		}

		m[d.ID] = deviceState{
			id:        d.ID,
			staticKey: keyBytes,
		}
	}

	devices = m

	absPath, _ := filepath.Abs(path)
	fmt.Printf("Loaded %d device keys from %s\n", len(devices), absPath)
	return nil
}

// deriveAEADKey derives a per-message AEAD key from the KEM shared key and the device static key.
func deriveAEADKey(deviceKey, sharedKem []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, sharedKem, deviceKey, []byte("NovaKey v3 AEAD key"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, fmt.Errorf("hkdf derive AEAD key: %w", err)
	}
	return key, nil
}

// decryptMessageFrame decrypts a v3 outer frame and returns:
//
//   deviceID, msgType, payloadString
//
// Current (no-legacy) behavior:
// - msgType is the OUTER msgType byte (1 = inject, 2 = approve).
// - plaintext is: [u64 timestamp][payload bytes]
// - payload bytes are interpreted as UTF-8 for inject; approve may be empty.
func decryptMessageFrame(frame []byte) (string, uint8, string, error) {
	deviceID, msgType, plaintext, nonce, err := decryptOuterV3(frame)
	if err != nil {
		return "", 0, "", err
	}

	if len(plaintext) < 8 {
		return "", 0, "", fmt.Errorf("plaintext too short for timestamp: %d", len(plaintext))
	}

	ts := int64(binary.BigEndian.Uint64(plaintext[:8]))
	if err := validateFreshnessAndRate(deviceID, nonce, ts); err != nil {
		return "", 0, "", err
	}

	body := plaintext[8:]
	return deviceID, msgType, string(body), nil
}

// decryptPasswordFrame parses and decrypts a v3 frame payload and returns (deviceID, password).
//
// Kept for compatibility with older main paths.
// If msgType is approve, return approveMagicDefault() so older approve-magic logic still works.
func decryptPasswordFrame(frame []byte) (string, string, error) {
	dev, msgType, payload, err := decryptMessageFrame(frame)
	if err != nil {
		return "", "", err
	}
	if msgType == MsgTypeApprove {
		return dev, approveMagicDefault(), nil
	}
	return dev, payload, nil
}

func approveMagicDefault() string {
	if cfg.ApproveMagic != "" {
		return cfg.ApproveMagic
	}
	return "__NOVAKEY_APPROVE__"
}

// decryptOuterV3 performs the outer v3 parsing, KEM decapsulation, AEAD open.
// It returns (deviceID, msgType, plaintext, nonce, error).
//
// v3 frame layout:
//
//   [0]               = version (0x03)
//   [1]               = msgType (1=inject, 2=approve)
//   [2]               = idLen
//   [3 : 3+idLen]     = deviceID
//   [3+idLen : ...]   = payload as follows:
//
//      H = 3 + idLen
//      H..H+1               = kemCtLen (uint16, big-endian)
//      H+2..H+2+kemCtLen-1   = kemCt (ML-KEM-768 ciphertext)
//
//      K = H + 2 + kemCtLen
//      K..K+nonceLen-1       = nonce (XChaCha20-Poly1305, 24 bytes)
//      K+nonceLen..end       = ciphertext (AEAD)
//
// Plaintext layout (after AEAD decrypt):
//
//   [0:8]   = timestamp (uint64, unix seconds, big-endian)
//   [8:...] = payload bytes (UTF-8 for inject; may be empty for approve)
//
// AEAD AAD (header) is everything up to the start of the nonce:
//   header = frame[0:K] = version || msgType || idLen || deviceID || kemCtLen || kemCt
func decryptOuterV3(frame []byte) (string, uint8, []byte, []byte, error) {
	if len(frame) < 3 {
		return "", 0, nil, nil, fmt.Errorf("frame too short: %d", len(frame))
	}
	if frame[0] != protocolVersion {
		return "", 0, nil, nil, fmt.Errorf("unsupported protocol version: %d", frame[0])
	}

	msgType := frame[1]
	if msgType != MsgTypeInject && msgType != MsgTypeApprove {
		return "", 0, nil, nil, fmt.Errorf("unexpected msgType: %d", msgType)
	}

	idLen := int(frame[2])
	if idLen <= 0 {
		return "", 0, nil, nil, fmt.Errorf("invalid idLen: %d", idLen)
	}
	if len(frame) < 3+idLen {
		return "", 0, nil, nil, fmt.Errorf("frame too short for idLen=%d", idLen)
	}
	deviceID := string(frame[3 : 3+idLen])

	if devices == nil {
		return "", 0, nil, nil, fmt.Errorf("crypto not initialized (devices map nil)")
	}
	dev, ok := devices[deviceID]
	if !ok {
		return "", 0, nil, nil, fmt.Errorf("unknown deviceID: %q", deviceID)
	}
	if serverDecapKey == nil {
		return "", 0, nil, nil, fmt.Errorf("serverDecapKey is nil")
	}

	headerBaseEnd := 3 + idLen // start of kemCtLen
	if len(frame) < headerBaseEnd+2 {
		return "", 0, nil, nil, fmt.Errorf("frame too short for kemCtLen")
	}

	kemCtLen := int(binary.BigEndian.Uint16(frame[headerBaseEnd : headerBaseEnd+2]))
	if kemCtLen != mlkem768.CiphertextSize {
		return "", 0, nil, nil, fmt.Errorf("invalid kemCtLen: got %d expected %d", kemCtLen, mlkem768.CiphertextSize)
	}

	kemStart := headerBaseEnd + 2
	kemEnd := kemStart + kemCtLen
	if len(frame) < kemEnd {
		return "", 0, nil, nil, fmt.Errorf("frame too short for kemCt (len=%d)", kemCtLen)
	}

	kemCt := frame[kemStart:kemEnd]
	header := frame[:kemEnd] // AAD = header up through kemCt

	sharedKem, err := mlkem768.Decapsulate(serverDecapKey, kemCt)
	if err != nil {
		return "", 0, nil, nil, fmt.Errorf("mlkem768.Decapsulate failed: %w", err)
	}

	aeadKey, err := deriveAEADKey(dev.staticKey, sharedKem)
	if err != nil {
		return "", 0, nil, nil, err
	}

	aead, err := chacha20poly1305.NewX(aeadKey)
	if err != nil {
		return "", 0, nil, nil, fmt.Errorf("NewX with derived key failed: %w", err)
	}

	rest := frame[kemEnd:]
	nonceLen := aead.NonceSize()
	if len(rest) < nonceLen+aead.Overhead() {
		return "", 0, nil, nil, fmt.Errorf("frame too short for nonce+ciphertext")
	}

	nonce := rest[:nonceLen]
	ciphertext := rest[nonceLen:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, header)
	if err != nil {
		return "", 0, nil, nil, fmt.Errorf("AEAD.Open failed for device %q: %w", deviceID, err)
	}

	return deviceID, msgType, plaintext, nonce, nil
}

// validateFreshnessAndRate enforces timestamp window, replay protection, and rate limiting.
func validateFreshnessAndRate(deviceID string, nonce []byte, ts int64) error {
	now := time.Now().Unix()

	// Basic time window checks
	if ts > now+maxClockSkewSec {
		return fmt.Errorf("message timestamp is in the future (ts=%d, now=%d)", ts, now)
	}
	if now-ts > maxMsgAgeSec {
		return fmt.Errorf("message too old (ts=%d, now=%d)", ts, now)
	}

	nonceHex := hex.EncodeToString(nonce)

	replayMu.Lock()
	defer replayMu.Unlock()

	// ----- Replay protection -----
	m, ok := replayCache[deviceID]
	if !ok {
		m = make(map[string]int64)
		replayCache[deviceID] = m
	}

	if prevTs, exists := m[nonceHex]; exists {
		return fmt.Errorf("replay detected for device %q (nonce seen at ts=%d)", deviceID, prevTs)
	}
	m[nonceHex] = ts

	// Cleanup stale nonces
	for k, v := range m {
		if now-v > replayCacheTTL {
			delete(m, k)
		}
	}

	// ----- Rate limiting -----
	rw := rateState[deviceID]

	if now-rw.windowStart >= 60 || rw.windowStart == 0 {
		rw.windowStart = now
		rw.count = 0
	}

	rw.count++
	rateState[deviceID] = rw

	limit := maxRequestsPerDevicePerMin
	if cfg.MaxRequestsPerMin > 0 {
		limit = cfg.MaxRequestsPerMin
	}
	if rw.count > limit {
		return fmt.Errorf("rate limit exceeded for device %q: %d requests in current window (limit=%d)",
			deviceID, rw.count, limit)
	}

	return nil
}

