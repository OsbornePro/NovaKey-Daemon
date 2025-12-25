//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func loadDevicesFromDisk(path string) (map[string]deviceState, error) {
	// If storing devices.json as a DPAPI wrapper, load it here:
	// - Read wrapper JSON
	// - base64 decode blob
	// - CryptUnprotectData
	// - Unmarshal devicesConfigFile
	// - buildDevicesMap(dc, path)

	return loadDevicesFromFileWindows(path) // <-- call DPAPI loader
}

func saveDevicesToDisk(path string, dc devicesConfigFile) error {
	// Save devices file as DPAPI wrapper.
	// DPAPI save path should:
	// - MarshalIndent(dc)
	// - CryptProtectData
	// - base64 encode blob
	// - write wrapper JSON to path
	pt, err := json.MarshalIndent(&dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devices json: %w", err)
	}

	wrap, err := dpapiProtectToWrapper(pt) 
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(&wrap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dpapi wrapper: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

