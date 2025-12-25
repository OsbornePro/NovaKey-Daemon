// cmd/novakey/crypto.go
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

	// Outer msgType (must remain 1; approve/inject is inside the decrypted body)
	msgTypePassword = 1

	defaultDevicesFile = "devices.json"

	maxClockSkewSec = 120 // allow up to ±120 seconds clock skew
	maxMsgAgeSec    = 300 // reject messages older than 5 minutes
	replayCacheTTL  = 600 // keep seen nonces for 10 minutes

	maxRequestsPerDevicePerMin = 60 // per-device rate limit
)

var ErrNotPaired = errors.New("not paired (devices file missing/empty)")

// devices.json format
type deviceConfig struct {
	ID     string `json:"id"`
	KeyHex string `json:"key_hex"` // 32-byte per-device static key (hex)
}

type devicesConfigFile struct {
	Devices []deviceConfig `json:"devices"`
}

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

// initCrypto loads/creates server Kyber keys and attempts to load devices.json.
// IMPORTANT: If devices.json is missing, we DO NOT fail hard anymore.
// We start "unpaired" (devices map empty) so the pairing bootstrap can run.
func initCrypto() error {
	// 1) Load or create server Kyber keys.
	if err := loadOrCreateServerKeys(cfg.ServerKeysFile); err != nil {
		return fmt.Errorf("loading server Kyber keys: %w", err)
	}
	if serverDecapKey == nil || len(serverEncapKey) == 0 {
		return fmt.Errorf("server keys not initialized")
	}

	// 2) Try load per-device static keys.
	path := cfg.DevicesFile
	if path == "" {
		path = defaultDevicesFile
	}

	m, err := loadDevicesFromFile(path)
	if err != nil {
		// If missing or empty, keep running but mark as unpaired.
		if errors.Is(err, ErrNotPaired) {
			devices = make(map[string]deviceState)
			log.Printf("[pair] %v (will start pairing bootstrap)", err)
			return nil
		}
		return err
	}

	devices = m
	absPath, _ := filepath.Abs(path)
	log.Printf("Loaded %d device keys from %s", len(devices), absPath)
	return nil
}

func isPaired() bool {
	return devices != nil && len(devices) > 0
}

// reloadDevicesFromDisk re-reads devices.json after pairing completes.
func reloadDevicesFromDisk() error {
	path := cfg.DevicesFile
	if path == "" {
		path = defaultDevicesFile
	}
	m, err := loadDevicesFromFile(path)
	if err != nil {
		return err
	}
	devices = m
	absPath, _ := filepath.Abs(path)
	log.Printf("[pair] reloaded %d device keys from %s", len(devices), absPath)
	return nil
}

func loadDevicesFromFile(path string) (map[string]deviceState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s not found", ErrNotPaired, path)
		}
		return nil, fmt.Errorf("reading devices file %q: %w", path, err)
	}

	var dc devicesConfigFile
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("parsing devices file %q: %w", path, err)
	}
	if len(dc.Devices) == 0 {
		return nil, fmt.Errorf("%w: %s has no devices", ErrNotPaired, path)
	}

	m := make(map[string]deviceState, len(dc.Devices))
	for _, d := range dc.Devices {
		if d.ID == "" {
			return nil, fmt.Errorf("device with empty id in %q", path)
		}
		keyBytes, err := hex.DecodeString(d.KeyHex)
		if err != nil {
			return nil, fmt.Errorf("device %q: invalid key_hex: %w", d.ID, err)
		}
		if len(keyBytes) != chacha20poly1305.KeySize {
			return nil, fmt.Errorf("device %q: key must be %d bytes, got %d",
				d.ID, chacha20poly1305.KeySize, len(keyBytes))
		}

		m[d.ID] = deviceState{
			id:        d.ID,
			staticKey: keyBytes,
		}
	}
	return m, nil
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

// decryptMessageFrame decrypts a v3 outer frame and returns a typed *inner* message.
func decryptMessageFrame(frame []byte) (deviceID string, msgType uint8, payload []byte, err error) {
	devID, plaintext, nonce, err := decryptOuterV3(frame)
	if err != nil {
		return "", 0, nil, err
	}
	if len(plaintext) < 8 {
		return "", 0, nil, fmt.Errorf("plaintext too short for timestamp: %d", len(plaintext))
	}

	ts := int64(binary.BigEndian.Uint64(plaintext[:8]))
	if err := validateFreshnessAndRate(devID, nonce, ts); err != nil {
		return "", 0, nil, err
	}

	body := plaintext[8:]

	// Typed inner frame?
	if len(body) >= 1 && body[0] == byte(frameVersionV1) {
		innerDev, innerType, innerPayload, derr := decodeMessageFrame(body)
		if derr != nil {
			return "", 0, nil, fmt.Errorf("invalid inner message frame: %w", derr)
		}
		if innerDev != devID {
			return "", 0, nil, fmt.Errorf("inner deviceID mismatch (outer=%q inner=%q)", devID, innerDev)
		}
		return devID, innerType, innerPayload, nil
	}

	// Legacy: timestamp + UTF-8 password string
	return devID, MsgTypeInject, body, nil
}

