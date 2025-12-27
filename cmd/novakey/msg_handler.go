// cmd/novakey/msg_handler.go
package main

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
    "strconv"
	"time"
)

// handleMsgConn is used by router.go for "/msg" and legacy clients.
// It owns the connection and must close it.
func handleMsgConn(conn net.Conn) error {
	defer conn.Close()

	// Hard timeout for clients that connect and stall.
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	// Clear deadlines before returning (best practice).
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	reqID := nextReqID()
	remote := conn.RemoteAddr().String()
	logReqf(reqID, "connection opened from %s", remote)
    respond := func(st RespStatus, msg string) {
        logReqf(reqID, "responding status=%d msg=%q", st, msg)

        // JSON line response
        b, _ := json.Marshal(map[string]any{
            "status":  uint8(st),
            "message": msg,
        })
        b = append(b, '\n')
        _, _ = conn.Write(b)
    }

	maxLen := cfg.MaxPayloadLen

	var length uint16
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		if err != io.EOF {
			logReqf(reqID, "read length failed: %v", err)
			respond(StatusBadRequest, "read length failed")
		} else {
			logReqf(reqID, "client closed connection before sending length")
			respond(StatusBadRequest, "client closed before length")
		}
		return nil
	}
	logReqf(reqID, "declared payload length=%d", length)

	if length == 0 || int(length) > maxLen {
		logReqf(reqID, "invalid length (%d), max=%d", length, maxLen)
		respond(StatusBadRequest, "invalid length")
		return nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		logReqf(reqID, "read payload failed: %v", err)
		respond(StatusBadRequest, "read payload failed")
		return nil
	}

	// v3 outer frame -> typed inner message frame.
	deviceID, msgType, payload, err := decryptMessageFrame(buf)
    // --- ARM message ---
    if msgType == MsgTypeArm {
        if !cfg.ArmEnabled {
            respond(StatusBadRequest, "arm disabled")
            return nil
        }

        // Duration: default from config, override allowed.
        ms := cfg.ArmDurationMs
        if len(payload) > 0 {
            if n, err := strconv.Atoi(string(payload)); err == nil && n > 0 && n <= 300000 {
                ms = n // cap at 5 min for safety
            }
        }

        armGate.ArmFor(time.Duration(ms) * time.Millisecond)
        respond(StatusOK, fmt.Sprintf("armed_for_ms=%d", ms))
        return nil
    }

	if err != nil {
		logReqf(reqID, "decryptMessageFrame failed: %v", err)
		respond(StatusCryptoFail, "decrypt/auth failed")
		return nil
	}

	// --- TWO-MAN: approval control message (typed) ---
	if msgType == MsgTypeApprove {
		if !cfg.TwoManEnabled {
			logReqf(reqID, "approve message received but two_man_enabled=false; ignoring")
			respond(StatusBadRequest, "two-man disabled; approve ignored")
			return nil
		}
		until := approvalGate.Approve(deviceID, approveWindow())
		logReqf(reqID, "two-man approve received from device=%q; approved until %s",
			deviceID, until.Format(time.RFC3339Nano))
		respond(StatusOK, "approved")
		return nil
	}

	// --- INJECT message ---
	if msgType != MsgTypeInject {
		logReqf(reqID, "unknown msgType=%d from device=%q; dropping", msgType, deviceID)
		respond(StatusBadRequest, "unknown msgType")
		return nil
	}

	password := string(payload)
	logReqf(reqID, "decrypted password payload from device=%q: %s", deviceID, safePreview(password))

	// Unsafe-text filter
	if err := validateInjectText(password); err != nil {
		logReqf(reqID, "blocked injection (unsafe text): %v", err)

		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusBadRequest, "unsafe text; clipboard failed")
			} else {
				logReqf(reqID, "clipboard set (unsafe text blocked)")
				respond(StatusBadRequest, "unsafe text; clipboard set")
			}
			return nil
		}

		respond(StatusBadRequest, "unsafe text")
		return nil
	}

	// Target policy (do BEFORE consuming gates)
	if err := enforceTargetPolicy(); err != nil {
		logReqf(reqID, "blocked injection (target policy): %v", err)

		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusBadRequest, "target policy; clipboard failed")
			} else {
				logReqf(reqID, "blocked injection (target policy); clipboard set")
				respond(StatusBadRequest, "target policy; clipboard set")
			}
			return nil
		}

		respond(StatusBadRequest, "target policy blocked")
		return nil
	}

	// Serialize injection paths (shared global mutex)
	injectMu.Lock()
	defer injectMu.Unlock()

	// Two-man gate
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
					respond(StatusNeedsApprove, "needs approve; clipboard failed")
				} else {
					logReqf(reqID, "blocked injection (two-man); clipboard set")
					respond(StatusNeedsApprove, "needs approve; clipboard set")
				}
				return nil
			}

			respond(StatusNeedsApprove, "needs approve")
			return nil
		}
		logReqf(reqID, "two-man approval OK; proceeding")
	}

	// Arm gate
	if cfg.ArmEnabled {
		consume := boolDeref(cfg.ArmConsumeOnInject, true)
		if !armGate.Consume(consume) {
			logReqf(reqID, "blocked injection (not armed)")

			if allowClipboardWhenBlocked() {
				if err2 := trySetClipboard(password); err2 != nil {
					logReqf(reqID, "clipboard set failed: %v", err2)
					respond(StatusNotArmed, "not armed; clipboard failed")
				} else {
					logReqf(reqID, "blocked injection (not armed); clipboard set")
					respond(StatusNotArmed, "not armed; clipboard set")
				}
				return nil
			}

			respond(StatusNotArmed, "not armed")
			return nil
		}
		logReqf(reqID, "armed gate open; proceeding with injection")
	}

	// Inject (platform-specific implementation behind this symbol)
	if err := InjectPasswordToFocusedControl(password); err != nil {
		logReqf(reqID, "InjectPasswordToFocusedControl error: %v", err)

		// IMPORTANT: this is "inject failed after gates passed" behavior,
		// intended for Wayland and similar environments.
		if allowClipboardOnInjectFailure() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusInternal, "inject failed; clipboard failed")
			} else {
				logReqf(reqID, "injection failed; clipboard set")
				respond(StatusOKClipboard, "clipboard set (inject unavailable)")
			}
			return nil
		}

		respond(StatusInternal, "inject failed")
		return nil
	}

	logReqf(reqID, "injection complete")
	respond(StatusOK, "ok")
	return nil
}
