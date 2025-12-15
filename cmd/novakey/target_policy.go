// cmd/novakey/target_policy.go
package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
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

	procKey := normProcKey(procRaw)  // e.g. "C:\...\msedge.exe" -> "msedge"
	titleKey := normTitleKey(titleRaw) // normalized for substring checks

	// Denylist first (wins)
	if matchProcExact(procKey, cfg.DeniedProcessNames) || matchTitleSubstring(titleKey, cfg.DeniedWindowTitles) {
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

	if matchProcExact(procKey, allowedProcs) || matchTitleSubstring(titleKey, allowedTitles) {
		return nil
	}

	return fmt.Errorf("focused target not allowed (proc=%q title=%q)", procRaw, titleRaw)
}

func builtInAllowedProcessNames() []string {
	// Canonical process keys (NO .exe needed). We normalize anyway.
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
		"totalpassword", "total password",

		// Testing / editors
		"notepad",
		"textedit",
		"gedit",
		"kate",
	}
}

func builtInAllowedWindowTitleHints() []string {
	// Substring match on normalized title
	return []string{
		"chrome", "chromium", "brave", "firefox", "safari", "edge", "opera", "vivaldi",
		"duckduckgo", "ecosia", "aloha",
		"1password", "bitwarden", "dashlane", "keeper", "roboform", "nordpass", "proton", "lastpass",
		"notepad", "textedit", "gedit", "kate",
	}
}

/*
Normalization helpers

Goal:
- make "msedge", "msedge.exe", and "C:\...\msedge.exe" all compare equal
- also tolerate friendly names like "Microsoft Edge" in config
*/

func normProcKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// If a full path sneaks in, reduce to basename.
	s = filepath.Base(s)

	// Lowercase
	s = strings.ToLower(s)

	// Strip common Windows extension (we compare canonical key without .exe)
	if strings.HasSuffix(s, ".exe") {
		s = strings.TrimSuffix(s, ".exe")
	}

	// Collapse to alnum only so "Microsoft Edge" -> "microsoftedge"
	s = stripToAlnum(s)

	// Alias map to reduce common “friendly names” to canonical process keys
	if aliased, ok := procAliases()[s]; ok {
		return aliased
	}
	return s
}

func normTitleKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// lower; keep spaces (we’ll substring match)
	s = strings.ToLower(s)

	// Some apps include weird unicode separators; normalize by stripping control-ish chars.
	// (We keep most characters; just drop non-printing/control.)
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)

	return s
}

func stripToAlnum(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func procAliases() map[string]string {
	// Keys must already be alnum-only + lowercase.
	return map[string]string{
		"microsoftedge": "msedge",
		"googlechrome":  "chrome",
		"bravebrowser":  "brave",
		"duckduckgobrowser": "duckduckgo",
		"protonpass":    "protonpass",
		"protonpassapp": "protonpass",
		"totalpassword": "totalpassword",
		"onepassword":   "1password",
	}
}

func matchProcExact(procKey string, list []string) bool {
	if procKey == "" || len(list) == 0 {
		return false
	}
	for _, it := range list {
		if normProcKey(it) == procKey {
			return true
		}
	}
	return false
}

func matchTitleSubstring(titleKey string, list []string) bool {
	if titleKey == "" || len(list) == 0 {
		return false
	}
	for _, it := range list {
		n := strings.ToLower(strings.TrimSpace(it))
		if n != "" && strings.Contains(titleKey, n) {
			return true
		}
	}
	return false
}

