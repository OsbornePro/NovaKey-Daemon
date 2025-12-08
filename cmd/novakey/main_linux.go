//go:build linux
// +build linux

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
)

func main() {
	loadSettings()

	priv, _, err := GenerateKeyPair()
	if err != nil {
		LogError("Key generation failed", err)
		return
	}

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
		ln4, err4 := net.Listen("tcp4", addrV4)
		if err4 == nil {
			listeners = append(listeners, ln4)
			LogInfo("Listening on IPv4 " + addrV4)
		}

		ln6, err6 := net.Listen("tcp6", addrV6)
		if err6 == nil {
			listeners = append(listeners, ln6)
			LogInfo("Listening on IPv6 " + addrV6)
		}

		if len(listeners) == 0 {
			LogError("Failed to start any listeners in dual mode", nil)
			return
		}
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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	LogInfo("Stopping server")
}
