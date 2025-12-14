// cmd/novakey/target_policy.go
package main

import (
	"fmt"
	"strings"
)

func enforceTargetPolicy() error {
	// If policy disabled AND no explicit allow/deny lists => no restriction.
	if !cfg.TargetPolicyEnabled &&
		len(cfg.AllowedProcessNames) == 0 &&
		len(cfg.AllowedWindowTitles) == 0 &&
		len(cfg.DeniedProcessNames) == 0 &&
		len(cfg.DeniedWindowTitles) == 0 {
		return nil
	}

	proc, title, err := getFocusedTarget()
	if err != nil {
		return fmt.Errorf("getFocusedTarget: %w", err)
	}

	p := norm(proc)
	t := norm(title)

	// Denylist first (wins)
	if matchExact(p, cfg.DeniedProcessNames) || matchSubstring(t, cfg.DeniedWindowTitles) {
		return fmt.Errorf("focused target denied (proc=%q title=%q)", proc, title)
	}

	// Compute allowlists: explicit config + optional built-in allowlist
	allowedProcs := make([]string, 0, len(cfg.AllowedProcessNames)+64)
	allowedTitles := make([]string, 0, len(cfg.AllowedWindowTitles)+64)

	allowedProcs = append(allowedProcs, cfg.AllowedProcessNames...)
	allowedTitles = append(allowedTitles, cfg.AllowedWindowTitles...)

	if cfg.UseBuiltInAllowlist {
		allowedProcs = append(allowedProcs, builtInAllowedProcessNames()...)
		allowedTitles = append(allowedTitles, builtInAllowedWindowTitleHints()...)
	}

	// If allowlists empty => allow (unless denied above)
	if len(allowedProcs) == 0 && len(allowedTitles) == 0 {
		return nil
	}

	if matchExact(p, allowedProcs) || matchSubstring(t, allowedTitles) {
		return nil
	}

	return fmt.Errorf("focused target not allowed (proc=%q title=%q)", proc, title)
}

func builtInAllowedProcessNames() []string {
	// Browsers (common process names across OSes)
	return []string{
		// Edge / Chrome-family
		"msedge", "msedge.exe", "microsoft edge",
		"chrome", "chrome.exe", "google chrome",
		"chromium", "chromium-browser", "chromium.exe",
		"brave", "brave.exe", "brave-browser", "brave browser",
		"vivaldi", "vivaldi.exe",
		"opera", "opera.exe",

		// Firefox
		"firefox", "firefox.exe",

		// Safari (macOS)
		"safari",

		// DuckDuckGo desktop browser (names vary)
		"duckduckgo", "duckduckgo.exe", "duckduckgo browser",

		// Ecosia / Aloha (best-effort; names vary)
		"ecosia", "ecosia.exe",
		"aloha", "aloha.exe",

		// Password managers (best-effort)
		"1password", "1password.exe",
		"bitwarden", "bitwarden.exe",
		"lastpass", "lastpass.exe",
		"dashlane", "dashlane.exe",
		"keeper", "keeper.exe",
		"roboform", "roboform.exe",
		"nordpass", "nordpass.exe",
		"protonpass", "proton pass", "protonpass.exe",
		"aurapassword", "aura", "aura.exe",
		"norton", "norton.exe",
		"avira", "avira.exe",
		"totalpassword", "total password",
		"keepass", "keepass.exe", // common tester request

		// Notepad / basic editors for testing
		"notepad", "notepad.exe",
		"textedit", // macOS
		"gedit", "kate",
	}
}

func builtInAllowedWindowTitleHints() []string {
	// Substring match on window title (helps when process name is ambiguous)
	return []string{
		"chrome", "chromium", "brave", "firefox", "safari", "edge", "opera", "vivaldi",
		"1password", "bitwarden", "dashlane", "keeper", "roboform", "nordpass", "proton",
		"notepad", "textedit",
	}
}

func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func matchExact(val string, list []string) bool {
	if val == "" || len(list) == 0 {
		return false
	}
	for _, it := range list {
		if norm(it) == val {
			return true
		}
	}
	return false
}

func matchSubstring(val string, list []string) bool {
	if val == "" || len(list) == 0 {
		return false
	}
	for _, it := range list {
		n := norm(it)
		if n != "" && strings.Contains(val, n) {
			return true
		}
	}
	return false
}

