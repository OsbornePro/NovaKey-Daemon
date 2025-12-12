// cmd/nvpair/main.go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
)

type deviceConfig struct {
	ID     string `json:"id"`
	KeyHex string `json:"key_hex"`
}

type devicesConfigFile struct {
	Devices []deviceConfig `json:"devices"`
}

var (
	devicesFileFlag = flag.String("devices-file", "devices.json", "path to devices.json")
	deviceIDFlag    = flag.String("id", "", "device ID to add or update (required)")
	forceFlag       = flag.Bool("force", false, "overwrite existing device with same ID")
)

func main() {
	flag.Parse()

	if *deviceIDFlag == "" {
		fmt.Fprintln(os.Stderr, "ERROR: -id is required (device ID)")
		flag.Usage()
		os.Exit(1)
	}

	path := *devicesFileFlag
	absPath, _ := filepath.Abs(path)

	cfg, err := loadDevices(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("devices file %s does not exist, creating new one\n", absPath)
			cfg = &devicesConfigFile{Devices: []deviceConfig{}}
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: loading devices file %s: %v\n", absPath, err)
			os.Exit(1)
		}
	}

	// Generate a new random key
	keyBytes := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(keyBytes); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: rand.Read key: %v\n", err)
		os.Exit(1)
	}
	keyHex := hex.EncodeToString(keyBytes)

	// Add or update device
	existingIdx := -1
	for i, d := range cfg.Devices {
		if d.ID == *deviceIDFlag {
			existingIdx = i
			break
		}
	}

	if existingIdx >= 0 && !*forceFlag {
		fmt.Fprintf(os.Stderr, "ERROR: device ID %q already exists in %s (use -force to overwrite)\n",
			*deviceIDFlag, absPath)
		os.Exit(1)
	}

	if existingIdx >= 0 && *forceFlag {
		cfg.Devices[existingIdx].KeyHex = keyHex
		fmt.Printf("Updated existing device %q in %s\n", *deviceIDFlag, absPath)
	} else if existingIdx == -1 {
		cfg.Devices = append(cfg.Devices, deviceConfig{
			ID:     *deviceIDFlag,
			KeyHex: keyHex,
		})
		fmt.Printf("Added new device %q to %s\n", *deviceIDFlag, absPath)
	}

	if err := saveDevices(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: saving devices file %s: %v\n", absPath, err)
		os.Exit(1)
	}

	fmt.Println("------------------------------------------------------------")
	fmt.Println(" Pairing info")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("Device ID : %s\n", *deviceIDFlag)
	fmt.Printf("Key (hex) : %s\n", keyHex)
	fmt.Println()
	fmt.Println("Use these values with nvclient or your real client, e.g.:")
	fmt.Printf("  nvclient -addr 127.0.0.1:60768 -device-id %q -key-hex %s -password \"...\"\n",
		*deviceIDFlag, keyHex)
}

func loadDevices(path string) (*devicesConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg devicesConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveDevices(path string, cfg *devicesConfigFile) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

