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

	listenAddr := strings.TrimSpace(settings.Control.ListenAddress)
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}

	addr := fmt.Sprintf("%s:%d", listenAddr, settings.Control.ListenPort)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		LogError("Control listener failed to start", err)
		return
	}

	if listenAddr == "0.0.0.0" || listenAddr == "::" {
		LogError("WARNING: control listener bound to wildcard address", nil)
	}

	LogInfo("Control listener active on " + addr)

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
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
		LogError("Control command rejected: malformed input", nil)
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
		arm()
	case "DISARM":
		disarm()
	case "RELOAD":
		if err := reloadSettings(); err != nil {
			LogError("Config reload failed via control channel", err)
			return
		}
		LogInfo("Config reloaded via control channel")
	default:
		LogError("Unknown control command: "+command, nil)
	}
}
