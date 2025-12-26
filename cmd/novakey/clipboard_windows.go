// cmd/novakey/clipboard_windows.go
//go:build windows

package main

// Windows clipboard helper.
// Uses the existing setClipboardText() from inject_windows.go.
func trySetClipboard(text string) error {
	return setClipboardText(text)
}