func decryptOuterV3(frame []byte) (string, []byte, []byte, error) {
	if len(frame) < 3 {
		return "", nil, nil, fmt.Errorf("frame too short: %d", len(frame))
	}
	if frame[0] != protocolVersion {
		return "", nil, nil, fmt.Errorf("unsupported protocol version: %d", frame[0])
	}
	if frame[1] != msgTypePassword {
		return "", nil, nil, fmt.Errorf("unexpected msgType: %d", frame[1])
	}

	idLen := int(frame[2])
	if idLen <= 0 {
		return "", nil, nil, fmt.Errorf("invalid idLen: %d", idLen)
	}
	if len(frame) < 3+idLen {
		return "", nil, nil, fmt.Errorf("frame too short for idLen=%d", idLen)
	}
	deviceID := string(frame[3 : 3+idLen])

	if devices == nil {
		return "", nil, nil, fmt.Errorf("crypto not initialized (devices map nil)")
	}
	dev, ok := devices[deviceID]
	if !ok {
		return "", nil, nil, fmt.Errorf("unknown deviceID: %q", deviceID)
	}
	if serverDecapKey == nil {
		return "", nil, nil, fmt.Errorf("serverDecapKey is nil")
	}

	headerBaseEnd := 3 + idLen // start of kemCtLen
	if len(frame) < headerBaseEnd+2 {
		return "", nil, nil, fmt.Errorf("frame too short for kemCtLen")
	}

	kemCtLen := int(binary.BigEndian.Uint16(frame[headerBaseEnd : headerBaseEnd+2]))
	if kemCtLen != mlkem768.CiphertextSize {
		return "", nil, nil, fmt.Errorf("invalid kemCtLen: got %d expected %d", kemCtLen, mlkem768.CiphertextSize)
	}

	kemStart := headerBaseEnd + 2
	kemEnd := kemStart + kemCtLen
	if len(frame) < kemEnd {
		return "", nil, nil, fmt.Errorf("frame too short for kemCt (len=%d)", kemCtLen)
	}

	kemCt := frame[kemStart:kemEnd]
	header := frame[:kemEnd] // AAD

	sharedKem, err := mlkem768.Decapsulate(serverDecapKey, kemCt)
	if err != nil {
		return "", nil, nil, fmt.Errorf("mlkem768.Decapsulate failed: %w", err)
	}

	aeadKey, err := deriveAEADKey(dev.staticKey, sharedKem)
	if err != nil {
		return "", nil, nil, err
	}

	aead, err := chacha20poly1305.NewX(aeadKey)
	if err != nil {
		return "", nil, nil, fmt.Errorf("NewX with derived key failed: %w", err)
	}

	rest := frame[kemEnd:]
	nonceLen := aead.NonceSize()
	if len(rest) < nonceLen+aead.Overhead() {
		return "", nil, nil, fmt.Errorf("frame too short for nonce+ciphertext")
	}

	nonce := rest[:nonceLen]
	ciphertext := rest[nonceLen:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, header)
	if err != nil {
		return "", nil, nil, fmt.Errorf("AEAD.Open failed for device %q: %w", deviceID, err)
	}

	return deviceID, plaintext, nonce, nil
}

func validateFreshnessAndRate(deviceID string, nonce []byte, ts int64) error {
	now := time.Now().Unix()

	if ts > now+maxClockSkewSec {
		return fmt.Errorf("message timestamp is in the future (ts=%d, now=%d)", ts, now)
	}
	if now-ts > maxMsgAgeSec {
		return fmt.Errorf("message too old (ts=%d, now=%d)", ts, now)
	}

	nonceHex := hex.EncodeToString(nonce)

	replayMu.Lock()
	defer replayMu.Unlock()

	m, ok := replayCache[deviceID]
	if !ok {
		m = make(map[string]int64)
		replayCache[deviceID] = m
	}
	if prevTs, exists := m[nonceHex]; exists {
		return fmt.Errorf("replay detected for device %q (nonce seen at ts=%d)", deviceID, prevTs)
	}
	m[nonceHex] = ts

	for k, v := range m {
		if now-v > replayCacheTTL {
			delete(m, k)
		}
	}

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

func loadDevicesFromFileWindows(path string) (map[string]deviceState, error) {
	// 1) Read outer wrapper
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrap dpapiFile
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, fmt.Errorf("parse dpapi wrapper: %w", err)
	}
	if wrap.V != 1 || wrap.DPAPIB64 == "" {
		return nil, fmt.Errorf("invalid dpapi wrapper")
	}

	// 2) Decode + unprotect
	ct, err := dpapiDecode(wrap.DPAPIB64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode dpapi blob: %w", err)
	}
	pt, err := dpapiUnprotect(ct)
	if err != nil {
		return nil, fmt.Errorf("dpapi unprotect: %w", err)
	}

	// 3) pt is JSON of devicesConfigFile
	var dc devicesConfigFile
	if err := json.Unmarshal(pt, &dc); err != nil {
		return nil, fmt.Errorf("parse devices json inside dpapi: %w", err)
	}

	return buildDevicesMap(dc, path)
}

func buildDevicesMap(dc devicesConfigFile, path string) (map[string]deviceState, error) {
	if len(dc.Devices) == 0 {
		return nil, fmt.Errorf("%w: %s has no devices", ErrNotPaired, path)
	}

	m := make(map[string]deviceState, len(dc.Devices))
	for _, d := range dc.Devices {
		// (same checks you already have)
		...
	}
	return m, nil
}
