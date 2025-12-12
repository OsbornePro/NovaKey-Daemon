// cmd/nvclient/crypto_client.go
package main

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	protocolVersion = 2
	msgTypePassword = 1
)

// IMPORTANT: this must match the staticKey in cmd/novakey/crypto.go
var staticKey = []byte{
	0x10, 0x21, 0x32, 0x43, 0x54, 0x65, 0x76, 0x87,
	0x98, 0xa9, 0xba, 0xcb, 0xdc, 0xed, 0xfe, 0x0f,
	0xf0, 0xe1, 0xd2, 0xc3, 0xb4, 0xa5, 0x96, 0x87,
	0x78, 0x69, 0x5a, 0x4b, 0x3c, 0x2d, 0x1e, 0x0f,
}

var aead cipher.AEAD

func initCrypto() error {
	if len(staticKey) != chacha20poly1305.KeySize {
		return fmt.Errorf("staticKey must be %d bytes, have %d",
			chacha20poly1305.KeySize, len(staticKey))
	}
	a, err := chacha20poly1305.NewX(staticKey)
	if err != nil {
		return fmt.Errorf("NewX: %w", err)
	}
	aead = a
	return nil
}

// encryptPasswordFrame builds a v2 payload:
//   [0]   = version
//   [1]   = msgType
//   [2:2+nonceLen] = nonce
//   [2+nonceLen:]  = ciphertext
func encryptPasswordFrame(password string) ([]byte, error) {
	if aead == nil {
		return nil, fmt.Errorf("crypto not initialized")
	}

	header := []byte{protocolVersion, msgTypePassword}

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

