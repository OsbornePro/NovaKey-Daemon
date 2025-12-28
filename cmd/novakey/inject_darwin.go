// cmd/novakey/inject_darwin.go
//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// macOS: primary = clipboard+Cmd+V, alternate = keystroke typing via AppleScript
func InjectPasswordToFocusedControl(password string) error {
	log.Printf("[darwin] InjectPasswordToFocusedControl called; len=%d", len(password))

	if err := injectViaClipboardPaste(password); err == nil {
		log.Printf("[darwin] clipboard+Cmd+V path succeeded")
		return nil
	} else {
		log.Printf("[darwin] clipboard-paste failed; falling back to keystroke typing: %v", err)
	}

	if err := injectViaAppleScriptType(password); err != nil {
		return fmt.Errorf("both clipboard-paste and keystroke typing failed: %w", err)
	}

	log.Printf("[darwin] keystroke typing succeeded")
	return nil
}

func injectViaClipboardPaste(password string) error {
	// Save clipboard (best-effort)
	var oldClipboard []byte
	if out, err := exec.Command("pbpaste").Output(); err == nil {
		oldClipboard = out
	}

	// Set clipboard
	setCmd := exec.Command("pbcopy")
	setCmd.Stdin = bytes.NewBufferString(password)
	if out, err := setCmd.CombinedOutput(); err != nil {
		if len(out) > 0 {
			log.Printf("[darwin] pbcopy output: %s", string(out))
		}
		return fmt.Errorf("pbcopy failed: %w", err)
	}

	// Cmd+V via System Events
	ascript := `tell application "System Events" to keystroke "v" using command down`
	keyCmd := exec.Command("osascript", "-e", ascript)
	if out, err := keyCmd.CombinedOutput(); err != nil {
		if len(out) > 0 {
			log.Printf("[darwin] osascript cmd+v output: %s", string(out))
		}
		restoreClipboardMac(oldClipboard)
		return fmt.Errorf("osascript cmd+v failed: %w", err)
	}

	restoreClipboardMac(oldClipboard)
	return nil
}

func restoreClipboardMac(old []byte) {
	if len(old) == 0 {
		return
	}
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewReader(old)
	_, _ = cmd.CombinedOutput()
}

func injectViaAppleScriptType(password string) error {
	script := `
on run argv
    set t to item 1 of argv
    tell application "System Events"
        keystroke t
    end tell
end run
`
	cmd := exec.Command("osascript", "-e", script, "--", password)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		log.Printf("[darwin] osascript type output: %s", string(out))
	}
	if err != nil {
		return fmt.Errorf("osascript keystroke failed: %w", err)
	}
	return nil
}
