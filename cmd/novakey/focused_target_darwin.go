// cmd/novakey/focused_target_darwin.go
//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Uses osascript (System Events). This is "best effort" and may require Accessibility permissions.
func getFocusedTarget() (string, string, error) {
	// App name
	appScript := `tell application "System Events" to get name of first application process whose frontmost is true`
	app, err := runAppleScript(appScript)
	if err != nil {
		return "", "", fmt.Errorf("osascript app name: %w", err)
	}
	app = strings.TrimSpace(app)

	// Window title (best effort; can fail for some apps)
	titleScript := `tell application "System Events" to tell (first application process whose frontmost is true) to get name of front window`
	title, _ := runAppleScript(titleScript)
	title = strings.TrimSpace(title)

	if app == "" {
		return "", title, fmt.Errorf("frontmost app name empty")
	}
	return app, title, nil
}

func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
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

