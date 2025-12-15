// cmd/novakey/target_policy.go
package main

import (
	"fmt"
	"strings"
)

func enforceTargetPolicy() error {
	// ✅ Only enforce when explicitly enabled.
	if !cfg.TargetPolicyEnabled {
		return nil
	}

	proc, title, err := getFocusedTarget()
	if err != nil {
		// Preserve your existing error strings (Wayland, etc.)
		return err
	}

	procNorm := normalizeProcName(proc)
	titleNorm := strings.ToLower(strings.TrimSpace(title))

	// Build normalized allow/deny lists
	allowProcs := normalizeProcList(cfg.AllowedProcessNames)
	denyProcs := normalizeProcList(cfg.DeniedProcessNames)

	allowTitles := normalizeTitleList(cfg.AllowedWindowTitles)
	denyTitles := normalizeTitleList(cfg.DeniedWindowTitles)

	// Deny wins
	if procNorm != "" && stringInSlice(procNorm, denyProcs) {
		return fmt.Errorf("focused target denied (proc=%q title=%q)", proc, title)
	}
	if titleNorm != "" && titleMatchesAny(titleNorm, denyTitles) {
		return fmt.Errorf("focused target denied (proc=%q title=%q)", proc, title)
	}

	// If any allowlist is present, require a match
	if len(allowProcs) > 0 || len(allowTitles) > 0 {
		if procNorm != "" && stringInSlice(procNorm, allowProcs) {
			return nil
		}
		if titleNorm != "" && titleMatchesAny(titleNorm, allowTitles) {
			return nil
		}
		return fmt.Errorf("focused target not allowed (proc=%q title=%q)", proc, title)
	}

	// If enabled but no lists were provided, optionally fall back to built-in allowlist.
	// If UseBuiltInAllowlist is false here, we allow all.
	if !cfg.UseBuiltInAllowlist {
		return nil
	}

	builtin := []string{
		"msedge", "chrome", "chromium", "brave", "firefox", "opera", "vivaldi", "safari",
		"1password", "bitwarden", "lastpass", "dashlane", "keeper", "nordpass", "protonpass", "roboform",
		"notepad", "textedit", "gedit", "kate",
	}
	builtin = normalizeProcList(builtin)

	if procNorm != "" && stringInSlice(procNorm, builtin) {
		return nil
	}

	return fmt.Errorf("focused target not allowed (proc=%q title=%q)", proc, title)
}

func normalizeProcName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	// strip trailing .exe for Windows proc names
	if strings.HasSuffix(s, ".exe") {
		s = strings.TrimSuffix(s, ".exe")
	}
	return s
}

func normalizeProcList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, x := range in {
		n := normalizeProcName(x)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func normalizeTitleList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, x := range in {
		n := strings.ToLower(strings.TrimSpace(x))
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func stringInSlice(s string, list []string) bool {
	for _, x := range list {
		if s == x {
			return true
		}
	}
	return false
}

// Title rules: case-insensitive substring match.
// (If you want exact match only, change strings.Contains -> ==)
func titleMatchesAny(titleLower string, patternsLower []string) bool {
	for _, p := range patternsLower {
		if p == "" {
			continue
		}
		if strings.Contains(titleLower, p) {
			return true
		}
	}
	return false
}

