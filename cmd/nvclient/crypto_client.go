// cmd/nvclient/crypto_client.go
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"filippo.io/mlkem768"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	protocolVersion = 3
)

var (
	clientDeviceID  string
	deviceStaticKey []byte // from devices.json key_hex (32 bytes)
	serverEncapKey  []byte // ML-KEM-768 encapsulation key (public)
)

// deviceKeyHex: 32-byte device key (hex), must match devices.json.
// serverKyberPubB64: base64-encoded ML-KEM-768 EncapsulationKey from server_keys.json.
func initCryptoClient(id, deviceKeyHex, serverKyberPubB64 string) error {
	if id == "" {
		return fmt.Errorf("device id must not be empty")
	}
	clientDeviceID = id

	keyBytes, err := hex.DecodeString(deviceKeyHex)
	if err != nil {
		return fmt.Errorf("invalid key_hex: %w", err)
	}
	if len(keyBytes) != chacha20poly1305.KeySize {
		return fmt.Errorf("device static key must be %d bytes, got %d",
			chacha20poly1305.KeySize, len(keyBytes))
	}
	deviceStaticKey = keyBytes

	pub, err := base64.StdEncoding.DecodeString(serverKyberPubB64)
	if err != nil {
		return fmt.Errorf("decoding server Kyber pub (base64): %w", err)
	}
	if len(pub) != mlkem768.EncapsulationKeySize {
		return fmt.Errorf("server Kyber pub has wrong length: got %d, want %d",
			len(pub), mlkem768.EncapsulationKeySize)
	}
	serverEncapKey = pub

	return nil
}

// deriveAEADKey mirrors the server’s HKDF construction.
func deriveAEADKey(deviceKey, sharedKem []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, sharedKem, deviceKey, []byte("NovaKey v3 AEAD key"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, fmt.Errorf("hkdf derive AEAD key: %w", err)
	}
	return key, nil
}
