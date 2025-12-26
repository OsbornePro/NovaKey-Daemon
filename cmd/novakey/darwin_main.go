// cmd/novakey/darwin_main.go
//go:build darwin

package main

import (
	"encoding/binary"
	"fmt"
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

func maybeStartPairingQR() {
	if isPaired() {
		return
	}

	host, port := splitHostPortOrDie(cfg.ListenAddr)
	advertiseHost := chooseAdvertiseHost(host)

	tokenB64, tokenID, exp := startOrRefreshPairToken(10 * time.Minute)
	fp := fp16Hex(serverEncapKey)

	qr := fmt.Sprintf("novakey://pair?v=4&host=%s&port=%d&token=%s&fp=%s&exp=%d",
		advertiseHost, port, tokenB64, fp, exp.Unix())

	pngPath, err := writeAndOpenPairQR(".", qr)
	if err != nil {
		log.Printf("[pair] token id=%s expires=%s; QR at %s (viewer open failed: %v)",
			tokenID, exp.Format(time.RFC3339), pngPath, err)
	} else {
		log.Printf("[pair] token id=%s expires=%s; QR opened at %s",
			tokenID, exp.Format(time.RFC3339), pngPath)
	}
}

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("loadConfig failed: %v", err)
	}
	initLoggingFromConfig()

	if err := initCrypto(); err != nil {
		log.Fatalf("initCrypto failed: %v", err)
	}

	maybeStartPairingQR()
	startArmAPI()

	if err := startUnifiedListener(); err != nil {
		log.Fatalf("startUnifiedListener failed: %v", err)
	}

	log.Printf("NovaKey (macOS) started (listener=%s)", cfg.ListenAddr)
	select {}
}

func handleMsgConn(conn net.Conn) error {
	reqID := nextReqID()
	handleConnDarwin(reqID, conn, cfg.MaxPayloadLen)
	return nil
}

func handleConnDarwin(reqID uint64, conn net.Conn, maxLen int) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	logReqf(reqID, "connection opened from %s", remote)

	respond := func(st RespStatus, msg string) {
		logReqf(reqID, "responding status=%d msg=%q", st, msg)
		writeResp(conn, st, msg)
	}

	var length uint16
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		if err != io.EOF {
			logReqf(reqID, "read length failed: %v", err)
			respond(StatusBadRequest, "read length failed")
		} else {
			logReqf(reqID, "client closed connection before sending length")
			respond(StatusBadRequest, "client closed before length")
		}
		return
	}
	logReqf(reqID, "declared payload length=%d", length)

	if length == 0 || int(length) > maxLen {
		logReqf(reqID, "invalid length (%d), max=%d", length, maxLen)
		respond(StatusBadRequest, "invalid length")
		return
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		logReqf(reqID, "read payload failed: %v", err)
		respond(StatusBadRequest, "read payload failed")
		return
	}

	deviceID, msgType, payload, err := decryptMessageFrame(buf)
	if err != nil {
		logReqf(reqID, "decryptMessageFrame failed: %v", err)
		respond(StatusCryptoFail, "decrypt/auth failed")
		return
	}

	if msgType == MsgTypeApprove {
		if !cfg.TwoManEnabled {
			logReqf(reqID, "approve message received but two_man_enabled=false; ignoring")
			respond(StatusBadRequest, "two-man disabled; approve ignored")
			return
		}
		until := approvalGate.Approve(deviceID, approveWindow())
		logReqf(reqID, "two-man approve received from device=%q; approved until %s",
			deviceID, until.Format(time.RFC3339Nano))
		respond(StatusOK, "approved")
		return
	}

	if msgType != MsgTypeInject {
		logReqf(reqID, "unknown msgType=%d from device=%q; dropping", msgType, deviceID)
		respond(StatusBadRequest, "unknown msgType")
		return
	}

	password := string(payload)
	logReqf(reqID, "decrypted password payload from device=%q: %s", deviceID, safePreview(password))

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
			return
		}

		respond(StatusBadRequest, "unsafe text")
		return
	}

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
			return
		}

		respond(StatusBadRequest, "target policy blocked")
		return
	}

	injectMu.Lock()
	defer injectMu.Unlock()

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
				return
			}

			respond(StatusNeedsApprove, "needs approve")
			return
		}
		logReqf(reqID, "two-man approval OK; proceeding")
	}

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
				return
			}

			respond(StatusNotArmed, "not armed")
			return
		}
		logReqf(reqID, "armed gate open; proceeding with injection")
	}

	if err := InjectPasswordToFocusedControl(password); err != nil {
		logReqf(reqID, "InjectPasswordToFocusedControl error: %v", err)

		if allowClipboardWhenBlocked() {
			if err2 := trySetClipboard(password); err2 != nil {
				logReqf(reqID, "clipboard set failed: %v", err2)
				respond(StatusInternal, "inject failed; clipboard failed")
			} else {
				logReqf(reqID, "injection failed; clipboard set")
				respond(StatusInternal, "inject failed; clipboard set")
			}
			return
		}

		respond(StatusInternal, "inject failed")
		return
	}

	logReqf(reqID, "injection complete")
	respond(StatusOK, "ok")
}
