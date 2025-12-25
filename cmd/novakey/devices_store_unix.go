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
	"runtime"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	keyringServiceDevices = "novakey"
	keyringAccountDevices = "devices-key"
)

func saveDevicesToDisk(path string, dc devicesConfigFile) error {
	pt, err := json.MarshalIndent(&dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devices json: %w", err)
	}

	key, err := getOrCreateDevicesKey()
	if err != nil {
		// Headless Linux fallback (no unlocked keyring): write plaintext with strict perms.
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

	aad := []byte("NovaKey devices v1")
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
	log.Printf("[pair] created %s/%s in %s", keyringServiceDevices, keyringAccountDevices, runtime.GOOS)
	return key, nil
}

func atomicWrite0600(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
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
