// cmd/novakey/darwin_main.go
//go:build darwin

package main

import (
	"fmt"
	"log"
	"time"
)

func maybeStartPairingQR() {
	if isPaired() {
		return
	}

	host, port := splitHostPortOrDie(cfg.ListenAddr)
	advertiseHost := chooseAdvertiseHost(host)

	tokenB64, tokenID, exp := startOrRefreshPairToken(10 * time.Minute)
	fp := fp16Hex(serverEncapKey)

	qr := fmt.Sprintf("novakey://pair?v=4&host=%s&port=%d&token=%s&fp=%s&exp=%d",
		advertiseHost, port, tokenB64, fp, exp.Unix())

	pngPath, err := writeAndOpenPairQR(".", qr)
	if err != nil {
		log.Printf("[pair] token id=%s expires=%s; QR at %s (viewer open failed: %v)",
			tokenID, exp.Format(time.RFC3339), pngPath, err)
	} else {
		log.Printf("[pair] token id=%s expires=%s; QR opened at %s",
			tokenID, exp.Format(time.RFC3339), pngPath)
	}
}

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("loadConfig failed: %v", err)
	}
	initLoggingFromConfig()

	if err := initCrypto(); err != nil {
		log.Fatalf("initCrypto failed: %v", err)
	}

	maybeStartPairingQR()
	startArmAPI()

	if err := startUnifiedListener(); err != nil {
		log.Fatalf("startUnifiedListener failed: %v", err)
	}

	log.Printf("NovaKey (macOS) started (listener=%s)", cfg.ListenAddr)
	select {}
}
