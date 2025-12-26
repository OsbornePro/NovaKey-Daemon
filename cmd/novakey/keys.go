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
	KyberPriv string `json:"kyber768_secret"` // base64 seed (dk.Bytes())
}

var (
	srvKeys        serverKeys
	serverDecapKey *mlkem768.DecapsulationKey
	serverEncapKey []byte // raw bytes
)

// loadOrCreateServerKeys loads server_keys.json if present; otherwise generates new keys.
// If cfg.RotateKyberKeys is true, it ALWAYS generates and overwrites the file.
func loadOrCreateServerKeys(path string) error {
	if path == "" {
		path = "server_keys.json"
	}
	abs, _ := filepath.Abs(path)

	if cfg.RotateKyberKeys {
		log.Printf("[keys] rotate_kyber_keys=true; generating new ML-KEM-768 keypair (%s)", abs)
		return generateAndSaveServerKeys(path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[keys] server keys file %s not found; generating new ML-KEM-768 keypair", abs)
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

	if err := materializeServerKeys(abs); err != nil {
		return err
	}

	log.Printf("[keys] loaded server ML-KEM keys from %s", abs)
	return nil
}

func materializeServerKeys(absPath string) error {
	privSeed, err := base64.StdEncoding.DecodeString(srvKeys.KyberPriv)
	if err != nil {
		return fmt.Errorf("decoding kyber768_secret in %s: %w", absPath, err)
	}
	if len(privSeed) != mlkem768.SeedSize {
		return fmt.Errorf("kyber768_secret in %s has wrong length: got %d want %d",
			absPath, len(privSeed), mlkem768.SeedSize)
	}

	dk, err := mlkem768.NewKeyFromSeed(privSeed)
	if err != nil {
		return fmt.Errorf("mlkem768.NewKeyFromSeed: %w", err)
	}
	serverDecapKey = dk

	pubBytes, err := base64.StdEncoding.DecodeString(srvKeys.KyberPub)
	if err != nil {
		return fmt.Errorf("decoding kyber768_public in %s: %w", absPath, err)
	}
	if len(pubBytes) != mlkem768.EncapsulationKeySize {
		return fmt.Errorf("kyber768_public in %s has wrong length: got %d want %d",
			absPath, len(pubBytes), mlkem768.EncapsulationKeySize)
	}

	// Optional consistency check: pub in file should match pub derived from seed
	derived := dk.EncapsulationKey()
	if len(derived) == len(pubBytes) {
		same := true
		for i := range derived {
			if derived[i] != pubBytes[i] {
				same = false
				break
			}
		}
		if !same {
			return fmt.Errorf("server keys mismatch in %s: public key does not match private seed", absPath)
		}
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

	privSeed := dk.Bytes()
	pubKey := dk.EncapsulationKey()

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

	// Initialize runtime objects from what we just created
	if err := materializeServerKeys(abs); err != nil {
		return err
	}

	log.Printf("[keys] generated new server ML-KEM keys at %s", abs)
	return nil
}
