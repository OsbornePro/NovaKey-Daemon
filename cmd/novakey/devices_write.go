// cmd/novakey/devices_write.go
package main

import (
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// writeDevicesFile upserts (deviceID, deviceKeyHex) into the persisted device store.
// It is used by pairing_proto.go.
func writeDevicesFile(path string, deviceID string, deviceKeyHex string) error {
	if deviceID == "" {
		return fmt.Errorf("writeDevicesFile: empty deviceID")
	}
	k, err := hex.DecodeString(deviceKeyHex)
	if err != nil {
		return fmt.Errorf("writeDevicesFile: invalid device_key_hex: %w", err)
	}
	if len(k) != chacha20poly1305.KeySize {
		return fmt.Errorf("writeDevicesFile: device key must be %d bytes, got %d", chacha20poly1305.KeySize, len(k))
	}

	// Load existing store (or start fresh if not paired yet).
	existing, err := loadDevicesFromDisk(path)
	if err != nil {
		if errors.Is(err, ErrNotPaired) {
			existing = make(map[string]deviceState)
		} else {
			return err
		}
	}

	// Upsert
	existing[deviceID] = deviceState{id: deviceID, staticKey: k}

	// Convert to devicesConfigFile and persist via platform-specific saveDevicesToDisk.
	dc := devicesConfigFile{Devices: make([]deviceConfig, 0, len(existing))}
	for _, st := range existing {
		dc.Devices = append(dc.Devices, deviceConfig{
			ID:     st.id,
			KeyHex: hex.EncodeToString(st.staticKey),
		})
	}

	return saveDevicesToDisk(path, dc)
}
