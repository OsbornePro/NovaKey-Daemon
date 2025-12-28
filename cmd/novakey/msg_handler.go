// cmd/novakey/msg_handler.go
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
	"errors"
)

// handleMsgConn is used by router.go for "/msg" and legacy clients.
// It owns the connection and must close it.
func handleMsgConn(conn net.Conn) error {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	reqID := nextReqID()
	remote := conn.RemoteAddr().String()
	logReqf(reqID, "connection opened from %s", remote)

	respond := func(st RespStatus, msg string) {
		writeReplyLine(conn, ServerReply{
			Status: uint8(st),
			Msg:    msg,
		})
	}
	respondX := func(st RespStatus, msg, stage string, reason ReplyReason, details map[string]any) {
		writeReplyLine(conn, ServerReply{
			Status:  uint8(st),
			Msg:     msg,
			Stage:   stage,
			Reason:  reason,
			Details: details,
		})
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

	// Decrypt FIRST. Never branch on msgType until err == nil.
	deviceID, msgType, payload, err := decryptMessageFrame(buf)
	if err != nil {
		logReqf(reqID, "decryptMessageFrame failed: %v", err)
		respond(StatusCryptoFail, "decrypt/auth failed")
		return nil
	}

	// Now safe to route by msgType.
	switch msgType {

	case MsgTypeArm:
		// payload is optional JSON: {"ms":15000}
		ms := cfg.ArmDurationMs
		if len(payload) > 0 {
			var obj struct{ MS int `json:"ms"` }
			if err := json.Unmarshal(payload, &obj); err == nil && obj.MS > 0 {
				ms = obj.MS
			}
		}
		armGate.ArmFor(time.Duration(ms) * time.Millisecond)
		respond(StatusOK, fmt.Sprintf("armed_for_ms=%d", ms))
		return nil

	case MsgTypeDisarm:
		armGate.Disarm()
		respond(StatusOK, "disarmed")
		return nil

	case MsgTypeApprove:
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

	case MsgTypeInject:
		// continue below

	default:
		logReqf(reqID, "unknown msgType=%d from device=%q; dropping", msgType, deviceID)
		respond(StatusBadRequest, "unknown msgType")
		return nil
	}

	// ---- INJECT path (same as your existing code) ----
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
				respondX(StatusOKClipboard, "clipboard set (unsafe text blocked)", "inject", ReasonClipboardFallback,
					map[string]any{"blocked": "unsafe_text"},
				)
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
				respondX(StatusOKClipboard, "clipboard set (target policy blocked)", "inject", ReasonClipboardFallback,
					map[string]any{"blocked": "target_policy"},
				)
			}
			return nil
		}

		respond(StatusBadRequest, "target policy blocked")
		return nil
	}

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
					respondX(StatusOKClipboard, "clipboard set (needs approve)", "inject", ReasonClipboardFallback,
						map[string]any{"blocked": "needs_approve"},
					)
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
					respondX(StatusOKClipboard, "clipboard set (not armed)", "inject", ReasonClipboardFallback,
						map[string]any{"blocked": "not_armed"},
					)
				}
				return nil
			}

			respond(StatusNotArmed, "not armed")
			return nil
		}
		logReqf(reqID, "armed gate open; proceeding with injection")
	}

	if err := InjectPasswordToFocusedControl(password); err != nil {
		logReqf(reqID, "InjectPasswordToFocusedControl error: %v", err)

		// Clipboard fallback policy:
		// - If Wayland sentinel error: clipboard fallback counts as SUCCESS (StatusOKClipboard).
		// - Otherwise: clipboard fallback is a helpful side-effect, but overall is NOT success.
		if allowClipboardOnInjectFailure() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusInternal, "inject failed; clipboard failed")
				return nil
			}

			// Success Wayland: treat clipboard as the intended “delivery”
			if errors.Is(err, ErrInjectUnavailableWayland) {
				respond(StatusOKClipboard, "clipboard set (wayland; paste to insert)")
				return nil
			}

			// Error Non-Wayland inject failure: do NOT report overall success
			respondX(StatusOKClipboard, "clipboard set (wayland; paste to insert)", "inject", ReasonInjectUnavailableWayland,
				map[string]any{"session": "wayland"},
			)

			return nil
		}
		respond(StatusInternal, "inject failed")
		return nil
	}
	logReqf(reqID, "injection complete")
	respond(StatusOK, "ok")
	return nil
}
