package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ServerConfig struct {
	ListenAddr          string `json:"listen_addr"`
	MaxPayloadLen       int    `json:"max_payload_len"`
	MaxRequestsPerMin   int    `json:"max_requests_per_min"`
	DevicesFile         string `json:"devices_file"`
}

var cfg ServerConfig

const defaultConfigFile = "server_config.json"

func loadConfig() error {
	path := os.Getenv("NOVAKEY_CONFIG_FILE")
	if path == "" {
		path = defaultConfigFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config file %q: %w", path, err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Defaults / sanity
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:60768"
	}
	if cfg.MaxPayloadLen <= 0 || cfg.MaxPayloadLen > 65535 {
		cfg.MaxPayloadLen = 4096
	}
	if cfg.MaxRequestsPerMin <= 0 {
		cfg.MaxRequestsPerMin = 60
	}
	if cfg.DevicesFile == "" {
		cfg.DevicesFile = "devices.json"
	}

	absPath, _ := filepath.Abs(path)
	fmt.Printf("Loaded server config from %s\n", absPath)
	return nil
}

