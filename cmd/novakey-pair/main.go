package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

    "github.com/OsbornePro/NovaKey/internal/config"
    "github.com/OsbornePro/NovaKey/internal/pairing"
)

func main() {
	var (
		deviceID = flag.String("device-id", "", "Device ID to pair (e.g. iphone-rob)")
		host     = flag.String("host", "127.0.0.1", "Host/IP the phone should connect to")
		port     = flag.Int("port", 60768, "Port NovaKey service is listening on")
		cfgPath  = flag.String("config", "config.yaml", "Path to config.yaml")
	)
	flag.Parse()

	// Resolve config path & chdir
	absCfg, err := filepath.Abs(*cfgPath)
	if err != nil {
		die(err)
	}
	if err := os.Chdir(filepath.Dir(absCfg)); err != nil {
		die(err)
	}

	// Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		die(err)
	}

	if cfg.Devices.PairedDevices == nil {
		cfg.Devices.PairedDevices = make(map[string]config.DeviceConfig)
	}

	// Device ID
	id := *deviceID
	if id == "" {
		id = pairing.GenerateDeviceID()

	}

	// Device secret
	secret, err := pairing.GenerateDeviceSecret()
	if err != nil {
		die(err)
	}

	// Store (hashed)
	cfg.Devices.PairedDevices[id] = config.DeviceConfig{
		SecretHash: config.HashSecret(secret),
	}

	if err := config.Save("config.yaml", cfg); err != nil {
		die(err)
	}

	// Build QR payload
	payload, err := pairing.BuildPairingPayload(
		id,
		secret,
		*host,
		*port,
	)
	if err != nil {
		die(err)
	}

	pairing.PrintPairingInfo(id, secret, payload)
}

func die(err error) {
	fmt.Println("x Error:", err)
	os.Exit(1)
}
