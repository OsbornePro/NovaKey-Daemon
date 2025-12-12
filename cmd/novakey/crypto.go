package main

import (
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	protocolVersion = 2
	msgTypePassword = 1

	defaultDevicesFile = "devices.json"

	maxClockSkewSec = 120  // allow up to ±120 seconds clock skew
	maxMsgAgeSec    = 300  // reject messages older than 5 minutes
	replayCacheTTL  = 600  // keep seen nonces for 10 minutes

	maxRequestsPerDevicePerMin = 60 // per-device rate limit
)

type deviceConfig struct {
	ID     string `json:"id"`
	KeyHex string `json:"key_hex"`
}

type devicesConfigFile struct {
	Devices []deviceConfig `json:"devices"`
}

// deviceAEAD holds the AEAD cipher for a device.
type deviceAEAD struct {
	id   string
	aead cipher.AEAD
}

var deviceCiphers map[string]deviceAEAD

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

// initCrypto loads per-device keys and builds AEADs.
// Call this from main() before listening.
func initCrypto() error {
	path := os.Getenv("NOVAKEY_DEVICES_FILE")
	if path == "" {
		path = defaultDevicesFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading devices file %q: %w", path, err)
	}

	var cfg devicesConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing devices file %q: %w", path, err)
	}

	if len(cfg.Devices) == 0 {
		return fmt.Errorf("devices file %q has no devices", path)
	}

	m := make(map[string]deviceAEAD, len(cfg.Devices))
	for _, d := range cfg.Devices {
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

		a, err := chacha20poly1305.NewX(keyBytes)
		if err != nil {
			return fmt.Errorf("device %q: NewX failed: %w", d.ID, err)
		}
		m[d.ID] = deviceAEAD{id: d.ID, aead: a}
	}

	deviceCiphers = m

	absPath, _ := filepath.Abs(path)
	fmt.Printf("Loaded %d device keys from %s\n", len(deviceCiphers), absPath)
	return nil
}

// decryptPasswordFrame parses and decrypts a v2 frame payload and returns (deviceID, password).
//
// Frame layout:
//   [0]               = version
//   [1]               = msgType
//   [2]               = idLen
//   [3 : 3+idLen]     = deviceID
//   [3+idLen : 3+idLen+nonceLen] = nonce
//   [rest]            = ciphertext
//
// Plaintext layout (after AEAD decrypt):
//   [0:8]   = timestamp (uint64, unix seconds, big-endian)
//   [8:...] = password (UTF-8)
func decryptPasswordFrame(frame []byte) (string, string, error) {
	if len(frame) < 3 {
		return "", "", fmt.Errorf("frame too short: %d", len(frame))
	}
	if frame[0] != protocolVersion {
		return "", "", fmt.Errorf("unsupported protocol version: %d", frame[0])
	}
	if frame[1] != msgTypePassword {
		return "", "", fmt.Errorf("unexpected msgType: %d", frame[1])
	}

	idLen := int(frame[2])
	if idLen <= 0 {
		return "", "", fmt.Errorf("invalid idLen: %d", idLen)
	}
	if len(frame) < 3+idLen {
		return "", "", fmt.Errorf("frame too short for idLen=%d", idLen)
	}
	deviceID := string(frame[3 : 3+idLen])

	if deviceCiphers == nil {
		return "", "", fmt.Errorf("crypto not initialized")
	}
	dev, ok := deviceCiphers[deviceID]
	if !ok {
		return "", "", fmt.Errorf("unknown deviceID: %q", deviceID)
	}

	headerEnd := 3 + idLen
	header := frame[:headerEnd]

	nonceLen := dev.aead.NonceSize()
	if len(frame) < headerEnd+nonceLen+dev.aead.Overhead() {
		return "", "", fmt.Errorf("frame too short for nonce+ciphertext")
	}

	nonce := frame[headerEnd : headerEnd+nonceLen]
	ciphertext := frame[headerEnd+nonceLen:]

	plaintext, err := dev.aead.Open(nil, nonce, ciphertext, header)
	if err != nil {
		return "", "", fmt.Errorf("AEAD.Open failed for device %q: %w", deviceID, err)
	}
	if len(plaintext) < 8 {
		return "", "", fmt.Errorf("plaintext too short for timestamp: %d", len(plaintext))
	}

	ts := int64(binary.BigEndian.Uint64(plaintext[:8]))
	password := string(plaintext[8:])

	if err := validateFreshnessAndRate(deviceID, nonce, ts); err != nil {
		return "", "", err
	}

	return deviceID, password, nil
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
		// Reset window
		rw.windowStart = now
		rw.count = 0
	}

	rw.count++
	rateState[deviceID] = rw

	if rw.count > maxRequestsPerDevicePerMin {
		return fmt.Errorf("rate limit exceeded for device %q: %d requests in current window",
			deviceID, rw.count)
	}

	return nil
}

