//go:build darwin
// +build darwin

package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/signal"
)

func main() {
	loadSettings()

	priv, pub, err := GenerateKeyPair()
	if err != nil {
		LogError("Key generation failed", err)
		return
	}

	// ===== OPTION B: Ephemeral server keypair =====
	// Print server public key for pairing (valid for this run only)
	pubBytes, err := pub.MarshalBinary()
	if err != nil {
		LogError("Failed to marshal server public key", err)
		return
	}
	fmt.Println("=== NovaKey Server Public Key (session-only) ===")
	fmt.Println(base64.StdEncoding.EncodeToString(pubBytes))
	fmt.Println("================================================")

	// ----- OPTION A (COMMENTED OUT FOR FUTURE) -----
	// loadOrCreateServerKeyPair()
	// priv, pub = loadedPriv, loadedPub
	// ----------------------------------------------

	var listeners []net.Listener

	addrV4 := fmt.Sprintf("%s:%d", settings.Network.ListenAddress, settings.Network.ListenPort)
	addrV6 := fmt.Sprintf(":%d", settings.Network.ListenPort)

	switch settings.Network.Mode {
	case "ipv4":
		ln, err := net.Listen("tcp4", addrV4)
		if err != nil {
			LogError("Failed to start IPv4 listener", err)
			return
		}
		listeners = append(listeners, ln)
		LogInfo("Listening on IPv4 " + addrV4)

	case "ipv6":
		ln, err := net.Listen("tcp6", addrV6)
		if err != nil {
			LogError("Failed to start IPv6 listener", err)
			return
		}
		listeners = append(listeners, ln)
		LogInfo("Listening on IPv6 " + addrV6)

	case "dual":
		if ln4, err4 := net.Listen("tcp4", addrV4); err4 == nil {
			listeners = append(listeners, ln4)
			LogInfo("Listening on IPv4 " + addrV4)
		}
		if ln6, err6 := net.Listen("tcp6", addrV6); err6 == nil {
			listeners = append(listeners, ln6)
			LogInfo("Listening on IPv6 " + addrV6)
		}
		if len(listeners) == 0 {
			LogError("Failed to start any listeners in dual mode", nil)
			return
		}

	default:
		LogError("Invalid network mode: "+settings.Network.Mode, nil)
		return
	}

	for _, ln := range listeners {
		go func(l net.Listener) {
			defer l.Close()
			for {
				conn, err := l.Accept()
				if err != nil {
					continue
				}
				go handleConn(conn, priv)
			}
		}(ln)
	}

	LogInfo("NovaKey server running")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	LogInfo("Stopping server")
}
