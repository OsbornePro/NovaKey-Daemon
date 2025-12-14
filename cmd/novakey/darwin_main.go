// cmd/novakey/darwin_main.go
//go:build darwin

package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("loadConfig failed: %v", err)
	}
	if err := initCrypto(); err != nil {
		log.Fatalf("initCrypto failed: %v", err)
	}
	startArmAPI()

	listenAddr := cfg.ListenAddr
	maxLen := cfg.MaxPayloadLen

	log.Printf("NovaKey (macOS) service starting (listener=%s)", listenAddr)

	ln, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		log.Fatalf("listen on %s: %v", listenAddr, err)
	}
	log.Printf("NovaKey (macOS) listening on %s", listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[accept] error: %v", err)
			continue
		}
		reqID := nextReqID()
		go handleConnDarwin(reqID, conn, maxLen)
	}
}

func handleConnDarwin(reqID uint64, conn net.Conn, maxLen int) {
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

	if length == 0 || int(length) > maxLen {
		logReqf(reqID, "invalid length (%d), max=%d", length, maxLen)
		return
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		logReqf(reqID, "read payload failed: %v", err)
		return
	}

	deviceID, password, err := decryptPasswordFrame(buf)
	if err != nil {
		logReqf(reqID, "decryptPasswordFrame failed: %v", err)
		return
	}

	// --- TWO-MAN: approval control message ---
	if cfg.TwoManEnabled && isApproveControlPayload(password) {
		until := approvalGate.Approve(deviceID, approveWindow())
		logReqf(reqID, "two-man approve received from device=%q; approved until %s",
			deviceID, until.Format(time.RFC3339Nano))
		return
	}

	logReqf(reqID, "decrypted password payload from device=%q: %s", deviceID, safePreview(password))

	// --- Filter unsafe injection text (newlines, max len, etc.) ---
	if err := validateInjectText(password); err != nil {
		logReqf(reqID, "blocked injection (unsafe text): %v", err)
		return
	}
    // --- Process whitelist ---
    if err := enforceTargetPolicy(); err != nil {
        logReqf(reqID, "blocked injection (target policy): %v", err)
        return
    }

	injectMu.Lock()
	defer injectMu.Unlock()

	// --- TWO-MAN: require recent approval for this device ---
	if cfg.TwoManEnabled {
		consume := *cfg.ApproveConsumeOnInject
		if !approvalGate.Consume(deviceID, consume) {
			until := approvalGate.ApprovedUntil(deviceID)
			if until.IsZero() {
				logReqf(reqID, "blocked injection (two-man: not approved)")
			} else {
				logReqf(reqID, "blocked injection (two-man: approval expired at %s)", until.Format(time.RFC3339Nano))
			}
			return
		}
		logReqf(reqID, "two-man approval OK; proceeding")
	}

	// --- ARM GATE ---
	// Two-man implies arm must also be open.
	if cfg.ArmEnabled || cfg.TwoManEnabled {
		ok := armGate.Consume(*cfg.ArmConsumeOnInject)
		if !ok {
			logReqf(reqID, "blocked injection (not armed)")
			return
		}
		logReqf(reqID, "armed gate open; proceeding with injection")
	}

	if err := InjectPasswordToFocusedControl(password); err != nil {
		logReqf(reqID, "InjectPasswordToFocusedControl error: %v", err)
		return
	}

	logReqf(reqID, "injection complete")
}
