package main

import (
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type DeviceConfig struct {
	Secret   string `yaml:"secret"`
	CanAdmin bool   `yaml:"can_admin"` // ✅ phone allowed to reload config
}

type Settings struct {
	Version int `yaml:"version"`

	Network struct {
		ListenAddress string `yaml:"listen_address"`
		ListenPort    int    `yaml:"listen_port"`
		Mode          string `yaml:"mode"`
	} `yaml:"network"`

	Security struct {
		RequireArming    bool `yaml:"require_arming"`
		EnforceAllowlist bool `yaml:"enforce_allowlist"`
	} `yaml:"security"`

	Arming struct {
		AllowHotkey    bool `yaml:"allow_hotkey"`
		AllowCLI       bool `yaml:"allow_cli"`
		AutoDisarm     bool `yaml:"auto_disarm"`
		TimeoutSeconds int  `yaml:"timeout_seconds"`

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
			BundleIDs   []string `yaml:"bundle_ids"`
			Executables []string `yaml:"executables"`
		} `yaml:"darwin"`
	} `yaml:"allowlist"`

	Devices struct {
		RequireKnownDevice bool                    `yaml:"require_known_device"`
		AutoRegister       bool                    `yaml:"auto_register"`
		PairedDevices      map[string]DeviceConfig `yaml:"paired_devices"`
	} `yaml:"devices"`

	Control struct {
		Enabled       bool   `yaml:"enabled"`
		ListenAddress string `yaml:"listen_address"`
		ListenPort    int    `yaml:"listen_port"`
		Token         string `yaml:"token"`
	} `yaml:"control"`
}

var (
	settings   Settings
	settingsMu sync.RWMutex
)

func loadSettingsFromDisk() (Settings, error) {
	var s Settings

	f, err := os.Open("config.yaml")
	if err != nil {
		return s, err
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&s)
	if err != nil {
		return s, err
	}

	if s.Devices.PairedDevices == nil {
		s.Devices.PairedDevices = make(map[string]DeviceConfig)
	}

	return s, nil
}

func loadSettings() {
	s, err := loadSettingsFromDisk()
	if err != nil {
		LogError("Failed to load config.yaml", err)
		os.Exit(1)
	}

	settingsMu.Lock()
	settings = s
	settingsMu.Unlock()
}

func reloadSettings() error {
	s, err := loadSettingsFromDisk()
	if err != nil {
		return err
	}

	settingsMu.Lock()
	settings = s
	settingsMu.Unlock()

	return nil
}

func saveSettings(path string) error {
	settingsMu.RLock()
	defer settingsMu.RUnlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	return enc.Encode(&settings)
}
