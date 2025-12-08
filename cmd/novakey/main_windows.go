//go:build windows
// +build windows

package main

import (
	"net"
	"os"
	"os/signal"
	"path/filepath"
)

func main() {
	// Generate Kyber keypair
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		LogError("Key generation failed", err)
		return
	}

	// TEMPORARY: export public key for novakey-send testing
	// This writes server.pub next to the executable (safe for Windows services)
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		pubPath := filepath.Join(exeDir, "server.pub")

		if pubBytes, err := pub.MarshalBinary(); err == nil {
			if err := os.WriteFile(pubPath, pubBytes, 0600); err == nil {
				LogInfo("Exported server public key to " + pubPath)
			} else {
				LogError("Failed to write server.pub", err)
			}
		} else {
			LogError("Failed to marshal public key", err)
		}
	} else {
		LogError("Failed to determine executable path", err)
	}
	// TEMPORARY END BLOCK

	addr := ":60768"
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		LogError("Failed to start TCP server", err)
		return
	}
	defer ln.Close()

	LogInfo("TCP listener started on " + addr + " (IPv4 only)")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				LogError("Accept error", err)
				continue
			}
			go handleConn(conn, priv)
		}
	}()

	<-stop
	LogInfo("Stopping server")
}
