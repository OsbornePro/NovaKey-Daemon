// cmd/novakey/pair_bootstrap.go
package main

import (
	"fmt"
	"log"
	"time"
)

// maybeStartPairingBootstrap kept for compatibility with older call sites.
// It now generates the same QR payload shape as maybeStartPairingQR in *_main.go.
func maybeStartPairingBootstrap() {
	if isPaired() {
		return
	}
	if serverDecapKey == nil || len(serverEncapKey) == 0 {
		log.Printf("[pair] cannot start pairing: server keys not initialized")
		return
	}

	host, port := splitHostPortOrDie(cfg.ListenAddr)
	advertiseHost := chooseAdvertiseHost(host)

	tokenB64, tokenID, exp := startOrRefreshPairToken(10 * time.Minute)
	fp := fp16Hex(serverEncapKey)

	payload := fmt.Sprintf("novakey://pair?v=3&host=%s&port=%d&token=%s&fp=%s&exp=%d",
		advertiseHost, port, tokenB64, fp, exp.Unix())

	pngPath, err := writeAndOpenPairQR(".", payload)
	if err != nil {
		log.Printf("[pair] token id=%s expires=%s; QR at %s (viewer open failed: %v)",
			tokenID, exp.Format(time.RFC3339), pngPath, err)
	} else {
		log.Printf("[pair] token id=%s expires=%s; QR opened at %s",
			tokenID, exp.Format(time.RFC3339), pngPath)
	}
}
