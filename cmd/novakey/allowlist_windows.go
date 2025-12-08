//go:build windows
// +build windows

package main

import "strings"

var allowedForegroundApps = map[string]struct{}{
	// Browsers
	"chrome.exe":     {},
	"chromium.exe":  {},
	"msedge.exe":    {},
	"opera.exe":     {},
	"firefox.exe":   {},
	"waterfox.exe":  {},
	"brave.exe":     {},
	"browser.exe":   {}, // Arc, Chromium shells
	"comet.exe":     {},
	"workona.exe":   {},
	"chatgpt.exe":   {},

	// Password Managers
	"bitwarden.exe": {},
	"protonpass.exe": {},
	"keeper.exe":    {},
	"keepass.exe":   {},
	"keepassxc.exe": {},
	"lastpass.exe":  {},
	"1password.exe": {},
	"nordpass.exe":  {},
	"roboform.exe":  {},
}

func foregroundAppAllowed() (bool, string, error) {
	exe, err := getForegroundExe()
	if err != nil {
		return false, "", err
	}

	_, ok := allowedForegroundApps[strings.ToLower(exe)]
	return ok, exe, nil
}
