//go:build linux
// +build linux

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func TypeString(s string) {
	time.Sleep(600 * time.Millisecond)

	// Use xdotool if available (most Linux desktops have it)
	cmd := exec.Command("xdotool", "type", "--clearmodifiers", "--delay", "15", s)
	cmd.Run()

	// Press Enter
	exec.Command("xdotool", "key", "Return").Run()
	time.Sleep(100 * time.Millisecond)
}

// Fallback if xdotool not installed
func init() {
	if _, err := exec.LookPath("xdotool"); err != nil {
		fmt.Println("[!] WARNING: xdotool not found. Auto-type will not work on Linux.")
		fmt.Println("    Install with: sudo apt install xdotool   (or your package manager)")
	}
}
