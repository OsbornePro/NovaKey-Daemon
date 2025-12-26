// cmd/novakey/pair_bootstrap.go
package main

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"
)

// maybeStartPairingBootstrap triggers pairing mode when no devices are paired.
// It generates/refreshes a one-time token and writes/opens a QR code that the phone app scans.
func maybeStartPairingBootstrap() {
	if isPaired() {
		return
	}

	if serverDecapKey == nil || len(serverEncapKey) == 0 {
		log.Printf("[pair] cannot start pairing: server keys not initialized")
		return
	}

	// Create/refresh a pairing token.
	tokenB64, tokenID, exp := startOrRefreshPairToken(10 * time.Minute)

	// Advertise the main listener address (single-port design).
	host, port := splitHostPortOrDie(cfg.ListenAddr)
	advHost := chooseAdvertiseHost(host)

	// Fingerprint of the server public key (client should verify this matches QR).
	fp16 := fp16Hex(serverEncapKey)

	// Build a compact URL payload that your iOS app can parse.
	// NOTE: This is a *QR payload format choice*. Keep stable once your app depends on it.
	u := url.URL{
		Scheme: "novakey",
		Host:   "pair",
	}
	q := u.Query()
	q.Set("v", "1")
	q.Set("host", advHost)
	q.Set("port", strconv.Itoa(port))
	q.Set("token", tokenB64)
	q.Set("fp16", fp16)
	q.Set("exp", strconv.FormatInt(exp.Unix(), 10))
	u.RawQuery = q.Encode()

	payload := u.String()

	// Write QR and open it (best-effort).
	pngPath, err := writeAndOpenPairQR(".", payload)
	if err != nil {
		log.Printf("[pair] QR written to %s but open failed: %v", pngPath, err)
	} else {
		log.Printf("[pair] opened QR at %s", pngPath)
	}

	log.Printf("[pair] pairing active id=%s expires=%s", tokenID, exp.Format(time.RFC3339))
	log.Printf("[pair] scan QR to pair: %s:%d (fp16=%s)", advHost, port, fp16)
	log.Printf("[pair] /pair route requires: 'NOVAK/1 /pair\\n' then hello JSON with token")
}

// Optional: for a stable human-readable string for logs.
func fmtHostPort(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}
