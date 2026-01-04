// cmd/novakey/darwin_main.go
//go:build darwin

package main

import (
	"log"
)

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("loadConfig failed: %v", err)
	}
	initLoggingFromConfig()

	if err := initCrypto(); err != nil {
		log.Fatalf("initCrypto failed: %v", err)
	}

	maybeStartPairingQR()

	if err := startUnifiedListener(); err != nil {
		log.Fatalf("startUnifiedListener failed: %v", err)
	}

	log.Printf("NovaKey (macOS) started (listener=%s)", cfg.ListenAddr)
	select {}
}
