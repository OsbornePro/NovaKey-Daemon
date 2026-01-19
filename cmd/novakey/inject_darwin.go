// cmd/novakey/inject_darwin.go
//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// macOS injection:
// - Default (per keylogger concern): clipboard paste (pbcopy + Cmd+V), then OPTIONAL AppleScript typing fallback.
// - We return which method was used so the client can show a clear visual cue.
func InjectPasswordToFocusedControl(password string) (InjectMethod, error) {
	log.Printf("[darwin] InjectPasswordToFocusedControl called; len=%d", len(password))

	preferClipboard := boolDeref(cfg.MacOSPreferClipboard, true)
	allowTyping := boolDeref(cfg.AllowTypingFallback, true)

	if preferClipboard {
		if err := injectViaClipboardPaste(password); err == nil {
			log.Printf("[darwin] clipboard+Cmd+V path succeeded")
			return InjectMethodClipboard, nil
		} else {
			log.Printf("[darwin] clipboard-paste failed: %v", err)
		}

		if allowTyping {
			if err := injectViaAppleScriptType(password); err != nil {
				return "", fmt.Errorf("clipboard paste failed and typing fallback failed: %w", err)
			}
			log.Printf("[darwin] AppleScript keystroke typing succeeded (fallback)")
			return InjectMethodTyping, nil
		}

		return "", fmt.Errorf("clipboard paste failed and typing fallback disabled")
	}

	// If user flips preference, try typing first.
	if allowTyping {
		if err := injectViaAppleScriptType(password); err == nil {
			log.Printf("[darwin] AppleScript keystroke typing succeeded")
			return InjectMethodTyping, nil
		} else {
			log.Printf("[darwin] AppleScript typing failed: %v", err)
		}
	}

	if err := injectViaClipboardPaste(password); err != nil {
		return "", fmt.Errorf("typing failed/disabled and clipboard paste failed: %w", err)
	}
	log.Printf("[darwin] clipboard+Cmd+V path succeeded (fallback)")
	return InjectMethodClipboard, nil
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

