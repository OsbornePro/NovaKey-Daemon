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
)

func ensureArmToken(path string) (string, error) {
	// If token exists, read it
	if st, err := os.Stat(path); err == nil {
		// Refuse weak perms on Unix: must not be readable/writable/executable by group/other.
		if runtime.GOOS != "windows" {
			perm := st.Mode().Perm()
			if (perm & 0o077) != 0 {
				return "", fmt.Errorf("arm token file has insecure permissions (must be 0600 or stricter): %s (got %04o)", path, perm)
			}
		}

		if b, err := os.ReadFile(path); err == nil {
			t := string(bytesTrimSpace(b))
			if t == "" {
				return "", fmt.Errorf("arm token file is empty: %s", path)
			}
			return t, nil
		}
	}

	// Create a new token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	tok := base64.RawURLEncoding.EncodeToString(raw)

	// Best-effort restrictive perms on unix; windows will ignore modes.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("create token file: %w", err)
	}
	defer f.Close()
	if _, err := io.WriteString(f, tok+"\n"); err != nil {
		return "", fmt.Errorf("write token file: %w", err)
	}
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

	log.Printf("[arm] arm API enabled on http://%s (token file: %s, header: %s)",
		cfg.ArmListenAddr, cfg.ArmTokenFile, cfg.ArmTokenHeader)

	// IMPORTANT: routes are registered ONLY in buildArmAPIMux to avoid duplicate patterns.
	mux := buildArmAPIMux(token)

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

