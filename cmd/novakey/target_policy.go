// cmd/novakey/target_policy.go
package main

import (
	"fmt"
	"path/filepath"
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

	procRaw, titleRaw, err := getFocusedTarget()
	if err != nil {
		return fmt.Errorf("getFocusedTarget: %w", err)
	}

	proc := canonProc(procRaw)
	title := norm(titleRaw)

	// Denylist first (wins)
	if matchProc(proc, cfg.DeniedProcessNames) || matchSubstring(title, cfg.DeniedWindowTitles) {
		return fmt.Errorf("focused target denied (proc=%q title=%q)", procRaw, titleRaw)
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

	if matchProc(proc, allowedProcs) || matchSubstring(title, allowedTitles) {
		return nil
	}

	return fmt.Errorf("focused target not allowed (proc=%q title=%q)", procRaw, titleRaw)
}

func builtInAllowedProcessNames() []string {
	// Names are normalized via canonProc(), so include “logical” names;
	// .exe and full paths are handled automatically.
	return []string{
		// Browsers
		"msedge", "microsoft edge",
		"chrome", "google chrome",
		"chromium", "chromium-browser",
		"brave", "brave-browser", "brave browser",
		"vivaldi",
		"opera",
		"firefox",
		"safari",
		"duckduckgo", "duckduckgo browser",
		"ecosia",
		"aloha",

		// Password managers
		"1password",
		"bitwarden",
		"lastpass",
		"dashlane",
		"keeper",
		"roboform",
		"nordpass",
		"protonpass", "proton pass",
		"aura",
		"norton",
		"avira",
		"totalpassword",
		"keepass",

		// Editors for testing
		"notepad",
		"textedit",
		"gedit",
		"kate",
	}
}

func builtInAllowedWindowTitleHints() []string {
	return []string{
		"chrome", "chromium", "brave", "firefox", "safari", "edge", "opera", "vivaldi",
		"1password", "bitwarden", "dashlane", "keeper", "roboform", "nordpass", "proton",
		"notepad", "textedit",
	}
}

// canonProc normalizes a process name for matching:
// - lowercases
// - strips any directory
// - strips trailing ".exe"
func canonProc(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Strip paths (handles both / and \ via filepath.Base)
	s = filepath.Base(s)

	s = strings.ToLower(strings.TrimSpace(s))

	// Strip ".exe" if present
	s = strings.TrimSuffix(s, ".exe")

	// Collapse internal whitespace (optional, helps "Microsoft Edge" variants)
	s = strings.Join(strings.Fields(s), " ")

	return s
}

func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func matchProc(procCanon string, list []string) bool {
	if procCanon == "" || len(list) == 0 {
		return false
	}
	for _, it := range list {
		if canonProc(it) == procCanon {
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

