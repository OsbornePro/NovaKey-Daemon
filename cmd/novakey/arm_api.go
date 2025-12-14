package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

func ensureArmToken(path string) (string, error) {
	// If token exists, validate perms (unix) then read it.
	if fi, err := os.Lstat(path); err == nil {
		// Unix hardening: refuse insecure perms / symlinks
		if runtime.GOOS != "windows" {
			// Refuse symlinks for the token file (prevents redirects)
			if fi.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("arm token file is a symlink (refusing for safety): %s", path)
			}

			// Require no group/other bits (0600 or stricter)
			perm := fi.Mode().Perm()
			if perm&0o077 != 0 {
				return "", fmt.Errorf(
					"arm token file has insecure permissions (must be 0600 or stricter): %s (got %04o)",
					path, perm,
				)
			}
		}

		// Read token
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read arm token file: %w", err)
		}
		t := string(bytesTrimSpace(b))
		if t == "" {
			return "", fmt.Errorf("arm token file is empty: %s", path)
		}
		return t, nil
	}

	// Create a new token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	tok := base64.RawURLEncoding.EncodeToString(raw)

	// Best-effort restrictive perms on unix; windows will ignore modes.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("create token file: %w", err)
	}
	defer f.Close()

	if _, err := io.WriteString(f, tok+"\n"); err != nil {
		return "", fmt.Errorf("write token file: %w", err)
	}

	// Note: never log the token value.
	return tok, nil
}

// tiny helper to avoid importing strings for one thing
func bytesTrimSpace(b []byte) string {
	// very small trim for \r\n\t space
	i := 0
	j := len(b)
	for i < j {
		c := b[i]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			i++
			continue
		}
		break
	}
	for j > i {
		c := b[j-1]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			j--
			continue
		}
		break
	}
	return string(b[i:j])
}

func startArmAPI() {
	if !cfg.ArmAPIEnabled {
		log.Printf("[arm] arm API disabled")
		return
	}

	// Safety: refuse to bind non-loopback
	host, _, err := net.SplitHostPort(cfg.ArmListenAddr)
	if err != nil {
		log.Printf("[arm] invalid arm_listen_addr=%q: %v", cfg.ArmListenAddr, err)
		return
	}
	if host != "127.0.0.1" && host != "::1" && host != "localhost" {
		log.Printf("[arm] refusing to bind non-loopback address: %s", cfg.ArmListenAddr)
		return
	}

	token, err := ensureArmToken(cfg.ArmTokenFile)
	if err != nil {
		log.Printf("[arm] token init failed: %v", err)
		return
	}

	// Do NOT print token. Log only safe metadata.
	log.Printf("[arm] arm API enabled on http://%s (token file: %s, header: %s)",
		cfg.ArmListenAddr, cfg.ArmTokenFile, cfg.ArmTokenHeader)

	mux := http.NewServeMux()

	mux.HandleFunc("/arm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get(cfg.ArmTokenHeader) != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ms := cfg.ArmDurationMs
		if q := r.URL.Query().Get("ms"); q != "" {
			if v, err := strconv.Atoi(q); err == nil && v > 0 && v <= 120000 {
				ms = v
			}
		}

		armGate.Arm(time.Duration(ms) * time.Millisecond)
		cfg.ArmEnabled = true // arming implies gate is active, even if normally off for testing
		fmt.Fprintf(w, "armed for %dms\n", ms)
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get(cfg.ArmTokenHeader) != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		until := armGate.ArmedUntil()
		fmt.Fprintf(w, "armed_until=%s\n", until.Format(time.RFC3339Nano))
	})

	srv := &http.Server{
		Addr:    cfg.ArmListenAddr,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[arm] ListenAndServe error: %v", err)
		}
	}()
}
