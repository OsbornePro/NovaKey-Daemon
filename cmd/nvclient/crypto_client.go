// cmd/nvclient/crypto_client.go
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"filippo.io/mlkem768"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	protocolVersion = 3
	msgTypePassword = 1
)

var (
	clientDeviceID   string
	deviceStaticKey  []byte // from devices.json key_hex
	serverEncapKey   []byte // ML-KEM-768 encapsulation key (public)
)

// initCryptoClient initializes the client’s device static key and
// the server’s Kyber public key (EncapsulationKey).
//
// deviceKeyHex: 32-byte device key (hex), must match devices.json.
// serverKyberPubB64: base64-encoded ML-KEM-768 EncapsulationKey from server_keys.json.
func initCryptoClient(id, deviceKeyHex, serverKyberPubB64 string) error {
	if id == "" {
		return fmt.Errorf("device id must not be empty")
	}
	clientDeviceID = id

	// Per-device static key
	keyBytes, err := hex.DecodeString(deviceKeyHex)
	if err != nil {
		return fmt.Errorf("invalid key_hex: %w", err)
	}
	if len(keyBytes) != chacha20poly1305.KeySize {
		return fmt.Errorf("device static key must be %d bytes, got %d",
			chacha20poly1305.KeySize, len(keyBytes))
	}
	deviceStaticKey = keyBytes

	// Server ML-KEM-768 encapsulation key
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

// encryptPasswordFrame builds a v3 payload:
//
//   [0]               = version
//   [1]               = msgType
//   [2]               = idLen
//   [3 : 3+idLen]     = deviceID
//
//   H = 3 + idLen
//   [H : H+1]         = kemCtLen (uint16, BE)
//   [H+2 : H+2+kemCtLen] = kemCt (ML-KEM-768 ciphertext)
//
//   K = H + 2 + kemCtLen
//   [K : K+nonceLen]  = nonce (XChaCha20-Poly1305)
//   [rest]            = ciphertext
//
// Plaintext:
//
//   [0:8]   = timestamp (uint64, unix seconds, BE)
//   [8:...] = password
func encryptPasswordFrame(password string) ([]byte, error) {
	if deviceStaticKey == nil || len(deviceStaticKey) == 0 {
		return nil, fmt.Errorf("device static key not initialized")
	}
	if serverEncapKey == nil || len(serverEncapKey) == 0 {
		return nil, fmt.Errorf("server Kyber public key not initialized")
	}

	idBytes := []byte(clientDeviceID)
	if len(idBytes) == 0 || len(idBytes) > 255 {
		return nil, fmt.Errorf("deviceID length invalid: %d (must be 1..255)", len(idBytes))
	}
	idLen := byte(len(idBytes))

	// 1) KEM encapsulation to get (kemCt, sharedKem)
	kemCt, sharedKem, err := mlkem768.Encapsulate(serverEncapKey)
	if err != nil {
		return nil, fmt.Errorf("mlkem768.Encapsulate: %w", err)
	}
	if len(kemCt) != mlkem768.CiphertextSize {
		return nil, fmt.Errorf("internal: kemCt length %d != CiphertextSize %d",
			len(kemCt), mlkem768.CiphertextSize)
	}

	// 2) Derive AEAD key from sharedKem + deviceStaticKey
	aeadKey, err := deriveAEADKey(deviceStaticKey, sharedKem)
	if err != nil {
		return nil, err
	}

	aead, err := chacha20poly1305.NewX(aeadKey)
	if err != nil {
		return nil, fmt.Errorf("NewX with derived key failed: %w", err)
	}

	// 3) Build header (AAD)
	// header = version || msgType || idLen || deviceID || kemCtLen || kemCt
	header := make([]byte, 0, 3+len(idBytes)+2+len(kemCt))
	header = append(header, protocolVersion)
	header = append(header, msgTypePassword)
	header = append(header, idLen)
	header = append(header, idBytes...)

	var kemLenBuf [2]byte
	binary.BigEndian.PutUint16(kemLenBuf[:], uint16(len(kemCt)))
	header = append(header, kemLenBuf[:]...)
	header = append(header, kemCt...)

	// 4) Plaintext = timestamp || password
	now := time.Now().Unix()
	pwBytes := []byte(password)
	plaintext := make([]byte, 8+len(pwBytes))
	binary.BigEndian.PutUint64(plaintext[:8], uint64(now))
	copy(plaintext[8:], pwBytes)

	// 5) Nonce + AEAD
	nonceLen := aead.NonceSize()
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("rand.Read nonce: %w", err)
	}

	ct := aead.Seal(nil, nonce, plaintext, header)

	// 6) Final frame
	out := make([]byte, 0, len(header)+len(nonce)+len(ct))
	out = append(out, header...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

