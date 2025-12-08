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

	addr := fmt.Sprintf("%s:%d",
		settings.Control.ListenAddress,
		settings.Control.ListenPort,
	)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		LogError("Control listener failed to start", err)
		return
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
		return
	}

	command := fields[0]
	token := fields[1]

	if token != settings.Control.Token {
		return
	}

	switch strings.ToUpper(command) {
	case "ARM":
		armOnce()
	}
}
