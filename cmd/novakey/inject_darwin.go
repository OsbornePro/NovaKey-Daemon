//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// macOS: primary = clipboard+Cmd+V, fallback = keystroke typing via AppleScript
func InjectPasswordToFocusedControl(password string) error {
	log.Printf("[darwin] InjectPasswordToFocusedControl called; len=%d", len(password))

	if err := injectViaClipboardPaste(password); err == nil {
		log.Printf("[darwin] clipboard+Cmd+V path succeeded")
		return nil
	} else {
		log.Printf("[darwin] clipboard-paste injection failed, falling back to keystroke typing: %v", err)
	}

	if err := injectViaAppleScriptType(password); err != nil {
		log.Printf("[darwin] keystroke typing failed: %v", err)
		return fmt.Errorf("both clipboard-paste and keystroke typing failed: %w", err)
	}

	log.Printf("[darwin] keystroke typing succeeded")
	return nil
}

func injectViaClipboardPaste(password string) error {
	log.Printf("[darwin] injectViaClipboardPaste start")
	var oldClipboard []byte
	readCmd := exec.Command("pbpaste")
	if out, err := readCmd.Output(); err == nil {
		oldClipboard = out
	} else {
		log.Printf("pbpaste (read) failed; clipboard will not be restored: %v", err)
	}

	setCmd := exec.Command("pbcopy")
	setCmd.Stdin = bytes.NewBufferString(password)
	if out, err := setCmd.CombinedOutput(); err != nil {
		if len(out) > 0 {
			log.Printf("pbcopy (set) output: %s", string(out))
		}
		return fmt.Errorf("pbcopy (set) failed: %w", err)
	}

	ascript := `
tell application "System Events"
    keystroke "v" using command down
end tell
`
	keyCmd := exec.Command("osascript", "-e", ascript)
	if out, err := keyCmd.CombinedOutput(); err != nil {
		if len(out) > 0 {
			log.Printf("osascript cmd+v output: %s", string(out))
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
	restoreCmd := exec.Command("pbcopy")
	restoreCmd.Stdin = bytes.NewReader(old)
	if out, err := restoreCmd.CombinedOutput(); err != nil {
		if len(out) > 0 {
			log.Printf("pbcopy (restore) output: %s", string(out))
		}
		log.Printf("failed to restore clipboard: %v", err)
	}
}

func injectViaAppleScriptType(password string) error {
	log.Printf("[darwin] injectViaAppleScriptType start")
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
		log.Printf("osascript type output: %s", string(out))
	}
	if err != nil {
		return fmt.Errorf("osascript keystroke failed: %w", err)
	}
	return nil
}
