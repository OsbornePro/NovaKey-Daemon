//go:build windows
// +build windows

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

	// 🔑 Start hotkey listener (REQUIRED)
	if settings.Arming.AllowHotkey {
		go startHotkeyListener()
	}

	priv, pub, err := GenerateKeyPair()
	if err != nil {
		LogError("Key generation failed", err)
		return
	}

	// ===== OPTION B: Ephemeral server keypair =====
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

	addr4 := fmt.Sprintf("%s:%d",
		settings.Network.ListenAddress,
		settings.Network.ListenPort,
	)

	var listeners []net.Listener

	switch settings.Network.Mode {
	case "ipv4":
		ln, err := net.Listen("tcp4", addr4)
		if err != nil {
			LogError("Failed to start IPv4 listener", err)
			return
		}
		listeners = append(listeners, ln)

	case "ipv6":
		ln, err := net.Listen("tcp6", ":"+fmt.Sprint(settings.Network.ListenPort))
		if err != nil {
			LogError("Failed to start IPv6 listener", err)
			return
		}
		listeners = append(listeners, ln)

	case "dual":
		if ln4, err := net.Listen("tcp4", addr4); err == nil {
			listeners = append(listeners, ln4)
		}
		if ln6, err := net.Listen("tcp6", ":"+fmt.Sprint(settings.Network.ListenPort)); err == nil {
			listeners = append(listeners, ln6)
		}
		if len(listeners) == 0 {
			LogError("Failed to start listeners in dual mode", nil)
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
