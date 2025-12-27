// cmd/novakey/pair_qr_start.go
package main

import (
	"fmt"
	"log"
	"time"
)

// maybeStartPairingQR triggers pairing QR generation when no devices are paired.
// Shared across all platforms to avoid drift.
func maybeStartPairingQR() {
	if isPaired() {
		return
	}

	host, port := splitHostPortOrDie(cfg.ListenAddr)
	advertiseHost := chooseAdvertiseHost(host)

	tokenB64, tokenID, exp := startOrRefreshPairToken(10 * time.Minute)
	fp := fp16Hex(serverEncapKey)

	// Stable scheme for your app to parse. Keep in sync with iOS client.
	qr := fmt.Sprintf("novakey://pair?v=3&host=%s&port=%d&token=%s&fp=%s&exp=%d",
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
