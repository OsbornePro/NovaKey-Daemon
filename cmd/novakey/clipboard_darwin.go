// cmd/novakey/clipboard_darwin.go
//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// macOS clipboard helper: pbcopy.
// Return nil on success, error otherwise.
func trySetClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("pbcopy failed: %v: %s", err, string(out))
		}
		return fmt.Errorf("pbcopy failed: %v", err)
	}
	return nil
}

