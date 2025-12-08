package main

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type DeviceConfig struct {
	Secret string `yaml:"secret"`
}

type Settings struct {
	Version int `yaml:"version"`

	Network struct {
		ListenAddress string `yaml:"listen_address"`
		ListenPort    int    `yaml:"listen_port"`
		Mode          string `yaml:"mode"` // ipv4 | ipv6 | dual
	} `yaml:"network"`

	Security struct {
		RequireArming    bool `yaml:"require_arming"`
		EnforceAllowlist bool `yaml:"enforce_allowlist"`
	} `yaml:"security"`

	Arming struct {
		AllowHotkey bool `yaml:"allow_hotkey"`
		AllowCLI    bool `yaml:"allow_cli"`
		AutoDisarm  bool `yaml:"auto_disarm"`

		Hotkey struct {
			Ctrl  bool   `yaml:"ctrl"`
			Alt   bool   `yaml:"alt"`
			Shift bool   `yaml:"shift"`
			Win   bool   `yaml:"win"`
			Key   string `yaml:"key"`
		} `yaml:"hotkey"`
	} `yaml:"arming"`

	Allowlist struct {
		Windows struct {
			Browsers         []string `yaml:"browsers"`
			PasswordManagers []string `yaml:"password_managers"`
		} `yaml:"windows"`

		Darwin struct {
			BundleIDs  []string `yaml:"bundle_ids"`
			Executables []string `yaml:"executables"`
		} `yaml:"darwin"`
	} `yaml:"allowlist"`

	Devices struct {
		RequireKnownDevice bool                    `yaml:"require_known_device"`
		PairedDevices      map[string]DeviceConfig `yaml:"paired_devices"`
	} `yaml:"devices"`

	Control struct {
		Enabled       bool   `yaml:"enabled"`
		ListenAddress string `yaml:"listen_address"`
		ListenPort    int    `yaml:"listen_port"`
		Token         string `yaml:"token"`
	} `yaml:"control"`
}

var settings Settings

func loadSettings() {
	f, err := os.Open("config.yaml")
	if err != nil {
		LogInfo("No config.yaml found; using defaults")

		settings.Version = 1

		// Network defaults
		settings.Network.ListenAddress = "0.0.0.0"
		settings.Network.ListenPort = 60768
		settings.Network.Mode = "ipv4"

		// Security defaults
		settings.Security.RequireArming = true
		settings.Security.EnforceAllowlist = true

		// Arming defaults
		settings.Arming.AllowHotkey = true
		settings.Arming.AllowCLI = true
		settings.Arming.AutoDisarm = true

		// Device defaults
		settings.Devices.RequireKnownDevice = false
		settings.Devices.PairedDevices = make(map[string]DeviceConfig)

		// Control listener defaults
		settings.Control.Enabled = true
		settings.Control.ListenAddress = "127.0.0.1"
		settings.Control.ListenPort = 60769
		settings.Control.Token = ""

		return
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&settings); err != nil {
		LogError("Failed to parse config.yaml", err)
	}

	if settings.Devices.PairedDevices == nil {
		settings.Devices.PairedDevices = make(map[string]DeviceConfig)
	}

	// Normalize and validate network mode
	mode := strings.ToLower(strings.TrimSpace(settings.Network.Mode))
	switch mode {
	case "", "ipv4":
		settings.Network.Mode = "ipv4"
	case "ipv6", "dual":
		settings.Network.Mode = mode
	default:
		LogError("Invalid network.mode in config.yaml; defaulting to ipv4", nil)
		settings.Network.Mode = "ipv4"
	}
}

func saveSettings(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(&settings); err != nil {
		return err
	}
	return enc.Close()
}
