// cmd/novakey/pairing_bootstrap.go
package main

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

// PairQR is what the iOS scanner expects to decode from the QR payload.
type PairQR struct {
	PairV              int    `json:"pair_v"`
	DeviceID           string `json:"device_id"`
	DeviceKeyHex       string `json:"device_key_hex"`
	ServerKyberPubB64  string `json:"server_kyber_pub_b64"`
	ListenPort         int    `json:"listen_port"`
	IssuedAtUnix       int64  `json:"issued_at_unix"`
	ExpiresAtUnix      int64  `json:"expires_at_unix"`
}

// ensureDevicesFileExistsAndShowQR implements:
// - If devices.json missing => generate a device, write devices.json, show QR.
func ensureDevicesFileExistsAndShowQR(devicesPath string) error {
	if devicesPath == "" {
		devicesPath = defaultDevicesFile
	}

	// If devices file already exists, nothing to do.
	if _, err := os.Stat(devicesPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat devices file %q: %w", devicesPath, err)
	}

	// Create parent dir if needed.
	if dir := filepath.Dir(devicesPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir devices dir: %w", err)
		}
	}

	// Generate device ID + 32-byte key.
	deviceID, err := randHex(16) // 32 hex chars
	if err != nil {
		return fmt.Errorf("rand device id: %w", err)
	}
	keyHex, err := randHex(32) // 64 hex chars => 32 bytes
	if err != nil {
		return fmt.Errorf("rand device key: %w", err)
	}

	// Redact these if they ever hit logs.
	addSecret(deviceID)
	addSecret(keyHex)

	// Determine listen port from cfg.ListenAddr (best effort).
	listenPort := 60768
	if _, p, err := net.SplitHostPort(cfg.ListenAddr); err == nil {
		// p is string port
		if v, err2 := parsePort(p); err2 == nil && v > 0 {
			listenPort = v
		}
	}

	// serverEncapKey is set by loadOrCreateServerKeys()
	if len(serverEncapKey) == 0 {
		return fmt.Errorf("serverEncapKey empty; server keys not initialized")
	}
	serverKyberPubB64 := base64.StdEncoding.EncodeToString(serverEncapKey)

	blob := newDefaultPairQR(deviceID, keyHex, serverKyberPubB64, listenPort)
	pairURL, err := buildPairURL(blob)
	if err != nil {
		return fmt.Errorf("build pair url: %w", err)
	}
	addSecret(pairURL)

	// Write devices.json immediately (autosave / simplest UX).
	dc := devicesConfigFile{
		Devices: []deviceConfig{
			{ID: deviceID, KeyHex: keyHex},
		},
	}
	j, err := json.MarshalIndent(&dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devices file: %w", err)
	}

	tmp := devicesPath + ".tmp"
	perm := os.FileMode(0o600)
	if runtime.GOOS == "windows" {
		perm = 0o644
	}
	if err := os.WriteFile(tmp, j, perm); err != nil {
		return fmt.Errorf("write tmp devices file: %w", err)
	}
	if err := os.Rename(tmp, devicesPath); err != nil {
		return fmt.Errorf("rename tmp devices file: %w", err)
	}

	absDev, _ := filepath.Abs(devicesPath)
	log.Printf("[pair] %s was missing; created a new device and wrote %s", filepath.Base(devicesPath), absDev)

	// Where to put the QR PNG
	outDir := pairingOutputDir()
	pngPath, openErr := writeAndOpenQRPNG(outDir, pairURL)

	absQR, _ := filepath.Abs(pngPath)
	log.Printf("[pair] QR code written to %s", absQR)
	log.Printf("[pair] Scan the QR with the NovaKey iOS app to add this device.")
	log.Printf("[pair] (Treat the QR as a secret; it contains the device key.)")

	// Not fatal if viewer can't open.
	if openErr != nil {
		log.Printf("[pair] NOTE: failed to open image viewer automatically: %v", openErr)
	}

	return nil
}

func pairingOutputDir() string {
	// Prefer cache dir; fall back to current directory.
	if d, err := os.UserCacheDir(); err == nil && d != "" {
		return filepath.Join(d, "novakey")
	}
	return "."
}

func randHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func parsePort(s string) (int, error) {
	// tiny int parser without strconv
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-digit in port")
		}
		n = n*10 + int(c-'0')
		if n > 65535 {
			return 0, fmt.Errorf("port too large")
		}
	}
	return n, nil
}

func newDefaultPairQR(deviceID, deviceKeyHex, serverKyberPubB64 string, listenPort int) PairQR {
	now := time.Now()
	return PairQR{
		PairV:             1,
		DeviceID:          deviceID,
		DeviceKeyHex:      deviceKeyHex,
		ServerKyberPubB64: serverKyberPubB64,
		ListenPort:        listenPort,
		IssuedAtUnix:      now.Unix(),
		ExpiresAtUnix:     now.Add(2 * time.Minute).Unix(),
	}
}

func buildPairURL(blob PairQR) (string, error) {
	raw, err := json.Marshal(blob)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		_ = zw.Close()
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}

	data := base64.RawURLEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("novakey://pair?v=1&data=%s", data), nil
}

func writeAndOpenQRPNG(outDir string, payload string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	pngPath := filepath.Join(outDir, "novakey-pair.png")

	// 512px is plenty for this size payload.
	if err := qrcode.WriteFile(payload, qrcode.Medium, 512, pngPath); err != nil {
		return "", err
	}

	if err := openDefault(pngPath); err != nil {
		// Not fatal: file exists; user can open manually.
		return pngPath, fmt.Errorf("wrote %s but failed to open viewer: %w", pngPath, err)
	}

	return pngPath, nil
}

func openDefault(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
