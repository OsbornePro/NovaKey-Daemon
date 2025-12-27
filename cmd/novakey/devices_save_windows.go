// cmd/novakey/devices_save_windows.go
//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// saveDevicesToDisk persists the devicesConfigFile to disk on Windows.
// This is the counterpart to loadDevicesFromDisk(path) implemented in *_windows.go.
//
// Format: JSON (same shape as devicesConfigFile).
// Write strategy: atomic-ish (write temp + replace).
func saveDevicesToDisk(path string, dc devicesConfigFile) error {
	if path == "" {
		return fmt.Errorf("saveDevicesToDisk: empty path")
	}

	// Ensure parent dir exists.
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("saveDevicesToDisk: mkdir %s: %w", dir, err)
		}
	}

	b, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		return fmt.Errorf("saveDevicesToDisk: marshal: %w", err)
	}

	tmp := path + ".tmp"

	// Best-effort cleanup.
	_ = os.Remove(tmp)

	// Write tmp file first.
	// Note: perms are best-effort on Windows.
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("saveDevicesToDisk: write tmp: %w", err)
	}

	// On Windows, os.Rename won't reliably overwrite an existing file.
	_ = os.Remove(path)

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("saveDevicesToDisk: rename: %w", err)
	}

	return nil
}

