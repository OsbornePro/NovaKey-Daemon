package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	Version  int `yaml:"version"`
	Devices  DeviceSettings
	Security SecuritySettings
}

type DeviceSettings struct {
	RequireKnownDevice bool                    `yaml:"require_known_device"`
	PairedDevices     map[string]DeviceConfig `yaml:"paired_devices"`
}

type DeviceConfig struct {
	SecretHash string `yaml:"secret_hash"`
}

type SecuritySettings struct {
	RequireArming    bool `yaml:"require_arming"`
	EnforceAllowlist bool `yaml:"enforce_allowlist"`
}

// Load reads config.yaml
func Load(path string) (*Settings, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Settings
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes config.yaml
func Save(path string, s *Settings) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

// HashSecret hashes a device secret for storage
func HashSecret(secret []byte) string {
	sum := sha256.Sum256(secret)
	return hex.EncodeToString(sum[:])
}
