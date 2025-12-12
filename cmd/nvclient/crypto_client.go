// cmd/nvclient/crypto_client.go
package main

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	protocolVersion = 2
	msgTypePassword = 1
)

var aead cipher.AEAD
var deviceID string

// initCryptoClient initializes AEAD with the given hex key and device ID.
func initCryptoClient(id string, keyHex string) error {
	if id == "" {
		return fmt.Errorf("device id must not be empty")
	}
	deviceID = id

	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("invalid key_hex: %w", err)
	}
	if len(keyBytes) != chacha20poly1305.KeySize {
		return fmt.Errorf("key must be %d bytes, got %d", chacha20poly1305.KeySize, len(keyBytes))
	}

	a, err := chacha20poly1305.NewX(keyBytes)
	if err != nil {
		return fmt.Errorf("NewX: %w", err)
	}
	aead = a
	return nil
}

// encryptPasswordFrame builds a v2 payload:
//
//   [0]               = version
//   [1]               = msgType
//   [2]               = idLen
//   [3 : 3+idLen]     = deviceID
//   [3+idLen : 3+idLen+nonceLen] = nonce
//   [rest]            = ciphertext
//
func encryptPasswordFrame(password string) ([]byte, error) {
	if aead == nil {
		return nil, fmt.Errorf("crypto not initialized")
	}
	idBytes := []byte(deviceID)
	if len(idBytes) > 255 {
		return nil, fmt.Errorf("deviceID too long (%d bytes, max 255)", len(idBytes))
	}
	idLen := byte(len(idBytes))

	// header used as AAD
	header := make([]byte, 0, 3+len(idBytes))
	header = append(header, protocolVersion)
	header = append(header, msgTypePassword)
	header = append(header, idLen)
	header = append(header, idBytes...)

	nonceLen := aead.NonceSize()
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("rand.Read nonce: %w", err)
	}

	ct := aead.Seal(nil, nonce, []byte(password), header)

	out := make([]byte, 0, len(header)+len(nonce)+len(ct))
	out = append(out, header...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

