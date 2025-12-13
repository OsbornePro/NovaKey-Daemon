package main

import (
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
)

type serverKeys struct {
    KyberPub  string `json:"kyber768_public"`
    KyberPriv string `json:"kyber768_secret"`
}

var srvKeys serverKeys

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

    log.Printf("Loaded server Kyber keys from %s", abs)
    return nil
}

func generateAndSaveServerKeys(path string) error {
    pub, priv, err := generateKyberKeyPair()
    if err != nil {
        return fmt.Errorf("generateKyberKeyPair: %w", err)
    }

    srvKeys = serverKeys{
        KyberPub:  base64.StdEncoding.EncodeToString(pub),
        KyberPriv: base64.StdEncoding.EncodeToString(priv),
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

    abs, _ := filepath.Abs(path)
    log.Printf("Generated new server Kyber keys at %s", abs)
    return nil
}

// TEMPORARY placeholder until we wire in a real Kyber-768 implementation.
func generateKyberKeyPair() ([]byte, []byte, error) {
    // Approximate Kyber-768 sizes; exact values are not critical until we actually
    // perform KEM operations. For now, this is just opaque key material that
    // will be handed to the phone app in pairing.
    const pubLen = 1184 // typical Kyber-768 public key size
    const privLen = 2400 // typical Kyber-768 secret key size

    pub := make([]byte, pubLen)
    priv := make([]byte, privLen)

    if _, err := rand.Read(pub); err != nil {
        return nil, nil, fmt.Errorf("rand.Read pub: %w", err)
    }
    if _, err := rand.Read(priv); err != nil {
        return nil, nil, fmt.Errorf("rand.Read priv: %w", err)
    }

    return pub, priv, nil
}

