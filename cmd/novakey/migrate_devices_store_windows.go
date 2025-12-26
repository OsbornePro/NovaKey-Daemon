// cmd/novakey/migrate_devices_store_windows.go
//go:build windows

package main

import (
	"fmt"
	"os"
)

func init() {
	if len(os.Args) >= 2 && os.Args[1] == "migrate-devices-store" {
		fmt.Fprintln(os.Stderr, "migrate-devices-store: not needed on Windows (device store uses DPAPI sealing).")
		os.Exit(0)
	}
}
