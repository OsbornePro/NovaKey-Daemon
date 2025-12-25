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

	// --------------------
	// Logging (optional)
	// --------------------
	// If LogFile is set, logs go to that file (with rotation).
	// Else if LogDir is set, logs go to LogDir/novakey.log (with rotation).
	// If neither is set, logs go to stderr only.
	LogFile     string `json:"log_file" yaml:"log_file"`
	LogDir      string `json:"log_dir" yaml:"log_dir"`
	LogRotateMB int    `json:"log_rotate_mb" yaml:"log_rotate_mb"` // default 10
	LogKeep     int    `json:"log_keep" yaml:"log_keep"`           // default 10
	LogStderr   *bool  `json:"log_stderr" yaml:"log_stderr"`       // default true
	LogRedact   *bool  `json:"log_redact" yaml:"log_redact"`       // default true

	// Arm gate (OFF by default)
	ArmEnabled         bool  `json:"arm_enabled" yaml:"arm_enabled"`
	ArmDurationMs      int   `json:"arm_duration_ms" yaml:"arm_duration_ms"`
	ArmConsumeOnInject *bool `json:"arm_consume_on_inject" yaml:"arm_consume_on_inject"`

	// When blocked (disarmed / two-man missing), allow clipboard copy?
	// Default FALSE (safer). Can be set true explicitly in config.
	AllowClipboardWhenDisarmed *bool `json:"allow_clipboard_when_disarmed" yaml:"allow_clipboard_when_disarmed"`

	// Local-only arming endpoint (OFF by default)
	ArmAPIEnabled  bool   `json:"arm_api_enabled" yaml:"arm_api_enabled"`
	ArmListenAddr  string `json:"arm_listen_addr" yaml:"arm_listen_addr"`
	ArmTokenFile   string `json:"arm_token_file" yaml:"arm_token_file"`
	ArmTokenHeader string `json:"arm_token_header" yaml:"arm_token_header"`

	// Injection safety
	AllowNewlines bool `json:"allow_newlines" yaml:"allow_newlines"`
	MaxInjectLen  int  `json:"max_inject_len" yaml:"max_inject_len"`

	// Two-man items
	TwoManEnabled             bool   `json:"two_man_enabled" yaml:"two_man_enabled"`
	ApproveWindowMs           int    `json:"approve_window_ms" yaml:"approve_window_ms"`
	ApproveConsumeOnInject    *bool  `json:"approve_consume_on_inject" yaml:"approve_consume_on_inject"`
	ApproveMagic              string `json:"approve_magic" yaml:"approve_magic"`
	LegacyApproveMagicEnabled bool   `json:"legacy_approve_magic_enabled" yaml:"legacy_approve_magic_enabled"`

	// Target policy (allow/deny of focused app)
	TargetPolicyEnabled bool     `json:"target_policy_enabled" yaml:"target_policy_enabled"`
	UseBuiltInAllowlist bool     `json:"use_built_in_allowlist" yaml:"use_built_in_allowlist"`
	AllowedProcessNames []string `json:"allowed_process_names" yaml:"allowed_process_names"`
	AllowedWindowTitles []string `json:"allowed_window_titles" yaml:"allowed_window_titles"`
	DeniedProcessNames  []string `json:"denied_process_names" yaml:"denied_process_names"`
	DeniedWindowTitles  []string `json:"denied_window_titles" yaml:"denied_window_titles"`
}

var cfg ServerConfig

const (
	defaultJSON = "server_config.json"
	defaultYAML = "server_config.yaml"
	defaultYML  = "server_config.yml"
)

func loadConfig() error {
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

	// IMPORTANT: initialize logging after defaults are applied.
	// This avoids needing to edit platform mains.
	initLoggingFromConfig()

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

	// Logging defaults
	if cfg.LogRotateMB == 0 {
		cfg.LogRotateMB = 10
	}
	if cfg.LogKeep == 0 {
		cfg.LogKeep = 10
	}
	if cfg.LogStderr == nil {
		v := true
		cfg.LogStderr = &v
	}
	if cfg.LogRedact == nil {
		v := true
		cfg.LogRedact = &v
	}

	// Arm defaults
	if cfg.ArmDurationMs == 0 {
		cfg.ArmDurationMs = 20000
	}
	if cfg.ArmConsumeOnInject == nil {
		v := true
		cfg.ArmConsumeOnInject = &v
	}

	// ✅ CHANGE: default to false instead of true
	if cfg.AllowClipboardWhenDisarmed == nil {
		v := false
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

	// Target policy defaults
	if cfg.TargetPolicyEnabled && !cfg.UseBuiltInAllowlist &&
		len(cfg.AllowedProcessNames) == 0 && len(cfg.AllowedWindowTitles) == 0 &&
		len(cfg.DeniedProcessNames) == 0 && len(cfg.DeniedWindowTitles) == 0 {
		cfg.UseBuiltInAllowlist = true
	}
}
