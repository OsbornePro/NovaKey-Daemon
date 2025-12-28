// cmd/novakey/inject_linux.go
//go:build linux

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// InjectPasswordToFocusedControl on Linux:
//
// - Wayland: do NOT attempt keystroke injection; return ErrInjectUnavailableWayland.
// - X11/Xwayland: set clipboard async (best effort) + type via xdotool.
func InjectPasswordToFocusedControl(password string) error {
	display := os.Getenv("DISPLAY")
	session := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))

	log.Printf("[linux] InjectPasswordToFocusedControl called; len=%d DISPLAY=%s XDG_SESSION_TYPE=%s",
		len(password), display, session)

	// Wayland path
	if session == "wayland" || os.Getenv("WAYLAND_DISPLAY") != "" {
		log.Printf("[linux] Wayland session detected; keystroke injection not supported")
		return ErrInjectUnavailableWayland
	}

	// X11 / Xwayland path: clipboard best-effort in background
	go func(p string) {
		if err := trySetClipboard(p); err != nil {
			log.Printf("[linux] async trySetClipboard failed: %v", err)
		}
	}(password)

	// Type via xdotool
	if err := injectViaXdotoolType(password); err != nil {
		return fmt.Errorf("xdotool typing failed: %w", err)
	}

	return nil
}

func injectViaXdotoolType(password string) error {
	cmd := exec.Command("xdotool", "type", "--clearmodifiers", "--delay", "1", "--", password)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		log.Printf("[linux] xdotool type output: %s", string(out))
	}
	if err != nil {
		return fmt.Errorf("xdotool type failed: %w", err)
	}
	return nil
}
