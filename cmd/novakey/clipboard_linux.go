// cmd/novakey/clipboard_linux.go
//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// trySetClipboard best-effort copies text to the user's clipboard.
// On Wayland it prefers wl-copy (wl-clipboard). On X11 it prefers xclip.
func trySetClipboard(text string) error {
	session := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))

	// Prefer Wayland-native clipboard tool when on Wayland.
	if session == "wayland" {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command("wl-copy")
			cmd.Stdin = strings.NewReader(text)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("wl-copy failed: %v (%s)", err, strings.TrimSpace(string(out)))
			}
			return nil
		}
		// Fall through to xclip if wl-copy is missing.
	}

	// X11 / fallback: xclip.
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("xclip failed: %v (%s)", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	return fmt.Errorf("no clipboard tool found (need wl-copy for Wayland or xclip for X11)")
}

