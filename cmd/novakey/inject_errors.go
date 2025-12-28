package main

import "errors"

// Sentinel error used by msg_handler.go to detect "can't inject; clipboard-only is acceptable" cases.
// Defined in a common file so all targets compile.
var ErrInjectUnavailableWayland = errors.New("inject unavailable on wayland")
