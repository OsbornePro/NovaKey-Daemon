//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// saveDevicesToDisk writes devicesConfigFile as a DPAPI-wrapped JSON file (atomic write).
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
