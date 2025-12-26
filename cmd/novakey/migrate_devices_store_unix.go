// cmd/novakey/migrate_devices_store_unix.go
//go:build !windows

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

func init() {
	// One-shot command: `novakey migrate-devices-store`
	if len(os.Args) >= 2 && os.Args[1] == "migrate-devices-store" {
		if err := runMigrateDevicesStore(); err != nil {
			fmt.Fprintf(os.Stderr, "migrate-devices-store: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "migrate-devices-store: OK")
		os.Exit(0)
	}
}

func runMigrateDevicesStore() error {
	// Load config so we know cfg.DevicesFile path (and any custom location).
	if err := loadConfig(); err != nil {
		return err
	}

	path := cfg.DevicesFile
	if path == "" {
		path = "devices.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	// If it already looks like a sealed wrapper, do nothing.
	var wrap sealedDevicesFileV1
	if err := json.Unmarshal(data, &wrap); err == nil &&
		wrap.V == 1 &&
		wrap.Alg == "xchacha20poly1305" &&
		wrap.NonceB64 != "" &&
		wrap.CtB64 != "" {
		fmt.Fprintf(os.Stdout, "%s already appears sealed (v=%d alg=%s); nothing to do.\n", path, wrap.V, wrap.Alg)
		return nil
	}

	// Parse legacy plaintext devices.json
	var dc devicesConfigFile
	if err := json.Unmarshal(data, &dc); err != nil {
		return fmt.Errorf("devices file is not a sealed wrapper and not valid plaintext json: %w", err)
	}
	if len(dc.Devices) == 0 {
		return fmt.Errorf("no devices found in %s; refusing to write a sealed empty store", path)
	}

	// Require keyring / sealing for migration (fail closed).
	key, err := getOrCreateDevicesKey()
	if err != nil {
		return fmt.Errorf("keyring unavailable; cannot seal devices store: %w", err)
	}

	// Create a backup first (0600).
	if err := writeBackup0600(path, data); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	// Seal and write atomically (0600).
	if err := sealAndWrite0600(path, &dc, key); err != nil {
		return fmt.Errorf("seal/write: %w", err)
	}

	log.Printf("[info] migrated %s from plaintext -> sealed wrapper (backup created)", path)
	return nil
}

func writeBackup0600(path string, data []byte) error {
	ts := time.Now().Unix()
	backup := fmt.Sprintf("%s.bak.%d", path, ts)

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(backup, data, 0o600); err != nil {
		return fmt.Errorf("write backup %s: %w", backup, err)
	}
	return nil
}

func sealAndWrite0600(path string, dc *devicesConfigFile, key []byte) error {
	pt, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plaintext: %w", err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return fmt.Errorf("NewX: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("rand nonce: %w", err)
	}

	ct := aead.Seal(nil, nonce, pt, []byte(devicesSealedAAD))

	wrap := sealedDevicesFileV1{
		V:        1,
		Alg:      "xchacha20poly1305",
		NonceB64: base64.StdEncoding.EncodeToString(nonce),
		CtB64:    base64.StdEncoding.EncodeToString(ct),
	}

	out, err := json.MarshalIndent(&wrap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal wrapper: %w", err)
	}

	// atomic write via temp + rename, perms 0600
	return atomicWrite0600(path, out)
}
