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

// Linux injection:
//
// - Wayland: do NOT attempt keystroke injection; return ErrInjectUnavailableWayland.
// - X11/Xwayland: type via xdotool.
// - IMPORTANT: do NOT set clipboard unless the user enabled clipboard fallback AND injection failed;
//   that logic lives in msg_handler.go via allowClipboardOnInjectFailure().
func InjectPasswordToFocusedControl(password string) (InjectMethod, error) {
	display := os.Getenv("DISPLAY")
	session := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))

	log.Printf("[linux] InjectPasswordToFocusedControl called; len=%d DISPLAY=%s XDG_SESSION_TYPE=%s",
		len(password), display, session)

	// Wayland path
	if session == "wayland" || os.Getenv("WAYLAND_DISPLAY") != "" {
		log.Printf("[linux] Wayland session detected; keystroke injection not supported")
		return "", ErrInjectUnavailableWayland
	}

	// X11 / Xwayland typing via xdotool
	if err := injectViaXdotoolType(password); err != nil {
		return "", fmt.Errorf("xdotool typing failed: %w", err)
	}

	return InjectMethodTyping, nil
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

