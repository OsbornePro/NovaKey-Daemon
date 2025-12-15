// cmd/novakey/windows_main.go
//go:build windows

package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

func allowClipboardWhenBlocked() bool {
	if cfg.AllowClipboardWhenDisarmed == nil {
		return false
	}
	return *cfg.AllowClipboardWhenDisarmed
}

func boolDeref(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

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

	log.Printf("NovaKey (Windows) starting (listener=%s)", listenAddr)

	ln, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		log.Fatalf("listen on %s: %v", listenAddr, err)
	}
	log.Printf("NovaKey (Windows) listening on %s", listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[accept] error: %v", err)
			continue
		}
		reqID := nextReqID()
		go handleConnWin(reqID, conn, maxLen)
	}
}

func handleConnWin(reqID uint64, conn net.Conn, maxLen int) {
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

	deviceID, msgType, payload, err := decryptMessageFrame(buf)
	if err != nil {
		logReqf(reqID, "decryptMessageFrame failed: %v", err)
		return
	}

	// --- TWO-MAN: approval control message (typed) ---
	if msgType == MsgTypeApprove {
		if !cfg.TwoManEnabled {
			logReqf(reqID, "approve message received but two_man_enabled=false; ignoring")
			return
		}
		until := approvalGate.Approve(deviceID, approveWindow())
		logReqf(reqID, "two-man approve received from device=%q; approved until %s",
			deviceID, until.Format(time.RFC3339Nano))
		return
	}

	if msgType != MsgTypeInject {
		logReqf(reqID, "unknown msgType=%d from device=%q; dropping", msgType, deviceID)
		return
	}

	password := payload
	logReqf(reqID, "decrypted password payload from device=%q: %s", deviceID, safePreview(password))

	// --- Filter unsafe injection text (newlines, max len, etc.) ---
	if err := validateInjectText(password); err != nil {
		logReqf(reqID, "blocked injection (unsafe text): %v", err)
		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
			} else {
				logReqf(reqID, "clipboard set (unsafe text blocked)")
			}
		}
		return
	}

	// --- TARGET POLICY (allow/deny list) ---
	// Do this BEFORE consuming approval/arm windows so a blocked focus can’t burn them.
	if err := enforceTargetPolicy(); err != nil {
		logReqf(reqID, "blocked injection (target policy): %v", err)
		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
			} else {
				logReqf(reqID, "blocked injection (target policy); clipboard set")
			}
		}
		return
	}

	// Serialize injection paths
	injectMu.Lock()
	defer injectMu.Unlock()

	// --- TWO-MAN: require recent approval for this device ---
	if cfg.TwoManEnabled {
		consume := boolDeref(cfg.ApproveConsumeOnInject, true)
		if !approvalGate.Consume(deviceID, consume) {
			until := approvalGate.ApprovedUntil(deviceID)
			if until.IsZero() {
				logReqf(reqID, "blocked injection (two-man: not approved)")
			} else {
				logReqf(reqID, "blocked injection (two-man: approval expired at %s)", until.Format(time.RFC3339Nano))
			}
			if allowClipboardWhenBlocked() {
				if err2 := trySetClipboard(password); err2 != nil {
					logReqf(reqID, "clipboard set failed: %v", err2)
				} else {
					logReqf(reqID, "blocked injection (two-man); clipboard set")
				}
			}
			return
		}
		logReqf(reqID, "two-man approval OK; proceeding")
	}

	// --- ARM GATE ---
	if cfg.ArmEnabled || cfg.TwoManEnabled {
		consume := boolDeref(cfg.ArmConsumeOnInject, true)
		ok := armGate.Consume(consume)
		if !ok {
			logReqf(reqID, "blocked injection (not armed)")
			if allowClipboardWhenBlocked() {
				if err2 := trySetClipboard(password); err2 != nil {
					logReqf(reqID, "clipboard set failed: %v", err2)
				} else {
					logReqf(reqID, "blocked injection (not armed); clipboard set")
				}
			}
			return
		}
		logReqf(reqID, "armed gate open; proceeding with injection")
	}

	if err := InjectPasswordToFocusedControl(password); err != nil {
		logReqf(reqID, "InjectPasswordToFocusedControl error: %v", err)
		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
			} else {
				logReqf(reqID, "injection failed; clipboard set")
			}
		}
		return
	}

	logReqf(reqID, "injection complete")
}

