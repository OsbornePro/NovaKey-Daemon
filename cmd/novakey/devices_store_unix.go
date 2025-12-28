// cmd/novakey/devices_store_unix.go
//go:build !windows

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/crypto/chacha20poly1305"
)

type sealedDevicesFileV1 struct {
	V        int    `json:"v"`
	Alg      string `json:"alg"` // "xchacha20poly1305"
	NonceB64 string `json:"nonce_b64"`
	CtB64    string `json:"ct_b64"`
}

const devicesSealedAAD = "NovaKey devices v1"

func loadDevicesFromDisk(path string) (map[string]deviceState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s not found", ErrNotPaired, path)
		}
		return nil, fmt.Errorf("%w: reading devices file %q: %v", ErrDevicesUnavailable, path, err)
	}

	// Try sealed wrapper first.
	var wrap sealedDevicesFileV1
	if err := json.Unmarshal(data, &wrap); err == nil &&
		wrap.V == 1 &&
		wrap.Alg == "xchacha20poly1305" &&
		wrap.NonceB64 != "" &&
		wrap.CtB64 != "" {
		return loadDevicesFromSealedWrapper(path, &wrap)
	}

	// If the file is not a sealed wrapper, it's plaintext JSON.
	// If require_sealed_device_store is enabled, fail closed.
	if cfg.RequireSealedDeviceStore {
		return nil, fmt.Errorf("%w: require_sealed_device_store=true but devices file is not sealed (plaintext): %s",
			ErrDevicesUnavailable, path)
	}

	// Plaintext JSON path:
	var dc devicesConfigFile
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("%w: parsing devices file %q: %v", ErrDevicesUnavailable, path, err)
	}

	if len(dc.Devices) == 0 {
		return nil, fmt.Errorf("%w: %s has no devices", ErrNotPaired, path)
	}

	return buildDevicesMap(dc, path)
}

func loadDevicesFromSealedWrapper(path string, wrap *sealedDevicesFileV1) (map[string]deviceState, error) {
	key, err := getOrCreateDevicesKey()
	if err != nil {
		return nil, fmt.Errorf("%w: keyring unavailable for sealed devices file: %v", ErrDevicesUnavailable, err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("%w: NewX: %v", ErrDevicesUnavailable, err)
	}

	nonce, err := base64.StdEncoding.DecodeString(wrap.NonceB64)
	if err != nil {
		return nil, fmt.Errorf("%w: decode nonce_b64: %v", ErrDevicesUnavailable, err)
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("%w: invalid nonce length: got %d want %d", ErrDevicesUnavailable, len(nonce), aead.NonceSize())
	}

	ct, err := base64.StdEncoding.DecodeString(wrap.CtB64)
	if err != nil {
		return nil, fmt.Errorf("%w: decode ct_b64: %v", ErrDevicesUnavailable, err)
	}
	if len(ct) < aead.Overhead() {
		return nil, fmt.Errorf("%w: ciphertext too short: got %d need at least %d", ErrDevicesUnavailable, len(ct), aead.Overhead())
	}

	aad := []byte(devicesSealedAAD)
	pt, err := aead.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, fmt.Errorf("%w: decrypt sealed devices file: %v", ErrDevicesUnavailable, err)
	}

	var dc devicesConfigFile
	if err := json.Unmarshal(pt, &dc); err != nil {
		return nil, fmt.Errorf("%w: parse devices json inside sealed wrapper: %v", ErrDevicesUnavailable, err)
	}
	if len(dc.Devices) == 0 {
		return nil, fmt.Errorf("%w: %s has no devices", ErrNotPaired, path)
	}
	return buildDevicesMap(dc, path)
}
