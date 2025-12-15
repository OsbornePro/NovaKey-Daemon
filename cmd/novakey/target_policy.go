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
		len(cfg.DeniedWindowTitles) == 0 &&
		!cfg.UseBuiltInAllowlist {
		return nil
	}

	procRaw, titleRaw, err := getFocusedTarget()
	if err != nil {
		return fmt.Errorf("getFocusedTarget: %w", err)
	}

	// Normalize title once (proc normalization is done inside matchProc)
	title := normTitle(titleRaw)

	// Denylist first (wins)
	if matchProc(procRaw, cfg.DeniedProcessNames) || matchTitle(title, cfg.DeniedWindowTitles) {
		return fmt.Errorf("focused target denied (proc=%q title=%q)", procRaw, titleRaw)
	}

	// Build effective allowlists: config + optional built-in allowlist
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

	if matchProc(procRaw, allowedProcs) || matchTitle(title, allowedTitles) {
		return nil
	}

	return fmt.Errorf("focused target not allowed (proc=%q title=%q)", procRaw, titleRaw)
}

// --------------------- Built-ins ---------------------

func builtInAllowedProcessNames() []string {
	// NOTE: we intentionally list base names (no .exe required).
	// matchProc() will normalize and handle .exe/.app/path.
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

		// Password managers (best-effort)
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

// --------------------- Matching helpers ---------------------

func normTitle(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func normProc(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// If caller gave a path, keep only basename
	s = filepath.Base(s)

	s = strings.ToLower(strings.TrimSpace(s))

	// Strip common platform suffixes
	if strings.HasSuffix(s, ".exe") {
		s = strings.TrimSuffix(s, ".exe")
	}
	if strings.HasSuffix(s, ".app") {
		s = strings.TrimSuffix(s, ".app")
	}

	return strings.TrimSpace(s)
}

func matchProc(procRaw string, list []string) bool {
	if procRaw == "" || len(list) == 0 {
		return false
	}

	pNorm := normProc(procRaw)
	pRaw := strings.ToLower(strings.TrimSpace(filepath.Base(procRaw)))

	for _, it := range list {
		if strings.TrimSpace(it) == "" {
			continue
		}
		n := normProc(it)
		r := strings.ToLower(strings.TrimSpace(filepath.Base(it)))

		// Accept:
		// - base vs base
		// - base vs base.exe/.app
		// - literal match if they supplied suffix
		if n != "" && n == pNorm {
			return true
		}
		if r != "" && r == pRaw {
			return true
		}
	}
	return false
}

func matchTitle(titleNorm string, list []string) bool {
	if titleNorm == "" || len(list) == 0 {
		return false
	}
	for _, it := range list {
		n := normTitle(it)
		if n != "" && strings.Contains(titleNorm, n) {
			return true
		}
	}
	return false
}

