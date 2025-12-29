// cmd/novakey/helpers_bool.go
package main

func boolDeref(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

func allowClipboardWhenBlocked() bool {
	return boolDeref(cfg.AllowClipboardWhenDisarmed, false)
}

func allowClipboardOnInjectFailure() bool {
	return boolDeref(cfg.AllowClipboardOnInjectFailure, false)
}
