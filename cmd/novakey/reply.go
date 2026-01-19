// cmd/novakey/reply.go
package main

import (
	"encoding/json"
	"net"
	"time"
)

const replyVersion = 1

type ReplyStage string

const (
	StageMsg     ReplyStage = "msg"
	StageInject  ReplyStage = "inject"
	StageApprove ReplyStage = "approve"
	StageArm     ReplyStage = "arm"
	StageDisarm  ReplyStage = "disarm"
)

type ReplyReason string

const (
	ReasonOK                      ReplyReason = "ok"
	ReasonClipboardFallback        ReplyReason = "clipboard_fallback"
	ReasonTypingFallback           ReplyReason = "typing_fallback"
	ReasonInjectUnavailableWayland ReplyReason = "inject_unavailable_wayland"

	ReasonNotArmed     ReplyReason = "not_armed"
	ReasonNeedsApprove ReplyReason = "needs_approve"
	ReasonNotPaired    ReplyReason = "not_paired"

	// NOTE: These are valid server reasons, but older iOS clients may not
	// include them in their decoding enums and may crash if they appear.
	ReasonBadRequest   ReplyReason = "bad_request"
	ReasonBadTimestamp ReplyReason = "bad_timestamp"
	ReasonReplay       ReplyReason = "replay"
	ReasonRateLimit    ReplyReason = "rate_limit"
	ReasonCryptoFail   ReplyReason = "crypto_fail"
	ReasonInternal     ReplyReason = "internal_error"
)

type ServerReply struct {
	V      int         `json:"v"`
	Status uint8       `json:"status"`
	Stage  ReplyStage  `json:"stage"`
	Reason ReplyReason `json:"reason"`
	Msg    string      `json:"msg"`
	TsUnix int64       `json:"ts_unix"`
	ReqID  uint64      `json:"req_id"`
}

// safeReasonForClient returns a reason that is less likely to crash strict iOS decoders.
// It preserves the true reason by prefixing the message when we have to downgrade.
func safeReasonForClient(st RespStatus, reason ReplyReason, msg string) (ReplyReason, string) {
	// These are the ones that most clients tend to support.
	switch reason {
	case ReasonOK, ReasonNotArmed, ReasonNeedsApprove, ReasonClipboardFallback, ReasonTypingFallback, ReasonInjectUnavailableWayland:
		return reason, msg
	}

	// Downgrade unknown reason to "ok" but keep status accurate and embed true reason in msg.
	if msg == "" {
		msg = "error"
	}
	return ReasonOK, "reason=" + string(reason) + "; " + msg
}

func writeReplyLine(conn net.Conn, r ServerReply) {
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	defer func() { _ = conn.SetWriteDeadline(time.Time{}) }()

	b, err := json.Marshal(r)
	if err != nil {
		b = []byte(`{"v":1,"status":127,"stage":"msg","reason":"ok","msg":"reason=internal_error; marshal failed","ts_unix":0,"req_id":0}` + "\n")
	} else {
		b = append(b, '\n')
	}

	for len(b) > 0 {
		n, err := conn.Write(b)
		if err != nil {
			return
		}
		if n == 0 {
			return
		}
		b = b[n:]
	}
}

func makeReply(reqID uint64, st RespStatus, stage ReplyStage, reason ReplyReason, msg string) ServerReply {
	safeReason, safeMsg := safeReasonForClient(st, reason, msg)

	return ServerReply{
		V:      replyVersion,
		Status: uint8(st),
		Stage:  stage,
		Reason: safeReason,
		Msg:    safeMsg,
		TsUnix: time.Now().Unix(),
		ReqID:  reqID,
	}
}

