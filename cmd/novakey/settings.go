package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	Version int `yaml:"version"`

	Network struct {
		ListenAddress string `yaml:"listen_address"`
		ListenPort    int    `yaml:"listen_port"`
		IPv6          bool   `yaml:"ipv6"`
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
		    BundleIDs    []string `yaml:"bundle_ids"`
		    Executables []string `yaml:"executables"`
	    } `yaml:"darwin"`
    } `yaml:"allowlist"`

	Devices struct {
		RequireKnownDevice bool            `yaml:"require_known_device"`
		PairedDevices      map[string]bool `yaml:"paired_devices"`
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

		// Security defaults
		settings.Security.RequireArming = true
		settings.Security.EnforceAllowlist = true

		// Arming defaults
		settings.Arming.AllowHotkey = true
		settings.Arming.AllowCLI = true
		settings.Arming.AutoDisarm = true

		// Device defaults
		settings.Devices.RequireKnownDevice = false
		settings.Devices.PairedDevices = make(map[string]bool)

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
		settings.Devices.PairedDevices = make(map[string]bool)
	}
}
