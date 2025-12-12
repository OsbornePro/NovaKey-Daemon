// inject_linux.go
//go:build linux

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// On Linux, we prioritize reliability:
//
//   1) Fire-and-forget: try to copy password to the clipboard (xclip) in a goroutine.
//      If it hangs or fails, injection still continues.
//   2) Synchronous: type the password into the currently focused window using xdotool type.
//
// This is "real typing" (keyloggers can see it), but it's robust across X11/Xwayland apps.
// macOS and Windows still keep the "non-typing first, typing fallback" behavior.
func InjectPasswordToFocusedControl(password string) error {
	log.Printf("[linux] InjectPasswordToFocusedControl called; len=%d DISPLAY=%s XDG_SESSION_TYPE=%s",
		len(password), os.Getenv("DISPLAY"), os.Getenv("XDG_SESSION_TYPE"))

	// 1) Try to set clipboard in the background so it can't block injection
	go func(p string) {
		if err := setClipboardLinux(p); err != nil {
			log.Printf("[linux] async setClipboardLinux failed: %v", err)
		} else {
			log.Printf("[linux] async password copied to clipboard")
		}
	}(password)

	// 2) Main path: xdotool type (synchronous)
	if err := injectViaXdotoolType(password); err != nil {
		log.Printf("[linux] xdotool typing failed: %v", err)
		return fmt.Errorf("xdotool typing failed: %w", err)
	}

	log.Printf("[linux] xdotool typing succeeded")
	return nil
}

// setClipboardLinux sets the clipboard contents using xclip.
// This may block or behave oddly on some Wayland/Xwayland setups,
// so we always call it from a goroutine.
func setClipboardLinux(password string) error {
	cmd := exec.Command("xclip", "-i", "-selection", "clipboard")
	cmd.Stdin = bytes.NewBufferString(password)
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
// Whatever window currently has focus will receive it.
func injectViaXdotoolType(password string) error {
	log.Printf("[linux] injectViaXdotoolType start")
	cmd := exec.Command("xdotool", "type", "--clearmodifiers", "--delay", "1", "--", password)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		log.Printf("[linux] xdotool type output: %s", string(out))
	}
	if err != nil {
		return fmt.Errorf("xdotool type failed: %w", err)
	}
	return nil
}

