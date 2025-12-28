// cmd/novakey/inject_linux.go
//go:build linux

package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Sentinel error: injection is unavailable because session is Wayland.
// The caller can treat clipboard fallback as success in this case.
var ErrInjectUnavailableWayland = errors.New("inject unavailable on wayland")

// InjectPasswordToFocusedControl on Linux:
//
// - Wayland: do NOT attempt keystroke injection; return ErrInjectUnavailableWayland.
//   (Caller can choose clipboard-only success policy.)
//
// - X11/Xwayland: set clipboard async (best effort) + type via xdotool.
func InjectPasswordToFocusedControl(password string) error {
	display := os.Getenv("DISPLAY")
	session := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))

	log.Printf("[linux] InjectPasswordToFocusedControl called; len=%d DISPLAY=%s XDG_SESSION_TYPE=%s",
		len(password), display, session)

	// ----- Wayland path -----
	if session == "wayland" || os.Getenv("WAYLAND_DISPLAY") != "" {
		log.Printf("[linux] Wayland session detected; keystroke injection not supported")
		return ErrInjectUnavailableWayland
	}

	// ----- X11 / Xwayland path -----

	// 1) Try to set clipboard in the background so it can't block injection.
	go func(p string) {
		// Prefer the unified helper (xclip / wl-copy logic is in trySetClipboard).
		if err := trySetClipboard(p); err != nil {
			log.Printf("[linux] async trySetClipboard failed: %v", err)
		} else {
			log.Printf("[linux] async password copied to clipboard")
		}
	}(password)

	// 2) Main path: xdotool type (sync)
	if err := injectViaXdotoolType(password); err != nil {
		log.Printf("[linux] xdotool typing failed: %v", err)
		return fmt.Errorf("xdotool typing failed: %w", err)
	}

	log.Printf("[linux] xdotool typing succeeded")
	return nil
}

func injectViaXdotoolType(password string) error {
	log.Printf("[linux] injectViaXdotoolType start")
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

// NOTE: setClipboardLinux() removed from use to avoid xclip-only behavior on Wayland.
// Clipboard is handled via trySetClipboard() in clipboard_linux.go.
func setClipboardLinux(password string) error {
	cmd := exec.Command("xclip", "-i", "-selection", "clipboard")
	cmd.Stdin = bytes.NewBufferString(password)
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		log.Printf("[linux] xclip output: %s", string(out))
	}
	if err != nil {
		return fmt.Errorf("xclip set clipboard failed: %w", err)
	}
	return nil
}
