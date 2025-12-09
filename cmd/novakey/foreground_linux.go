//go:build linux
// +build linux

package main

import (
	"errors"
	"os/exec"
	"strings"
)

func foregroundAppAllowed() (bool, string, error) {
	// Get active window ID
	out, err := exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
	if err != nil {
		return false, "", errors.New("xprop failed (no X11?)")
	}

	fields := strings.Fields(string(out))
	if len(fields) < 5 {
		return false, "", errors.New("unexpected xprop output")
	}
	windowID := fields[4]

	// Query WM_CLASS
	out, err = exec.Command("xprop", "-id", windowID, "WM_CLASS").Output()
	if err != nil {
		return false, "", errors.New("failed to query WM_CLASS")
	}

	lower := strings.ToLower(string(out))

	// NOTE: Linux allowlist can be added later.
	// For now: deny unless explicitly allowed in future config.
	return false, lower, nil
}
