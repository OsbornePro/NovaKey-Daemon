//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// loadDevicesFromDisk on Unix:
// - If file is sealed wrapper => decrypt via keyring key and load
// - Else treat as legacy plaintext JSON and best-effort migrate to sealed
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
		wrap.V == 1 && wrap.Alg == "xchacha20poly1305" &&
		wrap.NonceB64 != "" && wrap.CtB64 != "" {
		return loadDevicesFromSealedWrapper(path, &wrap)
	}

	// Legacy plaintext JSON.
	var dc devicesConfigFile
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("parsing devices file %q: %w", path, err)
	}

	m, err := buildDevicesMap(dc, path)
	if err != nil {
		return nil, err
	}

	// Best-effort migrate plaintext -> sealed (overwrite same path).
	if err := saveDevicesToDisk(path, dc); err == nil {
		log.Printf("[pair] migrated plaintext devices file to sealed format (%s)", path)
	} else {
		log.Printf("[pair] could not migrate devices file to sealed format: %v", err)
	}

	return m, nil
}
