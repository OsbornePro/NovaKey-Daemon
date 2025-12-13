// cmd/novakey/keys.go
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"filippo.io/mlkem768"
)

type serverKeys struct {
	KyberPub  string `json:"kyber768_public"`
	KyberPriv string `json:"kyber768_secret"` // base64-encoded mlkem768 seed (dk.Bytes())
}

// srvKeys is the on-disk representation.
// serverDecapKey / serverEncapKey are the in-memory key objects.
var (
	srvKeys        serverKeys
	serverDecapKey *mlkem768.DecapsulationKey
	serverEncapKey []byte // public encapsulation key (raw bytes)
)

// loadOrCreateServerKeys loads server_keys.json if present,
// otherwise generates a new ML-KEM-768 keypair and writes it to disk.
//
// It also initializes serverDecapKey and serverEncapKey for runtime use.
func loadOrCreateServerKeys(path string) error {
	if path == "" {
		path = "server_keys.json"
	}
	abs, _ := filepath.Abs(path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("server keys file %s not found; generating new Kyber keypair", abs)
			return generateAndSaveServerKeys(path)
		}
		return fmt.Errorf("reading %s: %w", abs, err)
	}

	if err := json.Unmarshal(data, &srvKeys); err != nil {
		return fmt.Errorf("parsing %s: %w", abs, err)
	}
	if srvKeys.KyberPub == "" || srvKeys.KyberPriv == "" {
		return fmt.Errorf("invalid %s: missing kyber768_public or kyber768_secret", abs)
	}

	// Materialize ML-KEM keys
	if err := materializeServerKeys(abs); err != nil {
		return err
	}

	log.Printf("Loaded server Kyber keys from %s", abs)
	return nil
}

func materializeServerKeys(absPath string) error {
	// Decode private seed and build DecapsulationKey
	privSeed, err := base64.StdEncoding.DecodeString(srvKeys.KyberPriv)
	if err != nil {
		return fmt.Errorf("decoding kyber768_secret in %s: %w", absPath, err)
	}
	if len(privSeed) != mlkem768.SeedSize {
		return fmt.Errorf("kyber768_secret in %s has wrong length: got %d, want %d",
			absPath, len(privSeed), mlkem768.SeedSize)
	}

	dk, err := mlkem768.NewKeyFromSeed(privSeed)
	if err != nil {
		return fmt.Errorf("mlkem768.NewKeyFromSeed: %w", err)
	}
	serverDecapKey = dk

	// Decode public encapsulation key
	pubBytes, err := base64.StdEncoding.DecodeString(srvKeys.KyberPub)
	if err != nil {
		return fmt.Errorf("decoding kyber768_public in %s: %w", absPath, err)
	}
	if len(pubBytes) != mlkem768.EncapsulationKeySize {
		return fmt.Errorf("kyber768_public in %s has wrong length: got %d, want %d",
			absPath, len(pubBytes), mlkem768.EncapsulationKeySize)
	}
	serverEncapKey = pubBytes
	return nil
}

func generateAndSaveServerKeys(path string) error {
	abs, _ := filepath.Abs(path)

	dk, err := mlkem768.GenerateKey()
	if err != nil {
		return fmt.Errorf("mlkem768.GenerateKey: %w", err)
	}

	privSeed := dk.Bytes()           // SeedSize bytes
	pubKey := dk.EncapsulationKey()  // EncapsulationKeySize bytes

	srvKeys = serverKeys{
		KyberPub:  base64.StdEncoding.EncodeToString(pubKey),
		KyberPriv: base64.StdEncoding.EncodeToString(privSeed),
	}

	data, err := json.MarshalIndent(&srvKeys, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal server keys: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}

	// Also initialize runtime objects
	if err := materializeServerKeys(abs); err != nil {
		return err
	}

	log.Printf("Generated new server Kyber keys at %s", abs)
	return nil
}

