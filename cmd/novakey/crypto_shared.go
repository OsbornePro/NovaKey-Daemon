package main

import (
	"crypto/sha256"
	"errors"
	"io"

	"github.com/cloudflare/circl/kem/kyber/kyber768"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	kyberCtSize   = 1088
	kyberSSSize   = 32
	sessionKeyLen = 32

	// Context string for HKDF and AEAD associated data.
	transportContext = "NovaKey-transport-v1"
)

// GenerateKeyPair returns a Kyber768 key pair (private first, public second).
func GenerateKeyPair() (*kyber768.PrivateKey, *kyber768.PublicKey, error) {
	// nil = use crypto/rand.Reader automatically
	pub, priv, err := kyber768.GenerateKeyPair(nil)
	if err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// Decapsulate derives the session key from the ciphertext using Kyber768
// and HKDF-SHA256. The returned key is always 32 bytes.
func Decapsulate(priv *kyber768.PrivateKey, ct []byte) ([]byte, error) {
	if priv == nil || len(ct) != kyberCtSize {
		return nil, errors.New("invalid private key or ciphertext length")
	}

	// Raw Kyber shared secret.
	ss := make([]byte, kyberSSSize)
	priv.DecapsulateTo(ss, ct)
	defer zeroBytes(ss)

	// Derive a transport session key via HKDF-SHA256 with a fixed context.
	h := hkdf.New(sha256.New, ss, nil, []byte(transportContext))

	key := make([]byte, sessionKeyLen)
	if _, err := io.ReadFull(h, key); err != nil {
		zeroBytes(key)
		return nil, err
	}

	return key, nil
}

// DecryptPayload decrypts the payload using XChaCha20-Poly1305, binding it to
// the supplied associated data (AAD). The AAD must be identical to the one
// used during encryption (header + context string).
func DecryptPayload(key, payload, aad []byte) ([]byte, error) {
	if len(payload) < chacha20poly1305.NonceSizeX {
		return nil, errors.New("payload too short")
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce := payload[:chacha20poly1305.NonceSizeX]
	ciphertext := payload[chacha20poly1305.NonceSizeX:]
	return aead.Open(nil, nonce, ciphertext, aad)
}
