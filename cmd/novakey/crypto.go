// cmd/novakey/crypto.go
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sync"
	"time"

	"filippo.io/mlkem768"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	protocolVersion = 3
	msgTypePassword = 1

	defaultDevicesFile = "devices.json"

	maxClockSkewSec = 120
	maxMsgAgeSec    = 300
	replayCacheTTL  = 600

	maxRequestsPerDevicePerMin = 60
)

var ErrNotPaired = errors.New("not paired (devices file missing/empty)")

type deviceConfig struct {
	ID     string `json:"id"`
	KeyHex string `json:"key_hex"` // 32 bytes hex
}

type devicesConfigFile struct {
	Devices []deviceConfig `json:"devices"`
}

type deviceState struct {
	id        string
	staticKey []byte
}

// Protect devices map (pairing reload swaps it).
var (
	devicesMu sync.RWMutex
	devices   map[string]deviceState
)

var (
	replayMu    sync.Mutex
	replayCache = make(map[string]map[string]int64)
	rateState   = make(map[string]rateWindow)
)

type rateWindow struct {
	windowStart int64
	count       int
}

func initCrypto() error {
	if err := loadOrCreateServerKeys(cfg.ServerKeysFile); err != nil {
		return fmt.Errorf("loading server Kyber keys: %w", err)
	}
	if serverDecapKey == nil || len(serverEncapKey) == 0 {
		return fmt.Errorf("server keys not initialized")
	}

	path := cfg.DevicesFile
	if path == "" {
		path = defaultDevicesFile
	}

	m, err := loadDevicesFromDisk(path)
	if err != nil {
    	if errors.Is(err, ErrNotPaired) {
        	devicesMu.Lock()
	        devices = make(map[string]deviceState)
    	    devicesMu.Unlock()

        	log.Printf("[pair] %v (no paired devices found; pairing is available)", err)
        	return nil
    	}

    	// This includes ErrDevicesUnavailable and any other read/decrypt/parse error.
    	log.Printf("[fatal] device store error: %v", err)
    	return err
	}
	devicesMu.Lock()
	devices = m
	devicesMu.Unlock()

	absPath, _ := filepath.Abs(path)
	log.Printf("Loaded %d device keys from %s", len(m), absPath)
	return nil
}

func isPaired() bool {
	devicesMu.RLock()
	defer devicesMu.RUnlock()
	return devices != nil && len(devices) > 0
}

func reloadDevicesFromDisk() error {
	path := cfg.DevicesFile
	if path == "" {
		path = defaultDevicesFile
	}

	m, err := loadDevicesFromDisk(path)
	if err != nil {
		return err
	}

	devicesMu.Lock()
	devices = m
	devicesMu.Unlock()

	absPath, _ := filepath.Abs(path)
	log.Printf("[pair] reloaded %d device keys from %s", len(m), absPath)
	return nil
}

func buildDevicesMap(dc devicesConfigFile, path string) (map[string]deviceState, error) {
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
		m[d.ID] = deviceState{id: d.ID, staticKey: keyBytes}
	}
	return m, nil
}

func deriveAEADKey(deviceKey, sharedKem []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, sharedKem, deviceKey, []byte("NovaKey v3 AEAD key"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, fmt.Errorf("hkdf derive AEAD key: %w", err)
	}
	return key, nil
}

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

	// Snapshot device state under RLock
	devicesMu.RLock()
	dev, ok := devices[deviceID]
	devicesMu.RUnlock()

	if devices == nil {
		return "", nil, nil, fmt.Errorf("crypto not initialized (devices map nil)")
	}
	if !ok {
		return "", nil, nil, fmt.Errorf("unknown deviceID: %q", deviceID)
	}
	if serverDecapKey == nil {
		return "", nil, nil, fmt.Errorf("serverDecapKey is nil")
	}

	headerBaseEnd := 3 + idLen
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
		return "", nil, nil, fmt.Errorf("frame too short for kemCt")
	}

	kemCt := frame[kemStart:kemEnd]
	header := frame[:kemEnd]

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
		return "", nil, nil, fmt.Errorf("NewX failed: %w", err)
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

	// --- Freshness checks (unchanged behavior) ---
	if ts > now+maxClockSkewSec {
		return fmt.Errorf("message timestamp is in the future (ts=%d, now=%d)", ts, now)
	}
	if now-ts > maxMsgAgeSec {
		return fmt.Errorf("message too old (ts=%d, now=%d)", ts, now)
	}

	nonceHex := hex.EncodeToString(nonce)

	replayMu.Lock()
	defer replayMu.Unlock()

	// Ensure per-device replay map exists.
	m, ok := replayCache[deviceID]
	if !ok {
		m = make(map[string]int64)
		replayCache[deviceID] = m
	}

	// Evict old replay entries (based on when we saw them).
	for k, seenAt := range m {
		if now-seenAt > replayCacheTTL {
			delete(m, k)
		}
	}

	// --- Rate limiting (do BEFORE consuming nonce) ---
	rw := rateState[deviceID]
	if rw.windowStart == 0 || now-rw.windowStart >= 60 {
		rw.windowStart = now
		rw.count = 0
	}

	limit := maxRequestsPerDevicePerMin
	if cfg.MaxRequestsPerMin > 0 {
		limit = cfg.MaxRequestsPerMin
	}

	// If this request would exceed the limit, reject WITHOUT recording the nonce.
	if rw.count+1 > limit {
		return fmt.Errorf("rate limit exceeded for device %q: %d requests in window (limit=%d)",
			deviceID, rw.count+1, limit)
	}

	// --- Replay check (only after passing rate limit) ---
	if prevSeenAt, exists := m[nonceHex]; exists {
		return fmt.Errorf("replay detected for device %q (nonce seen at ts=%d)", deviceID, prevSeenAt)
	}

	// Commit state: consume nonce + count request as accepted.
	m[nonceHex] = now
	rw.count++
	rateState[deviceID] = rw

	return nil
}
