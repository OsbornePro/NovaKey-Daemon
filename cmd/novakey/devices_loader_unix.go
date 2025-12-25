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

func loadDevicesFromDisk(path string) (map[string]deviceState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s not found", ErrNotPaired, path)
		}
		return nil, fmt.Errorf("reading devices file %q: %w", path, err)
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

	// Legacy plaintext JSON (will be migrated by saveDevicesToDisk when pairing completes).
	var dc devicesConfigFile
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("parsing devices file %q: %w", path, err)
	}
	return buildDevicesMap(dc, path)
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
