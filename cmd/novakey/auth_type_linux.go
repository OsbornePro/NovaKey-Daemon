//go:build linux
// +build linux

package main

import (
	"fmt"
	"os/exec"
	"time"
)

// SecureType uses xdotool to type the UTF-8 bytes without exposing the
// password in the process command line. It streams the bytes via stdin.
func SecureType(b []byte) {
	time.Sleep(600 * time.Millisecond)

	if _, err := exec.LookPath("xdotool"); err != nil {
		fmt.Println("[!] xdotool not found. Cannot auto-type on Linux.")
		return
	}

	// Use "type --file -" so the text comes from stdin, not argv.
	cmd := exec.Command("xdotool", "type", "--clearmodifiers", "--delay", "15", "--file", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("[!] Failed to open stdin for xdotool:", err)
		return
	}

	// Stream the password bytes to xdotool via stdin so they never
	// appear in argv or the environment.
	go func() {
		_, _ = stdin.Write(b)
		_ = stdin.Close()
	}()

	_ = cmd.Run()
	_ = exec.Command("xdotool", "key", "Return").Run()
	time.Sleep(100 * time.Millisecond)
}

func init() {
	if _, err := exec.LookPath("xdotool"); err != nil {
		fmt.Println("[!] WARNING: xdotool not found. Auto-type will not work on Linux.")
		fmt.Println("    Install with: sudo apt install xdotool")
	}
}
