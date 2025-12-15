// cmd/novakey/arm_api.go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
)

func startArmAPI() {
	if !cfg.ArmAPIEnabled {
		return
	}

	// Refuse non-loopback binds.
	if !isLoopbackListenAddr(cfg.ArmListenAddr) {
		log.Printf("[arm] refused to start: arm_listen_addr must be loopback (got %q)", cfg.ArmListenAddr)
		return
	}

	// Token init (create if missing, validate perms on Unix).
	if err := initArmTokenFile(cfg.ArmTokenFile); err != nil {
		log.Printf("[arm] token init failed: %v", err)
		return
	}

	log.Printf("[arm] arm API enabled on http://%s (token file: %s, header: %s)",
		cfg.ArmListenAddr, cfg.ArmTokenFile, cfg.ArmTokenHeader)

	mux := armMuxForTests()

	// Start serving in background.
	go func() {
		if err := http.ListenAndServe(cfg.ArmListenAddr, mux); err != nil {
			log.Printf("[arm] http server stopped: %v", err)
		}
	}()
}

func requireArmToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := readArmToken(cfg.ArmTokenFile)
		if err != nil {
			http.Error(w, "arm token not available", http.StatusInternalServerError)
			return
		}

		got := strings.TrimSpace(r.Header.Get(cfg.ArmTokenHeader))
		if got == "" || got != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func initArmTokenFile(path string) error {
	// If exists, validate perms (Unix) and return.
	if _, err := os.Stat(path); err == nil {
		if runtime.GOOS != "windows" {
			if err := ensureFileMode0600(path); err != nil {
				return err
			}
		}
		return nil
	}

	// Create new token and write file 0600 on Unix.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("rand: %w", err)
	}
	token := hex.EncodeToString(b)

	perm := os.FileMode(0600)
	if runtime.GOOS == "windows" {
		perm = 0644 // Windows ACLs differ; keep it readable to the user.
	}

	if err := os.WriteFile(path, []byte(token+"\n"), perm); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}

	// Re-check perms on Unix to catch umask surprises.
	if runtime.GOOS != "windows" {
		if err := ensureFileMode0600(path); err != nil {
			return err
		}
	}
	return nil
}

func readArmToken(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	tok := strings.TrimSpace(string(b))
	if tok == "" {
		return "", errors.New("empty token")
	}
	return tok, nil
}

func ensureFileMode0600(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := fi.Mode().Perm()
	// Reject group/world readable/writable/executable.
	if mode&0077 != 0 {
		return fmt.Errorf("arm token file has insecure permissions (must be 0600 or stricter): %s (got %04o)", path, mode)
	}
	return nil
}

func isLoopbackListenAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	// If host is a name, resolve and ensure all results are loopback.
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, x := range ips {
		if !x.IsLoopback() {
			return false
		}
	}
	return true
}

