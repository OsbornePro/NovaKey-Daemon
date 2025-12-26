// cmd/novakey/pair_token.go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

// Pairing token state. This replaces the HTTP bootstrap state.
// Token is presented by phone during /pair to avoid random LAN pairing attempts.
type pairTokenState struct {
	mu sync.Mutex

	active  bool
	token   []byte // raw bytes
	tokenID string // short printable id (for logs)
	expires time.Time

	// Optional: for cleanup of QR file if it gets generated.
	qrPngPath string
}

var pairTok pairTokenState

// startOrRefreshPairToken ensures a valid pairing token exists when not paired.
func startOrRefreshPairToken(ttl time.Duration) (tokenB64 string, tokenID string, exp time.Time) {
	pairTok.mu.Lock()
	defer pairTok.mu.Unlock()

	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	// If active and not expired, reuse.
	if pairTok.active && time.Now().Before(pairTok.expires) && len(pairTok.token) > 0 {
		return base64.RawURLEncoding.EncodeToString(pairTok.token), pairTok.tokenID, pairTok.expires
	}

	// Create fresh token.
	b := make([]byte, 16) // 128-bit
	_, _ = rand.Read(b)

	pairTok.active = true
	pairTok.token = b
	pairTok.tokenID = hex.EncodeToString(b[:4])
	pairTok.expires = time.Now().Add(ttl)

	log.Printf("[pair] pairing token active id=%s expires=%s", pairTok.tokenID, pairTok.expires.Format(time.RFC3339))
	return base64.RawURLEncoding.EncodeToString(pairTok.token), pairTok.tokenID, pairTok.expires
}

// consumePairToken validates token from phone and consumes it (one-time).
func consumePairToken(tokenB64 string) ([]byte, error) {
	pairTok.mu.Lock()
	defer pairTok.mu.Unlock()

	if !pairTok.active || len(pairTok.token) == 0 {
		return nil, fmt.Errorf("pairing not active")
	}
	if time.Now().After(pairTok.expires) {
		pairTok.active = false
		pairTok.token = nil
		return nil, fmt.Errorf("pairing expired")
	}

	got, err := base64.RawURLEncoding.DecodeString(tokenB64)
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding")
	}
	if !constTimeEq(got, pairTok.token) {
		return nil, fmt.Errorf("unauthorized")
	}

	// Consume
	out := make([]byte, len(pairTok.token))
	copy(out, pairTok.token)

	pairTok.active = false
	pairTok.token = nil
	pairTok.tokenID = ""
	pairTok.expires = time.Time{}

	return out, nil
}

func isPairingActive() bool {
	pairTok.mu.Lock()
	defer pairTok.mu.Unlock()
	return pairTok.active && len(pairTok.token) > 0 && time.Now().Before(pairTok.expires)
}

func currentPairExpiry() time.Time {
	pairTok.mu.Lock()
	defer pairTok.mu.Unlock()
	return pairTok.expires
}

func constTimeEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
