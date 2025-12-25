//go:build windows

package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type dpapiFile struct {
	V        int    `json:"v"`
	DPAPIB64 string `json:"dpapi_b64"`
}

func loadDevicesFromDisk(path string) (map[string]deviceState, error) {
	prefer := preferDPAPIPath(path)

	m, err := loadDevicesFromDPAPIFile(prefer)
	if err == nil {
		return m, nil
	}

	// If DPAPI missing: try plaintext migration.
	if errors.Is(err, ErrNotPaired) {
		plain := path
		if strings.HasSuffix(strings.ToLower(plain), ".dpapi.json") {
			plain = strings.TrimSuffix(plain, ".dpapi.json") + ".json"
		}

		pm, perr := loadDevicesFromPlainJSON(plain)
		if perr == nil {
			_ = saveDevicesToDisk(prefer, devicesConfigFromMap(pm))
			_ = os.Remove(plain)
			return loadDevicesFromDPAPIFile(prefer)
		}
		return nil, err
	}

	return nil, err
}

func saveDevicesToDisk(path string, dc devicesConfigFile) error {
	pt, err := json.MarshalIndent(&dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devices json: %w", err)
	}

	ct, err := dpapiProtect(pt)
	if err != nil {
		return err
	}

	wrap := dpapiFile{V: 1, DPAPIB64: dpapiEncode(ct)}
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

func loadDevicesFromPlainJSON(path string) (map[string]deviceState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s not found", ErrNotPaired, path)
		}
		return nil, fmt.Errorf("reading devices file %q: %w", path, err)
	}
	var dc devicesConfigFile
	if err := json.Unmarshal(b, &dc); err != nil {
		return nil, fmt.Errorf("parsing devices file %q: %w", path, err)
	}
	return buildDevicesMap(dc, path)
}

func devicesConfigFromMap(m map[string]deviceState) devicesConfigFile {
	out := devicesConfigFile{Devices: make([]deviceConfig, 0, len(m))}
	for _, st := range m {
		out.Devices = append(out.Devices, deviceConfig{
			ID:     st.id,
			KeyHex: hex.EncodeToString(st.staticKey),
		})
	}
	return out
}
