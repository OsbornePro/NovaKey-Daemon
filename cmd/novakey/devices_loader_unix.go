//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func loadDevicesFromDisk(path string) (map[string]deviceState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s not found", ErrNotPaired, path)
		}
		return nil, fmt.Errorf("reading devices file %q: %w", path, err)
	}

	var dc devicesConfigFile
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("parsing devices file %q: %w", path, err)
	}
	return buildDevicesMap(dc, path)
}
