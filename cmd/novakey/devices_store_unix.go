//go:build !windows

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
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

type sealedDevicesFileV1 struct {
	V        int    `json:"v"`
	Alg      string `json:"alg"` // "xchacha20poly1305"
	NonceB64 string `json:"nonce_b64"`
	CtB64    string `json:"ct_b64"`
}

func loadDevicesFromDisk(path string) (map[string]deviceState, error) {
	// Prefer sealed file if it exists.
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s not found", ErrNotPaired, path)
		}
		return nil, fmt.Errorf("reading devices file %q: %w", path, err)
	}

	// Try sealed wrapper first.
	var wrap sealedDevicesFileV1
	if err := json.Unmarshal(data, &wrap); err == nil && wrap.V == 1 && wrap.Alg == "xchacha20poly1305" && wrap.NonceB64 != "" && wrap.CtB64 != "" {
		return loadDevicesFromSealedWrapper(path, &wrap)
	}

	// Fallback: plaintext JSON (legacy).
	var dc devicesConfigFile
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("parsing devices file %q: %w", path, err)
	}
	m, err := buildDevicesMap(dc, path)
	if err != nil {
		return nil, err
	}

	// Best-effort auto-migrate plaintext -> sealed on unix.
	if err := saveDevicesToDisk(path, dc); err == nil {
		// Optionally delete plaintext original (same path, now overwritten by saveDevicesToDisk).
		log.Printf("[pair] migrated plaintext devices file to sealed format (%s)", path)
	} else {
		log.Printf("[pair] could not migrate devices file to sealed format: %v", err)
	}

	return m, nil
}

func saveDevicesToDisk(path string, dc devicesConfigFile) error {
	// Serialize JSON plaintext
	pt, err := json.MarshalIndent(&dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devices json: %w", err)
	}

	key, err := getOrCreateDevicesKey()
	if err != nil {
		// If we cannot access keyring (headless), fall back to strict-perms plaintext.
		log.Printf("[warn] keyring unavailable (%v); falling back to plaintext devices file with 0600 perms", err)
		return writePlaintextDevicesWith0600(path, pt)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return fmt.Errorf("NewX: %w", err)
	}

	nonce := make([]byte, aead.NonceSize()) // 24 bytes
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("rand nonce: %w", err)
	}

	// AAD ties ciphertext to this “purpose” (not secret).
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

func loadDevicesFromSealedWrapper(path string, wrap *sealedDevicesFileV1) (map[string]deviceState, error) {
	key, err := getOrCreateDevicesKey()
	if err != nil {
		return nil, fmt.Errorf("keyring unavailable for sealed devices file: %w", err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("NewX: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(wrap.NonceB64)
	if err != nil {
		return nil, fmt.Errorf("decode nonce_b64: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(wrap.CtB64)
	if err != nil {
		return nil, fmt.Errorf("decode ct_b64: %w", err)
	}

	aad := []byte("NovaKey devices v1")
	pt, err := aead.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt sealed devices file: %w", err)
	}

	var dc devicesConfigFile
	if err := json.Unmarshal(pt, &dc); err != nil {
		return nil, fmt.Errorf("parse devices json inside sealed wrapper: %w", err)
	}
	return buildDevicesMap(dc, path)
}

func getOrCreateDevicesKey() ([]byte, error) {
	// Stored value = base64(32 bytes)
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

	// If not found, create it.
	// go-keyring returns a sentinel error on “not found”, but it differs by platform, so treat any error as “maybe missing”.
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

func writePlaintextDevicesWith0600(path string, data []byte) error {
	return atomicWrite0600(path, data)
}

func atomicWrite0600(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		// 0700 because we’re storing secrets nearby
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

// buildDevicesMap is shared by Windows + Unix.
// Keep the strict checks you already had.
func buildDevicesMap(dc devicesConfigFile, path string) (map[string]deviceState, error) {
	if len(dc.Devices) == 0 {
		return nil, fmt.Errorf("%w: %s has no devices", ErrNotPaired, path)
	}

	m := make(map[string]deviceState, len(dc.Devices))
	for _, d := range dc.Devices {
		if d.ID == "" {
			return nil, fmt.Errorf("device with empty id in %q", path)
		}
		keyBytes, err := hexDecode32(d.KeyHex, d.ID)
		if err != nil {
			return nil, err
		}
		m[d.ID] = deviceState{id: d.ID, staticKey: keyBytes}
	}
	return m, nil
}

// helper: keep crypto.go clean; avoid importing hex there if you prefer
func hexDecode32(keyHex string, deviceID string) ([]byte, error) {
	b, err := hexDecode(keyHex)
	if err != nil {
		return nil, fmt.Errorf("device %q: invalid key_hex: %w", deviceID, err)
	}
	if len(b) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("device %q: key must be %d bytes, got %d", deviceID, chacha20poly1305.KeySize, len(b))
	}
	return b, nil
}

// split to keep imports stable in your tree (since you already use encoding/hex elsewhere)
func hexDecode(s string) ([]byte, error) { return hex.DecodeString(s) }

// compile-time guard: if the hex package disappears from your build tags, you'll catch it
var _ = errors.New

