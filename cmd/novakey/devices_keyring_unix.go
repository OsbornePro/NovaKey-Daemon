// cmd/novakey/devices_keyring_unix.go
//go:build !windows

package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"runtime"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	keyringServiceDevices = "novakey"
	keyringAccountDevices = "devices-key"
)

// getOrCreateDevicesKey returns a 32-byte key used to seal/unseal the devices store on non-Windows.
// Stored in the OS keyring when available.
func getOrCreateDevicesKey() ([]byte, error) {
	s, err := keyring.Get(keyringServiceDevices, keyringAccountDevices)
	if err == nil && s != "" {
		b, derr := base64.StdEncoding.DecodeString(s)
		if derr != nil {
			return nil, fmt.Errorf("keyring key invalid base64: %w", derr)
		}
		if len(b) != chacha20poly1305.KeySize {
			return nil, fmt.Errorf("keyring key wrong length: got %d want %d", len(b), chacha20poly1305.KeySize)
		}
		return b, nil
	}

	key := make([]byte, chacha20poly1305.KeySize)
	if _, rerr := rand.Read(key); rerr != nil {
		return nil, fmt.Errorf("rand devices key: %w", rerr)
	}

	if serr := keyring.Set(keyringServiceDevices, keyringAccountDevices, base64.StdEncoding.EncodeToString(key)); serr != nil {
		return nil, serr
	}
	log.Printf("[pair] created keyring item %s/%s on %s", keyringServiceDevices, keyringAccountDevices, runtime.GOOS)
	return key, nil
}
