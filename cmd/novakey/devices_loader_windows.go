//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadDevicesFromDisk on Windows expects cfg.DevicesFile to point to the DPAPI-wrapped JSON.
//
// Migration behavior:
// - Tries DPAPI file first (prefers *.dpapi.json sibling even if cfg points to devices.json)
// - If DPAPI file missing, tries plaintext devices.json, then writes DPAPI file and deletes plaintext.
func loadDevicesFromDisk(path string) (map[string]deviceState, error) {
	prefer := preferDPAPIPath(path)

	// 1) Try DPAPI-wrapped file first
	m, err := loadDevicesFromDPAPIFile(prefer)
	if err == nil {
		return m, nil
	}

	// 2) If "not paired" (missing/empty), try plaintext migration
	if errors.Is(err, ErrNotPaired) {
		plain := derivePlainPath(path)

		dc, pm, perr := loadDevicesConfigFromPlainJSON(plain)
		if perr == nil {
			// write DPAPI wrapper to preferred path
			if werr := saveDevicesToDisk(prefer, dc); werr == nil {
				_ = os.Remove(plain)
			}
			return pm, nil
		}

		// If plaintext also missing/empty, stay "not paired"
		if errors.Is(perr, ErrNotPaired) {
			return nil, fmt.Errorf("%w: %s not found", ErrNotPaired, prefer)
		}

		// Otherwise return original DPAPI error (more relevant)
		return nil, err
	}

	return nil, err
}

func preferDPAPIPath(path string) string {
	lo := strings.ToLower(path)
	if strings.HasSuffix(lo, ".dpapi.json") {
		return path
	}
	if strings.HasSuffix(lo, ".json") {
		return strings.TrimSuffix(path, filepath.Ext(path)) + ".dpapi.json"
	}
	return path + ".dpapi.json"
}

func derivePlainPath(path string) string {
	plain := path
	if strings.HasSuffix(strings.ToLower(plain), ".dpapi.json") {
		plain = strings.TrimSuffix(plain, ".dpapi.json") + ".json"
	}
	return plain
}

func loadDevicesFromDPAPIFile(path string) (map[string]deviceState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s not found", ErrNotPaired, path)
		}
		return nil, fmt.Errorf("reading dpapi devices file %q: %w", path, err)
	}

	var wrap dpapiFile
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, fmt.Errorf("parse dpapi wrapper %q: %w", path, err)
	}
	if wrap.V != 1 || strings.TrimSpace(wrap.DPAPIB64) == "" {
		return nil, fmt.Errorf("invalid dpapi wrapper %q", path)
	}

	ct, err := dpapiDecode(wrap.DPAPIB64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode dpapi blob: %w", err)
	}
	pt, err := dpapiUnprotect(ct)
	if err != nil {
		return nil, fmt.Errorf("dpapi unprotect: %w", err)
	}

	var dc devicesConfigFile
	if err := json.Unmarshal(pt, &dc); err != nil {
		return nil, fmt.Errorf("parse devices json inside dpapi: %w", err)
	}
	return buildDevicesMap(dc, path)
}

// loadDevicesConfigFromPlainJSON returns both the parsed config (for migration) and the validated map.
func loadDevicesConfigFromPlainJSON(path string) (devicesConfigFile, map[string]deviceState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return devicesConfigFile{}, nil, fmt.Errorf("%w: %s not found", ErrNotPaired, path)
		}
		return devicesConfigFile{}, nil, fmt.Errorf("reading devices file %q: %w", path, err)
	}

	var dc devicesConfigFile
	if err := json.Unmarshal(b, &dc); err != nil {
		return devicesConfigFile{}, nil, fmt.Errorf("parsing devices file %q: %w", path, err)
	}

	m, err := buildDevicesMap(dc, path)
	if err != nil {
		return devicesConfigFile{}, nil, err
	}
	return dc, m, nil
}
