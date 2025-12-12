package main

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	protocolVersion = 2
	msgTypePassword = 1
)

// TODO: later load this from config / pairing, not source code.
// MUST be 32 bytes.
var staticKey = []byte{
	0x10, 0x21, 0x32, 0x43, 0x54, 0x65, 0x76, 0x87,
	0x98, 0xa9, 0xba, 0xcb, 0xdc, 0xed, 0xfe, 0x0f,
	0xf0, 0xe1, 0xd2, 0xc3, 0xb4, 0xa5, 0x96, 0x87,
	0x78, 0x69, 0x5a, 0x4b, 0x3c, 0x2d, 0x1e, 0x0f,
}

var aead cipherAEAD

type cipherAEAD interface {
	NonceSize() int
	Overhead() int
	Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error)
	Seal(dst, nonce, plaintext, additionalData []byte) []byte
}

// initCrypto must be called from main() before accepting connections.
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

// decryptPasswordFrame parses and decrypts a v2 frame payload and returns the password.
// frame layout:
//   [0]   = version
//   [1]   = msgType
//   [2:2+nonceLen] = nonce
//   [2+nonceLen:]  = ciphertext
func decryptPasswordFrame(frame []byte) (string, error) {
	if len(frame) < 2 {
		return "", fmt.Errorf("frame too short: %d", len(frame))
	}
	if frame[0] != protocolVersion {
		return "", fmt.Errorf("unsupported protocol version: %d", frame[0])
	}
	if frame[1] != msgTypePassword {
		return "", fmt.Errorf("unexpected msgType: %d", frame[1])
	}
	if aead == nil {
		return "", fmt.Errorf("crypto not initialized")
	}

	header := frame[:2]
	nonceLen := aead.NonceSize()
	if len(frame) < 2+nonceLen+aead.Overhead() {
		return "", fmt.Errorf("frame too short for nonce+ciphertext: %d", len(frame))
	}
	nonce := frame[2 : 2+nonceLen]
	ciphertext := frame[2+nonceLen:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, header)
	if err != nil {
		return "", fmt.Errorf("AEAD.Open failed: %w", err)
	}
	return string(plaintext), nil
}

// encryptPasswordFrame is used only by the test client; you don't need it on the server,
// but it's handy to keep symmetric. If you want, move this into a separate client package.
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

