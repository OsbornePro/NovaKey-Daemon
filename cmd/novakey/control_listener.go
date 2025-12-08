package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func startControlListener() {
	if !settings.Control.Enabled {
		LogInfo("Control listener disabled by config")
		return
	}

	if settings.Control.Token == "" {
		LogError("Control listener disabled: missing token", nil)
		return
	}

	listenAddr := settings.Control.ListenAddress
	if strings.TrimSpace(listenAddr) == "" {
		// Fail-safe default: bind to localhost only.
		listenAddr = "127.0.0.1"
	}

	addr := fmt.Sprintf("%s:%d", listenAddr, settings.Control.ListenPort)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		LogError("Control listener failed to start", err)
		return
	}

	if listenAddr == "0.0.0.0" || listenAddr == "::" {
		LogError("Control listener is bound to a wildcard address; this is unsafe in most environments", nil)
	}

	LogInfo("Control listener active on " + addr)

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				// Noisy log here could be spammy; keep quiet on transient errors.
				continue
			}
			go handleControlConn(conn)
		}
	}()
}

func handleControlConn(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}

	line := strings.TrimSpace(scanner.Text())
	fields := strings.Fields(line)

	if len(fields) != 2 {
		// Require "COMMAND TOKEN"
		LogError("Control command rejected: malformed line", nil)
		return
	}

	command := strings.ToUpper(fields[0])
	token := fields[1]

	if token != settings.Control.Token {
		LogError("Control command rejected: invalid token", nil)
		return
	}

	switch command {
	case "ARM":
		armOnce()
	case "DISARM":
		disarm()
	default:
		LogError("Control command rejected: unknown command "+command, nil)
	}
}
