//go:build windows
// +build windows

package main

import (
	"net"
	"os"
	"os/signal"
)

func main() {
	priv, _, err := GenerateKeyPair()
	if err != nil {
		LogError("Key generation failed", err)
		return
	}

	addr := ":60768"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		LogError("Failed to start TCP server", err)
		return
	}
	defer ln.Close()

	LogInfo("TCP listener started on " + addr)

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
