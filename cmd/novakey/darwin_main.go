// cmd/novakey/darwin_main.go
//go:build darwin

package main

import (
    "encoding/binary"
    "io"
    "log"
    "net"
)

const (
    listenAddrDarwin = "127.0.0.1:60768"
    maxTextLenDarwin = 4096
)

func main() {
    if err := initCrypto(); err != nil {
        log.Fatalf("initCrypto failed: %v", err)
    }

    log.Printf("NovaKey (macOS) starting (listener=%s)", listenAddrDarwin)

    ln, err := net.Listen("tcp4", listenAddrDarwin)
    if err != nil {
        log.Fatalf("listen on %s: %v", listenAddrDarwin, err)
    }
    log.Printf("NovaKey (macOS) listening on %s", listenAddrDarwin)

    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Printf("[accept] error: %v", err)
            continue
        }
        reqID := nextReqID()
        go handleConnDarwin(reqID, conn)
    }
}

func handleConnDarwin(reqID uint64, conn net.Conn) {
    defer conn.Close()
    remote := conn.RemoteAddr().String()
    logReqf(reqID, "connection opened from %s", remote)

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

    if length == 0 || int(length) > maxTextLenDarwin {
        logReqf(reqID, "invalid length (%d), max=%d", length, maxTextLenDarwin)
        return
    }

    buf := make([]byte, length)
    if _, err := io.ReadFull(conn, buf); err != nil {
        logReqf(reqID, "read payload failed: %v", err)
        return
    }

    password, err := decryptPasswordFrame(buf)
    if err != nil {
        logReqf(reqID, "decryptPasswordFrame failed: %v", err)
        return
    }
    logReqf(reqID, "decrypted password payload: %s", safePreview(password))

    injectMu.Lock()
    defer injectMu.Unlock()

    if err := InjectPasswordToFocusedControl(password); err != nil {
        logReqf(reqID, "InjectPasswordToFocusedControl error: %v", err)
        return
    }

    logReqf(reqID, "injection complete")
}

