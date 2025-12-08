package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var (
		deviceID = flag.String("device-id", "", "Device ID to pair (e.g. iphone-rob)")
		host     = flag.String("host", "192.168.1.10", "Host/IP the phone should connect to")
		port     = flag.Int("port", 60768, "Port NovaKey service is listening on")
		config   = flag.String("config", "config.yaml", "Path to config.yaml")
	)
	flag.Parse()

	// Change working dir to location of config if it's a relative path.
	configPath, err := filepath.Abs(*config)
	if err != nil {
		fmt.Println("Error resolving config path:", err)
		os.Exit(1)
	}
	if err := os.Chdir(filepath.Dir(configPath)); err != nil {
		fmt.Println("Error changing directory:", err)
		os.Exit(1)
	}

	// Load existing settings
	loadSettings()

	// Ensure map is initialized
	if settings.Devices.PairedDevices == nil {
		settings.Devices.PairedDevices = make(map[string]DeviceConfig)
	}

	// Determine device ID
	id := *deviceID
	if id == "" {
		id, err = generateRandomDeviceID()
		if err != nil {
			fmt.Println("Error generating device ID:", err)
			os.Exit(1)
		}
	}

	// Generate secret
	secret, err := generateDeviceSecret()
	if err != nil {
		fmt.Println("Error generating device secret:", err)
		os.Exit(1)
	}

	// Store in config
	settings.Devices.PairedDevices[id] = DeviceConfig{
		Secret: secret,
	}

	if err := saveSettings("config.yaml"); err != nil {
		fmt.Println("Error saving config.yaml:", err)
		os.Exit(1)
	}

	// Build QR payload
	payload, err := buildPairingPayload(id, secret, *host, *port)
	if err != nil {
		fmt.Println("Error building pairing payload:", err)
		os.Exit(1)
	}

	printPairingInfo(id, secret, payload)
}
