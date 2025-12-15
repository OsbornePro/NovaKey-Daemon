// cmd/novakey/clipboard_linux.go
//go:build linux

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Linux clipboard helper: Wayland -> wl-copy, X11 -> xclip.
// Return nil on success, error otherwise.
func trySetClipboard(text string) error {
	isWayland := os.Getenv("WAYLAND_DISPLAY") != "" || strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland")

	if isWayland {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command("wl-copy")
			cmd.Stdin = strings.NewReader(text)
			if err := cmd.Run(); err == nil {
				log.Printf("[clipboard] set via wl-copy (wayland)")
				return nil
			} else {
				log.Printf("[clipboard] wl-copy failed: %v (will try xclip fallback)", err)
			}
		} else {
			log.Printf("[clipboard] wl-copy not found in PATH (will try xclip fallback)")
		}
	}

	// X11 path: xclip -selection clipboard
	if _, err := exec.LookPath("xclip"); err != nil {
		if isWayland {
			return fmt.Errorf("no clipboard helper available: wl-copy missing/failed and xclip not found")
		}
		return fmt.Errorf("xclip not found in PATH: %w", err)
	}

	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xclip failed: %w", err)
	}
	log.Printf("[clipboard] set via xclip")
	return nil
}

