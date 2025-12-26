package main

import (
	"fmt"
	"strings"
)

func validateInjectText(s string) error {
	if cfg.MaxInjectLen > 0 && len(s) > cfg.MaxInjectLen {
		return fmt.Errorf("inject text too long: %d > max_inject_len=%d", len(s), cfg.MaxInjectLen)
	}
	if !cfg.AllowNewlines {
		if strings.ContainsAny(s, "\r\n") {
			return fmt.Errorf("inject text contains newline but allow_newlines=false")
		}
	}
	return nil
}
