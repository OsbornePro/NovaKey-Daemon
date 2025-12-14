// cmd/novakey/config.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	ListenAddr        string `json:"listen_addr" yaml:"listen_addr"`
	MaxPayloadLen     int    `json:"max_payload_len" yaml:"max_payload_len"`
	MaxRequestsPerMin int    `json:"max_requests_per_min" yaml:"max_requests_per_min"`
	DevicesFile       string `json:"devices_file" yaml:"devices_file"`
	ServerKeysFile    string `json:"server_keys_file" yaml:"server_keys_file"`

	// Arm gate (OFF by default)
	ArmEnabled bool `json:"arm_enabled" yaml:"arm_enabled"`
	ArmDurationMs int `json:"arm_duration_ms" yaml:"arm_duration_ms"`

	// Pointer = allows "default true but allow false"
	ArmConsumeOnInject *bool `json:"arm_consume_on_inject" yaml:"arm_consume_on_inject"`

	// Pointer = allows "default true but allow false"
	AllowClipboardWhenDisarmed *bool `json:"allow_clipboard_when_disarmed" yaml:"allow_clipboard_when_disarmed"`

	// Local-only arming endpoint (OFF by default)
	ArmAPIEnabled  bool   `json:"arm_api_enabled" yaml:"arm_api_enabled"`
	ArmListenAddr  string `json:"arm_listen_addr" yaml:"arm_listen_addr"`
	ArmTokenFile   string `json:"arm_token_file" yaml:"arm_token_file"`
	ArmTokenHeader string `json:"arm_token_header" yaml:"arm_token_header"`

	// Injection safety
	AllowNewlines bool `json:"allow_newlines" yaml:"allow_newlines"`
	MaxInjectLen  int  `json:"max_inject_len" yaml:"max_inject_len"`

	// Two-man rule (OFF by default)
	TwoManEnabled   bool `json:"two_man_enabled" yaml:"two_man_enabled"`
	ApproveWindowMs int  `json:"approve_window_ms" yaml:"approve_window_ms"`

	// Pointer = allows "default true but allow false"
	ApproveConsumeOnInject *bool `json:"approve_consume_on_inject" yaml:"approve_consume_on_inject"`

	ApproveMagic string `json:"approve_magic" yaml:"approve_magic"`
}

var cfg ServerConfig

const (
	defaultJSON = "server_config.json"
	defaultYAML = "server_config.yaml"
	defaultYML  = "server_config.yml"
)

func loadConfig() error {
	// Prefer YAML if present, else JSON
	path := pickConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
	default:
		return fmt.Errorf("unsupported config extension %q (use .json/.yaml/.yml)", ext)
	}

	applyDefaults()
	return nil
}

func pickConfigPath() string {
	if fileExists(defaultYAML) {
		return defaultYAML
	}
	if fileExists(defaultYML) {
		return defaultYML
	}
	return defaultJSON
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func applyDefaults() {
	// Core defaults
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

	// Arm defaults
	if cfg.ArmDurationMs == 0 {
		cfg.ArmDurationMs = 20000
	}
	if cfg.ArmConsumeOnInject == nil {
		v := true
		cfg.ArmConsumeOnInject = &v
	}
	if cfg.AllowClipboardWhenDisarmed == nil {
		v := true
		cfg.AllowClipboardWhenDisarmed = &v
	}

	// Arm API defaults
	if cfg.ArmListenAddr == "" {
		cfg.ArmListenAddr = "127.0.0.1:60769"
	}
	if cfg.ArmTokenFile == "" {
		cfg.ArmTokenFile = "arm_token.txt"
	}
	if cfg.ArmTokenHeader == "" {
		cfg.ArmTokenHeader = "X-NovaKey-Token"
	}

	// Safety defaults
	if cfg.MaxInjectLen == 0 {
		cfg.MaxInjectLen = 256
	}
	// AllowNewlines defaults false

	// Two-man defaults
	if cfg.ApproveWindowMs == 0 {
		cfg.ApproveWindowMs = 15000
	}
	if cfg.ApproveConsumeOnInject == nil {
		v := true
		cfg.ApproveConsumeOnInject = &v
	}
	if cfg.ApproveMagic == "" {
		cfg.ApproveMagic = "__NOVAKEY_APPROVE__"
	}
}

