//go:build windows
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func writeDevicesFile(path string, deviceID string, deviceKeyHex string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("devices path empty")
	}

	inner := devicesConfigFile{
		Devices: []deviceConfig{{ID: deviceID, KeyHex: deviceKeyHex}},
	}
	innerJSON, err := json.Marshal(&inner)
	if err != nil {
		return err
	}

	ct, err := dpapiProtect(innerJSON)
	if err != nil {
		return err
	}

	wrap := dpapiFile{V: 1, DPAPIB64: dpapiEncode(ct)}
	outJSON, err := json.MarshalIndent(&wrap, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, outJSON, 0600); err != nil { // mode bits mostly ignored; ACL matters
		return err
	}
	return os.Rename(tmp, path)
}
