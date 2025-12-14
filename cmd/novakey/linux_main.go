// cmd/novakey/linux_main.go
//go:build linux

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

	// Arm API (if enabled)
	startArmAPI()

	listenAddr := cfg.ListenAddr
	maxLen := cfg.MaxPayloadLen

	log.Printf("NovaKey (Linux) service starting (listener=%s)", listenAddr)

	ln, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		log.Fatalf("listen on %s: %v", listenAddr, err)
	}
	log.Printf("NovaKey (Linux) service listening on %s", listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[accept] error: %v", err)
			continue
		}
		reqID := nextReqID()
		go handleConn(reqID, conn, maxLen)
	}
}

func handleConn(reqID uint64, conn net.Conn, maxLen int) {
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

	// Serialize injection paths
	injectMu.Lock()
	defer injectMu.Unlock()

	// --- TWO-MAN: require recent approval for this device ---
	if cfg.TwoManEnabled {
		consume := *cfg.ApproveConsumeOnInject
		// default to true if unset in config loader
		if !approvalGate.Consume(deviceID, consume) {
			until := approvalGate.ApprovedUntil(deviceID)
			if until.IsZero() {
				logReqf(reqID, "blocked injection (two-man: not approved)")
			} else {
				logReqf(reqID, "blocked injection (two-man: approval expired at %s)", until.Format(time.RFC3339Nano))
			}

			// Optional clipboard behavior when blocked
			if *cfg.AllowClipboardWhenDisarmed {
				_ = trySetClipboard(password) // best-effort helper you already added elsewhere; if not, remove this line
				logReqf(reqID, "blocked injection (two-man); clipboard set (if available)")
			}
			return
		}
		logReqf(reqID, "two-man approval OK; proceeding")
	}

	// --- ARM GATE ---
	// Two-man implies arm must be open too, even if arm_enabled was false.
	if cfg.ArmEnabled || cfg.TwoManEnabled {
		ok := armGate.Consume(cfg.ArmConsumeOnInject)
		if !ok {
			logReqf(reqID, "blocked injection (not armed)")

			if *cfg.AllowClipboardWhenDisarmed {
				_ = trySetClipboard(password) // best-effort; remove if you don't have it
				logReqf(reqID, "blocked injection (not armed); clipboard set (if available)")
			}
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
