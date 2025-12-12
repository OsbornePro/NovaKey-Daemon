// cmd/novakey/crypto.go
package main

import (
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	protocolVersion = 2
	msgTypePassword = 1

	defaultDevicesFile = "devices.json"
)

type deviceConfig struct {
	ID    string `json:"id"`
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
	password := string(plaintext)
	return deviceID, password, nil
}

