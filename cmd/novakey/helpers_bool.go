// cmd/novakey/helpers_bool.go
package main

func boolDeref(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

// allowClipboardWhenBlocked controls clipboard use when policy/gates block injection:
// unsafe text, target policy, needs approve, not armed, etc.
func allowClipboardWhenBlocked() bool {
	return boolDeref(cfg.AllowClipboardWhenDisarmed, false)
}

// allowClipboardOnInjectFailure controls clipboard use only when injection fails
// AFTER gates passed (e.g., Wayland can't inject).
func allowClipboardOnInjectFailure() bool {
    // default should already be set in applyDefaults(), but keep a safe default
	return boolDeref(cfg.AllowClipboardOnInjectFailure, false)
}
