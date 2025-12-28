// cmd/novakey/devices_save_unix.go
//go:build !windows

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
)

// saveDevicesToDisk writes devices as a sealed JSON file on non-Windows.
// It uses an OS keyring-stored key when available; if keyring is unavailable,
// it can fall back to plaintext JSON with strict perms (0600) unless
// cfg.RequireSealedDeviceStore=true, in which case it fails closed.
func saveDevicesToDisk(path string, dc devicesConfigFile) error {
	pt, err := json.MarshalIndent(&dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devices json: %w", err)
	}

	key, err := getOrCreateDevicesKey()
	if err != nil {
		if cfg.RequireSealedDeviceStore {
			return fmt.Errorf("%w: require_sealed_device_store=true but keyring is unavailable: %v",
				ErrDevicesUnavailable, err)
		}
        // Headless Linux/macOS edge cases: if keyring can’t be used, plaintext devices store may be required (explicit opt-in).
		log.Printf("[warn] keyring unavailable (%v); falling back to plaintext with 0600", err)
		return atomicWrite0600(path, pt)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return fmt.Errorf("NewX: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("rand nonce: %w", err)
	}

	aad := []byte(devicesSealedAAD)
	ct := aead.Seal(nil, nonce, pt, aad)

	wrap := sealedDevicesFileV1{
		V:        1,
		Alg:      "xchacha20poly1305",
		NonceB64: base64.StdEncoding.EncodeToString(nonce),
		CtB64:    base64.StdEncoding.EncodeToString(ct),
	}

	out, err := json.MarshalIndent(&wrap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sealed wrapper: %w", err)
	}

	return atomicWrite0600(path, out)
}

func atomicWrite0600(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}
