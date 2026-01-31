// cmd/novakey/msg_handler.go
package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type execEnvelope struct {
    Op    string         `json:"op"`
    Action string        `json:"action"`
    Params map[string]any `json:"params"`
    Req   string         `json:"req"`
    TTLMS int            `json:"ttl_ms"`
}
// handleMsgConn is used by router.go for "/msg".
// It owns the connection and must close it.
func handleMsgConn(conn net.Conn) error {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	reqID := nextReqID()
	remote := conn.RemoteAddr().String()
	logReqf(reqID, "connection opened from %s", remote)

	// ALWAYS reply with ONE newline-terminated JSON line (machine-readable).
	respond := func(st RespStatus, stage ReplyStage, reason ReplyReason, msg string) {
		writeReplyLine(conn, makeReply(reqID, st, stage, reason, msg))
	}

	maxLen := cfg.MaxPayloadLen

	// ---- Read length ----
	var length uint16
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		if err != io.EOF {
			logReqf(reqID, "read length failed: %v", err)
			respond(StatusBadRequest, StageMsg, ReasonBadRequest, "read length failed")
		} else {
			logReqf(reqID, "client closed connection before sending length")
			respond(StatusBadRequest, StageMsg, ReasonBadRequest, "client closed before length")
		}
		return nil
	}
	logReqf(reqID, "declared payload length=%d", length)

	if length == 0 || int(length) > maxLen {
		logReqf(reqID, "invalid length (%d), max=%d", length, maxLen)
		respond(StatusBadRequest, StageMsg, ReasonBadRequest, "invalid length")
		return nil
	}

	// ---- Read payload ----
	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		logReqf(reqID, "read payload failed: %v", err)
		respond(StatusBadRequest, StageMsg, ReasonBadRequest, "read payload failed")
		return nil
	}

	// ---- Decrypt FIRST. Never branch on msgType until err == nil. ----
	deviceID, msgType, payload, err := decryptMessageFrame(buf)
	if err != nil {
		logReqf(reqID, "decryptMessageFrame failed: %v", err)
		respond(StatusCryptoFail, StageMsg, ReasonCryptoFail, "decrypt/auth failed")
		return nil
	}

	// ---- Route by msgType ----
	switch msgType {

	case MsgTypeArm:
		// payload is optional JSON: {"ms":15000}
		ms := cfg.ArmDurationMs
		if len(payload) > 0 {
			var obj struct {
				MS int `json:"ms"`
			}
			if err := json.Unmarshal(payload, &obj); err == nil && obj.MS > 0 {
				ms = obj.MS
			}
		}
		armGate.ArmFor(time.Duration(ms) * time.Millisecond)
		respond(StatusOK, StageArm, ReasonOK, fmt.Sprintf("armed_for_ms=%d", ms))
		return nil

	case MsgTypeDisarm:
		armGate.Disarm()
		respond(StatusOK, StageDisarm, ReasonOK, "disarmed")
		return nil

	case MsgTypeApprove:
		// two_man_enabled defaults true via boolDeref(..., true)
		if !boolDeref(cfg.TwoManEnabled, true) {
			logReqf(reqID, "approve message received but two_man_enabled=false; ignoring")
			respond(StatusBadRequest, StageApprove, ReasonBadRequest, "two-man disabled; approve ignored")
			return nil
		}
		until := approvalGate.Approve(deviceID, approveWindow())
		logReqf(reqID, "two-man approve received from device=%q; approved until %s",
			deviceID, until.Format(time.RFC3339Nano))
		respond(StatusOK, StageApprove, ReasonOK, "approved")
		return nil

	case MsgTypeInject:
		// continue below

	default:
		logReqf(reqID, "unknown msgType=%d from device=%q; dropping", msgType, deviceID)
		respond(StatusBadRequest, StageMsg, ReasonBadRequest, "unknown msgType")
		return nil
	}

	// ---- INJECT path ----
	password := string(payload)

    // If payload looks like an exec request, route to runner (no shell).
    if cfg.ActionsEnabled && len(payload) > 0 && payload[0] == '{' {
        var env execEnvelope
        if err := json.Unmarshal(payload, &env); err == nil && env.Op == "exec" && env.Action != "" && env.Req != "" {
            // Enforce gates (same posture as injection)
            if boolDeref(cfg.TwoManEnabled, true) {
                consume := boolDeref(cfg.ApproveConsumeOnAction, true)
                if !approvalGate.Consume(deviceID, consume) {
                    respond(StatusNeedsApprove, StageInject, ReasonNeedsApprove, "needs approve")
                    return nil
                }
            }

            consumeArm := boolDeref(cfg.ArmConsumeOnAction, true)
            if !armGate.Consume(consumeArm) {
                respond(StatusNotArmed, StageInject, ReasonNotArmed, "not armed")
                return nil
            }

            // Call runner
            if runnerClient == nil {
                respond(StatusInternal, StageInject, ReasonInternal, "runner not configured")
                return nil
            }

            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()

            rresp, err := runnerClient.Exec(ctx, env.Req, env.Action, env.Params, deviceID, remote)
            if err != nil {
                logReqf(reqID, "runner exec failed: %v", err)
                respond(StatusInternal, StageInject, ReasonInternal, "exec failed")
                return nil
            }
            if !rresp.OK {
                // Keep client-safe reason; include details in msg
                respond(StatusInternal, StageInject, ReasonInternal,
                    fmt.Sprintf("exec error action=%s exit=%d err=%s", env.Action, rresp.ExitCode, rresp.Error))
                return nil
            }

            respond(StatusOK, StageInject, ReasonOK,
                fmt.Sprintf("exec ok action=%s exit=%d", env.Action, rresp.ExitCode))
            return nil
        }
    }

	logReqf(reqID, "decrypted payload from device=%q (len=%d)", deviceID, len(payload))

	// Unsafe-text filter
	if err := validateInjectText(password); err != nil {
		logReqf(reqID, "blocked injection (unsafe text): %v", err)

		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusBadRequest, StageInject, ReasonBadRequest, "unsafe text; clipboard failed")
			} else {
				logReqf(reqID, "clipboard set (unsafe text blocked)")
				respond(StatusOKClipboard, StageInject, ReasonClipboardFallback, "clipboard set (unsafe text blocked)")
			}
			return nil
		}

		respond(StatusBadRequest, StageInject, ReasonBadRequest, "unsafe text")
		return nil
	}

	// Target policy (do BEFORE consuming gates)
	if err := enforceTargetPolicy(); err != nil {
		logReqf(reqID, "blocked injection (target policy): %v", err)

		// Wayland: focused app detection is not implemented, so target policy cannot be evaluated.
		// Return a stable reply so clients can handle it cleanly.
		xdg := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))
		if xdg == "wayland" || os.Getenv("WAYLAND_DISPLAY") != "" {
			if allowClipboardWhenBlocked() {
				if err2 := trySetClipboard(password); err2 != nil {
					logReqf(reqID, "clipboard set failed: %v", err2)
					respond(StatusBadRequest, StageInject, ReasonBadRequest, "target policy unavailable on wayland; clipboard failed")
				} else {
					logReqf(reqID, "target policy unavailable on wayland; clipboard set")
					respond(StatusOKClipboard, StageInject, ReasonClipboardFallback, "clipboard set (target policy unavailable on wayland)")
				}
				return nil
			}

			respond(StatusBadRequest, StageInject, ReasonBadRequest,
				"target policy unavailable on wayland (disable target_policy_enabled, run under X11/Xwayland, or enable allow_clipboard_when_disarmed)")
			return nil
		}

		// Normal target policy denial (or other focused-target error)
		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusBadRequest, StageInject, ReasonBadRequest, "target policy blocked; clipboard failed")
			} else {
				logReqf(reqID, "blocked injection (target policy); clipboard set")
				respond(StatusOKClipboard, StageInject, ReasonClipboardFallback, "clipboard set (target policy blocked)")
			}
			return nil
		}

		respond(StatusBadRequest, StageInject, ReasonBadRequest, "target policy blocked")
		return nil
	}

	// Serialize injection to avoid overlapping OS-level input / clipboard behavior.
	injectMu.Lock()
	defer injectMu.Unlock()

	// Two-man gate
	if boolDeref(cfg.TwoManEnabled, true) {
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
					respond(StatusNeedsApprove, StageInject, ReasonNeedsApprove, "needs approve; clipboard failed")
				} else {
					logReqf(reqID, "blocked injection (two-man); clipboard set")
					respond(StatusOKClipboard, StageInject, ReasonClipboardFallback, "clipboard set (needs approve)")
				}
				return nil
			}

			respond(StatusNeedsApprove, StageInject, ReasonNeedsApprove, "needs approve")
			return nil
		}
		logReqf(reqID, "two-man approval OK; proceeding")
	}

	// Arm gate (always enforced; controlled by arm_duration_ms + arm_consume_on_inject)
	consumeArm := boolDeref(cfg.ArmConsumeOnInject, true)
	if !armGate.Consume(consumeArm) {
		logReqf(reqID, "blocked injection (not armed)")

		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusNotArmed, StageInject, ReasonNotArmed, "not armed; clipboard failed")
			} else {
				logReqf(reqID, "blocked injection (not armed); clipboard set")
				respond(StatusOKClipboard, StageInject, ReasonClipboardFallback, "clipboard set (not armed)")
			}
			return nil
		}

		respond(StatusNotArmed, StageInject, ReasonNotArmed, "not armed")
		return nil
	}
	logReqf(reqID, "armed gate open; proceeding with injection")

	// Perform injection (now returns method + err)
	method, err := InjectPasswordToFocusedControl(password)
	if err != nil {
		logReqf(reqID, "InjectPasswordToFocusedControl error: %v", err)

		if allowClipboardOnInjectFailure() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusInternal, StageInject, ReasonInternal, "inject failed; clipboard failed")
				return nil
			}

			// Wayland sentinel => clipboard counts as success (paste required)
			if errors.Is(err, ErrInjectUnavailableWayland) {
				respond(StatusOKClipboard, StageInject, ReasonInjectUnavailableWayland, "clipboard set (wayland; paste to insert)")
				return nil
			}

			// Non-wayland failure: clipboard is now the fallback path => also a success-with-paste
			respond(StatusOKClipboard, StageInject, ReasonClipboardFallback, "clipboard set (inject failed; paste to insert)")
			return nil
		}

		respond(StatusInternal, StageInject, ReasonInternal, "inject failed")
		return nil
	}

	// Success: include deterministic reason for UI cues
	logReqf(reqID, "injection complete; method=%s", method)
	switch method {
	case InjectMethodDirect:
		respond(StatusOK, StageInject, ReasonOK, "ok")
	case InjectMethodTyping:
		respond(StatusOK, StageInject, ReasonTypingFallback, "auto-typing used")
	case InjectMethodClipboard:
		// macOS clipboard+Cmd+V succeeded (actual paste occurred)
		respond(StatusOK, StageInject, ReasonClipboardFallback, "clipboard paste used")
	default:
		// Defensive: should not happen, but don't crash client logic
		respond(StatusOK, StageInject, ReasonOK, "ok")
	}
	return nil
}

