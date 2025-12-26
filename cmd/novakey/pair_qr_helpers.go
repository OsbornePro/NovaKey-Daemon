// cmd/novakey/pair_qr_helpers.go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// writeAndOpenPairQR writes a QR PNG to outDir/novakey-pair.png and tries to open it.
func writeAndOpenPairQR(outDir, payload string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	pngPath := filepath.Join(outDir, "novakey-pair.png")

	// Low density, easy scan
	if err := qrcode.WriteFile(payload, qrcode.Low, 512, pngPath); err != nil {
		return "", err
	}
	if err := openDefault(pngPath); err != nil {
		return pngPath, err
	}
	return pngPath, nil
}

func openDefault(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

// splitHostPortOrDie parses cfg.ListenAddr like "127.0.0.1:60768" or "60768".
// Returns (host, port). If only a port is given, host defaults to 127.0.0.1.
func splitHostPortOrDie(addr string) (string, int) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		// Allow passing just a port number as a convenience.
		if n, err2 := strconv.Atoi(strings.TrimSpace(addr)); err2 == nil {
			return "127.0.0.1", n
		}
		log.Fatalf("invalid listen_addr %q: %v", addr, err)
	}

	n, err := strconv.Atoi(p)
	if err != nil {
		log.Fatalf("invalid listen_addr port %q: %v", p, err)
	}
	return h, n
}

// chooseAdvertiseHost picks the host to embed in the QR code.
// If listenHost is blank or a loopback/any bind, it chooses a non-loopback IPv4 if available.
func chooseAdvertiseHost(listenHost string) string {
	h := strings.TrimSpace(listenHost)
	if h == "" {
		return firstNonLoopbackIPv4OrLocalhost()
	}
	if h == "0.0.0.0" || h == "127.0.0.1" || h == "::" || h == "::1" || strings.EqualFold(h, "localhost") {
		return firstNonLoopbackIPv4OrLocalhost()
	}
	return h
}

// firstNonLoopbackIPv4OrLocalhost returns the first non-loopback IPv4 address found,
// otherwise "127.0.0.1".
func firstNonLoopbackIPv4OrLocalhost() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, iface := range ifaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue
		}
		if (iface.Flags & net.FlagLoopback) != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String()
		}
	}
	return "127.0.0.1"
}

// randHex returns nBytes of crypto-random data, hex-encoded.
func randHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
