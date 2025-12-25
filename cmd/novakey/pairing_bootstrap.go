// cmd/novakey/pairing_bootstrap.go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

type pairingBlobV3 struct {
	V              int    `json:"v"`
	DeviceID       string `json:"device_id"`
	DeviceKeyHex   string `json:"device_key_hex"`
	ServerAddr     string `json:"server_addr"`
	ServerKyberPub string `json:"server_kyber768_pub"`
	ExpiresAtUnix  int64  `json:"expires_at_unix"`
}

type pairingState struct {
	mu         sync.Mutex
	active     bool
	token      string
	deviceID   string
	deviceKey  string // hex, 32 bytes
	serverAddr string
	expires    time.Time
	done       bool

	// for cleanup
	qrPngPath string
}

// IMPORTANT: snapshot type (no mutex) to avoid copying sync.Mutex
type pairingSnapshot struct {
	active     bool
	token      string
	deviceID   string
	deviceKey  string
	serverAddr string
	expires    time.Time
	done       bool
	qrPngPath  string
}

var pairState pairingState

// maybeStartPairingBootstrap starts the QR + pairing HTTP server if devices file is missing/empty.
func maybeStartPairingBootstrap() {
	if isPaired() {
		return
	}

	// Only start once
	pairState.mu.Lock()
	if pairState.active {
		pairState.mu.Unlock()
		return
	}
	pairState.active = true
	pairState.mu.Unlock()

	host, listenPort := splitHostPortOrDie(cfg.ListenAddr)

	// Pair API port: listenPort + 2 (avoid +1 because ArmAPI default is 60769)
	pairPort := listenPort + 2
	pairBind := fmt.Sprintf("0.0.0.0:%d", pairPort)

	advertiseHost := chooseAdvertiseHost(host)
	serverAddr := fmt.Sprintf("%s:%d", advertiseHost, listenPort)

	// Generate pending device + token
	token := randHex(16)
	deviceID := "ios-" + randHex(8)
	deviceKeyHex := randHex(32) // 32 bytes => 64 hex chars

	exp := time.Now().Add(5 * time.Minute)

	pairState.mu.Lock()
	pairState.token = token
	pairState.deviceID = deviceID
	pairState.deviceKey = deviceKeyHex
	pairState.serverAddr = serverAddr
	pairState.expires = exp
	pairState.done = false
	pairState.qrPngPath = ""
	pairState.mu.Unlock()

	// Start HTTP server for the phone to fetch the big blob
	mux := http.NewServeMux()
	mux.HandleFunc("/pair/status", handlePairStatus)
	mux.HandleFunc("/pair/bootstrap", handlePairBootstrap) // GET
	mux.HandleFunc("/pair/complete", handlePairComplete)   // POST

	go func() {
		log.Printf("[pair] Pairing API listening on http://%s (advertise host=%s)", pairBind, advertiseHost)
		if err := http.ListenAndServe(pairBind, mux); err != nil {
			log.Printf("[pair] pairing HTTP server stopped: %v", err)
		}
	}()

	// Small QR payload (easy to scan)
	// Phone scans -> calls:
	//   GET http://host:pairPort/pair/bootstrap?token=...
	qr := fmt.Sprintf("novakey://pair?v=2&host=%s&port=%d&token=%s", advertiseHost, pairPort, token)

	// Write QR to disk + open viewer
	outDir := "."
	pngPath, err := writeAndOpenPairQR(outDir, qr)

	pairState.mu.Lock()
	pairState.qrPngPath = pngPath
	pairState.mu.Unlock()

	if err != nil {
		log.Printf("[pair] QR written to %s but viewer open failed: %v", pngPath, err)
	} else {
		log.Printf("[pair] Opened QR at %s", pngPath)
	}

	log.Printf("[pair] Waiting for phone to fetch bootstrap + complete pairing.")
	log.Printf("[pair] (expires %s) server_addr=%s device_id=%s", exp.Format(time.RFC3339), serverAddr, deviceID)
}

func handlePairStatus(w http.ResponseWriter, r *http.Request) {
	st := currentPairState()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"active":  st.active,
		"done":    st.done,
		"expires": st.expires.Unix(),
		"server":  st.serverAddr,
	})
}

func handlePairBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st := currentPairState()
	if !st.active || st.token == "" {
		http.Error(w, "pairing not active", http.StatusNotFound)
		return
	}

	got := r.URL.Query().Get("token")
	if got == "" || got != st.token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if time.Now().After(st.expires) {
		http.Error(w, "expired", http.StatusGone)
		return
	}

	// Big blob returned here (device key + Kyber pubkey, etc.)
	pub := base64.StdEncoding.EncodeToString(serverEncapKey)

	blob := pairingBlobV3{
		V:              3,
		DeviceID:       st.deviceID,
		DeviceKeyHex:   st.deviceKey,
		ServerAddr:     st.serverAddr,
		ServerKyberPub: pub,
		ExpiresAtUnix:  st.expires.Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&blob)
}

func handlePairComplete(w http.ResponseWriter, r *http.Request) {
	// Phone calls this after it has successfully saved pairing info.
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st := currentPairState()
	got := r.URL.Query().Get("token")
	if got == "" || got != st.token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if time.Now().After(st.expires) {
		http.Error(w, "expired", http.StatusGone)
		return
	}

	// Save devices using OS-specific store:
	// - Windows: DPAPI wrapper
	// - macOS/Linux: keyring-sealed (or fallback)
	if err := writeDevicesFile(cfg.DevicesFile, st.deviceID, st.deviceKey); err != nil {
		http.Error(w, "write devices failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload into memory so daemon starts accepting requests immediately
	if err := reloadDevicesFromDisk(); err != nil {
		http.Error(w, "reload devices failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Mark complete + fetch QR path for cleanup
	pairState.mu.Lock()
	pairState.done = true
	qrPath := pairState.qrPngPath
	pairState.mu.Unlock()

	// Best-effort: delete QR png after pairing finishes.
	if qrPath != "" {
		_ = os.Remove(qrPath)
	}

	_, _ = w.Write([]byte("ok\n"))
	log.Printf("[pair] pairing complete; devices saved + loaded (device_id=%s)", st.deviceID)
}

// currentPairState returns a mutex-free snapshot (avoids copying sync.Mutex).
func currentPairState() pairingSnapshot {
	pairState.mu.Lock()
	defer pairState.mu.Unlock()

	return pairingSnapshot{
		active:     pairState.active,
		token:      pairState.token,
		deviceID:   pairState.deviceID,
		deviceKey:  pairState.deviceKey,
		serverAddr: pairState.serverAddr,
		expires:    pairState.expires,
		done:       pairState.done,
		qrPngPath:  pairState.qrPngPath,
	}
}

func writeDevicesFile(path, deviceID, deviceKeyHex string) error {
	if strings.TrimSpace(path) == "" {
		path = "devices.json"
	}
	out := devicesConfigFile{
		Devices: []deviceConfig{
			{ID: deviceID, KeyHex: deviceKeyHex},
		},
	}
	return saveDevicesToDisk(path, out)
}

func writeAndOpenPairQR(outDir, payload string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	pngPath := filepath.Join(outDir, "novakey-pair.png")

	// Low density, easy scan
	if err := qrcode.WriteFile(payload, qrcode.Low, 512, pngPath); err != nil {
		return "", err
	}
	if err := openDefault(pngPath); err != nil {
		return pngPath, err
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

func splitHostPortOrDie(addr string) (string, int) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		if n, err2 := strconv.Atoi(addr); err2 == nil {
			return "127.0.0.1", n
		}
		log.Fatalf("invalid listen_addr %q: %v", addr, err)
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		log.Fatalf("invalid listen_addr port %q: %v", addr, err)
	}
	return h, n
}

func chooseAdvertiseHost(listenHost string) string {
	h := strings.TrimSpace(listenHost)
	if h == "" {
		return firstNonLoopbackIPv4OrLocalhost()
	}
	if h == "0.0.0.0" || h == "127.0.0.1" || h == "::" || h == "::1" || strings.EqualFold(h, "localhost") {
		return firstNonLoopbackIPv4OrLocalhost()
	}
	return h
}

func firstNonLoopbackIPv4OrLocalhost() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, iface := range ifaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue
		}
		if (iface.Flags & net.FlagLoopback) != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String()
		}
	}
	return "127.0.0.1"
}

func randHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
