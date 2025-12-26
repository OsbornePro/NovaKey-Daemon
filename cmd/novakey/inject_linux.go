// cmd/novakey/inject_linux.go
//go:build linux

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// InjectPasswordToFocusedControl on Linux:
//
//   - If XDG_SESSION_TYPE=wayland:
//
//   - We DO NOT attempt keystroke injection (xdotool won't work).
//
//   - We best-effort copy the password to the clipboard in the background.
//
//   - We return an error so the logs clearly show that Wayland typing
//     is not supported yet.
//
//   - Otherwise (x11 / xwayland / unset):
//
//   - Original behavior: async clipboard via xclip,
//     plus synchronous keystroke typing via xdotool type.
func InjectPasswordToFocusedControl(password string) error {
	display := os.Getenv("DISPLAY")
	session := os.Getenv("XDG_SESSION_TYPE")

	log.Printf("[linux] InjectPasswordToFocusedControl called; len=%d DISPLAY=%s XDG_SESSION_TYPE=%s",
		len(password), display, session)

	// ----- Wayland path -----
	if session == "wayland" {
		log.Printf("[linux] Wayland session detected; keystroke injection via xdotool is not supported yet")

		// Still try to populate the clipboard so the user can paste manually.
		go func(p string) {
			if err := setClipboardLinux(p); err != nil {
				log.Printf("[linux] async setClipboardLinux failed (wayland): %v", err)
			} else {
				log.Printf("[linux] async password copied to clipboard (wayland; manual paste required)")
			}
		}(password)

		// Signal to caller/logs that typing did *not* occur.
		return fmt.Errorf("wayland session: keystroke injection not implemented; clipboard only")
	}

	// ----- X11 / Xwayland path (working behavior) -----

	// 1) Try to set clipboard in the background so it can't block injection
	go func(p string) {
		if err := setClipboardLinux(p); err != nil {
			log.Printf("[linux] async setClipboardLinux failed: %v", err)
		} else {
			log.Printf("[linux] async password copied to clipboard")
		}
	}(password)

	// 2) Main path: xdotool type (synchronous) - asyncrhonous I found was unable to insert the desired text
	if err := injectViaXdotoolType(password); err != nil {
		log.Printf("[linux] xdotool typing failed: %v", err)
		return fmt.Errorf("xdotool typing failed: %w", err)
	}

	log.Printf("[linux] xdotool typing succeeded")
	return nil
}

// setClipboardLinux sets the clipboard contents using xclip.
// This may behave oddly on some setups, so we call it from a goroutine.
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

// injectViaXdotoolType sends real keystrokes for each character of the password.
// Whatever window currently has focus will receive it (on X11/Xwayland).
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
