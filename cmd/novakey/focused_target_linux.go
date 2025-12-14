// cmd/novakey/focused_target_linux.go
//go:build linux

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func getFocusedTarget() (string, string, error) {
	// Wayland: best effort is unreliable without compositor-specific protocols.
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return "", "", fmt.Errorf("wayland session: focused app detection not implemented")
	}

	// X11 path using xdotool
	winID, err := cmdOut("xdotool", "getwindowfocus")
	if err != nil {
		return "", "", fmt.Errorf("xdotool getwindowfocus: %w", err)
	}
	winID = strings.TrimSpace(winID)

	title, _ := cmdOut("xdotool", "getwindowname", winID)
	title = strings.TrimSpace(title)

	pidStr, err := cmdOut("xdotool", "getwindowpid", winID)
	if err != nil {
		return "", title, fmt.Errorf("xdotool getwindowpid: %w", err)
	}
	pidStr = strings.TrimSpace(pidStr)

	// process name via /proc
	commPath := fmt.Sprintf("/proc/%s/comm", pidStr)
	b, err := os.ReadFile(commPath)
	if err == nil {
		proc := strings.TrimSpace(string(b))
		return proc, title, nil
	}

	// fallback: ps
	proc, err := cmdOut("ps", "-p", pidStr, "-o", "comm=")
	if err != nil {
		return "", title, fmt.Errorf("ps comm: %w", err)
	}
	return strings.TrimSpace(proc), title, nil
}

func cmdOut(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return out.String(), nil
}

