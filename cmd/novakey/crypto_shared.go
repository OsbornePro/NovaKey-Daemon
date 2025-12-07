package main

import (
	"errors"

	"github.com/cloudflare/circl/kem/kyber/kyber768"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	kyberCtSize   = 1088
	kyberSSSize   = 32
	sessionKeyLen = 32
)

// GenerateKeyPair returns a Kyber768 key pair (private first, public second)
func GenerateKeyPair() (*kyber768.PrivateKey, *kyber768.PublicKey, error) {
	// nil = use crypto/rand.Reader automatically
	pub, priv, err := kyber768.GenerateKeyPair(nil)
	if err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// Decapsulate derives the 32-byte shared secret from the ciphertext
func Decapsulate(priv *kyber768.PrivateKey, ct []byte) ([]byte, error) {
	if priv == nil || len(ct) != kyberCtSize {
		return nil, errors.New("invalid private key or ciphertext length")
	}

	ss := make([]byte, kyberSSSize)
	priv.DecapsulateTo(ss, ct)
	return ss[:sessionKeyLen], nil // always 32 bytes, but safe
}

// DecryptPayload decrypts the payload using XChaCha20-Poly1305
func DecryptPayload(key, payload []byte) ([]byte, error) {
	if len(payload) < chacha20poly1305.NonceSizeX {
		return nil, errors.New("payload too short")
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce := payload[:chacha20poly1305.NonceSizeX]
	ciphertext := payload[chacha20poly1305.NonceSizeX:]
	return aead.Open(nil, nonce, ciphertext, nil)
}
