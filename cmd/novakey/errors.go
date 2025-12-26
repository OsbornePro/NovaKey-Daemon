// cmd/novakey/errors.go
package main

import "errors"

// ErrNotPaired means there is no usable pairing material on disk (missing/empty).
// This is the ONLY condition that should trigger pairing mode.
var ErrNotPaired = errors.New("not paired (devices file missing/empty)")

// ErrDevicesUnavailable means device material exists but cannot be accessed/decrypted
// (e.g., keyring locked/unavailable, DPAPI unprotect failure, corruption).
// This should be treated as fatal (do NOT enter pairing mode).
var ErrDevicesUnavailable = errors.New("devices unavailable (cannot decrypt/access device store)")
