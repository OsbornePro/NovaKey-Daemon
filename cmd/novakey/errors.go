// cmd/novakey/errors.go
package main

import "errors"

// ErrNotPaired means no usable pairing material exists yet (missing/empty store).
// This is the ONLY case that should trigger pairing mode.
var ErrNotPaired = errors.New("not paired (devices file missing/empty)")

// ErrDevicesUnavailable means the store exists but cannot be read/decrypted/parsed.
// This must be treated as fatal (do NOT start pairing automatically).
var ErrDevicesUnavailable = errors.New("devices unavailable (cannot decrypt/access device store)")
