// linux_main.go
//go:build !windows

package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
)

const (
	listenAddr = "0.0.0.0:60768"
	maxTextLen = 4096
)

func main() {
	log.Printf("NovaKey service starting (listener=%s)", listenAddr)

	ln, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		log.Fatalf("listen on %s: %v", listenAddr, err)
	}
	log.Printf("NovaKey service listening on %s", listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[accept] error: %v", err)
			continue
		}
		reqID := nextReqID()
		go handleConn(reqID, conn)
	}
}

func handleConn(reqID uint64, conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	logReqf(reqID, "connection opened from %s", remote)

	// 1) Read 2-byte length (big-endian)
	var length uint16
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		if err != io.EOF {
			logReqf(reqID, "read length failed: %v", err)
		} else {
			logReqf(reqID, "client closed connection before sending length")
		}
		return
	}
	logReqf(reqID, "declared payload length=%d", length)

	if length == 0 || int(length) > maxTextLen {
		logReqf(reqID, "invalid length (%d), max=%d", length, maxTextLen)
		return
	}

	// 2) Read payload
	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		logReqf(reqID, "read payload failed: %v", err)
		return
	}
	text := string(buf)
	logReqf(reqID, "payload received: %s", safePreview(text))

	// 3) Inject
    injectMu.Lock()
    defer injectMu.Unlock()
	if err := InjectPasswordToFocusedControl(text); err != nil {
		logReqf(reqID, "InjectPasswordToFocusedControl error: %v", err)
		return
	}

	logReqf(reqID, "injection complete")
}

