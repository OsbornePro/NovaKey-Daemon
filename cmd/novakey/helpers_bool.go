// cmd/novakey/helpers_bool.go
package main

func boolDeref(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

func allowClipboardWhenBlocked() bool {
	if cfg.AllowClipboardWhenDisarmed == nil {
		return false
	}
	return *cfg.AllowClipboardWhenDisarmed
}
