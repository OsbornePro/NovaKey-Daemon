// cmd/novakey/config.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServerConfig struct {
	ListenAddr        string `json:"listen_addr"`
	MaxPayloadLen     int    `json:"max_payload_len"`
	MaxRequestsPerMin int    `json:"max_requests_per_min"`
	DevicesFile       string `json:"devices_file"`
	ServerKeysFile    string `json:"server_keys_file"`
}

var cfg ServerConfig

const defaultConfigFile = "server_config.json"

func loadConfig() error {
	const path = "server_config.json"

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:60768"
	}
	if cfg.MaxPayloadLen == 0 {
		cfg.MaxPayloadLen = 4096
	}
	if cfg.MaxRequestsPerMin == 0 {
		cfg.MaxRequestsPerMin = 60
	}
	if cfg.DevicesFile == "" {
		cfg.DevicesFile = "devices.json"
	}
	if cfg.ServerKeysFile == "" {
		cfg.ServerKeysFile = "server_keys.json"
	}

	return nil
}
