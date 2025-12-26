// cmd/novakey/helpers_bool.go
package main

func boolDeref(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

// allowClipboardWhenBlocked controls clipboard fallback when policy/gates block injection:
// unsafe text, target policy, needs approve, not armed, etc.
func allowClipboardWhenBlocked() bool {
	return boolDeref(cfg.AllowClipboardWhenDisarmed, false)
}

// allowClipboardOnInjectFailure controls clipboard fallback only when injection fails
// AFTER gates passed (e.g., Wayland can't inject).
func allowClipboardOnInjectFailure() bool {
	// default should already be set in applyDefaults(), but keep a safe fallback
	return boolDeref(cfg.AllowClipboardOnInjectFailure, false)
}
