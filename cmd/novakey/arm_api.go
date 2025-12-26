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
	"sync/atomic"
)

type armTokenSnapshot struct {
	Path  string
	Token string
}

var armTokenCache atomic.Value // stores armTokenSnapshot

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

	// Load token once; cache it for request handlers; also add to redaction secrets.
	if tok, err := readArmToken(cfg.ArmTokenFile); err == nil && tok != "" {
		snap := armTokenSnapshot{Path: cfg.ArmTokenFile, Token: tok}
		armTokenCache.Store(snap)
		addSecret(tok)
	}

	log.Printf("[arm] arm API enabled on http://%s (token file: %s, header: %s)",
		cfg.ArmListenAddr, cfg.ArmTokenFile, cfg.ArmTokenHeader)

	mux := armMuxForTests()

	go func() {
		if err := http.ListenAndServe(cfg.ArmListenAddr, mux); err != nil {
			log.Printf("[arm] http server stopped: %v", err)
		}
	}()
}

func cachedArmToken() (string, error) {
	// If we have a cached snapshot and it matches current config path, use it.
	if v := armTokenCache.Load(); v != nil {
		if snap, ok := v.(armTokenSnapshot); ok {
			if snap.Path == cfg.ArmTokenFile && strings.TrimSpace(snap.Token) != "" {
				return strings.TrimSpace(snap.Token), nil
			}
		}
	}

	// Otherwise read from disk for current cfg.ArmTokenFile.
	tok, err := readArmToken(cfg.ArmTokenFile)
	if err != nil {
		return "", err
	}
	tok = strings.TrimSpace(tok)
	if tok != "" {
		armTokenCache.Store(armTokenSnapshot{Path: cfg.ArmTokenFile, Token: tok})
		addSecret(tok)
	}
	return tok, nil
}

func requireArmToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := cachedArmToken()
		if err != nil || token == "" {
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
