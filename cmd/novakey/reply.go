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
	ReasonInjectUnavailableWayland ReplyReason = "inject_unavailable_wayland"

	ReasonNotArmed     ReplyReason = "not_armed"
	ReasonNeedsApprove ReplyReason = "needs_approve"
	ReasonNotPaired    ReplyReason = "not_paired"
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

// Always return ONE JSON line terminated by '\n'
func writeReplyLine(conn net.Conn, r ServerReply) {
	_ = conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	b, _ := json.Marshal(r)
	b = append(b, '\n')
	_, _ = conn.Write(b)
}

func makeReply(reqID uint64, st RespStatus, stage ReplyStage, reason ReplyReason, msg string) ServerReply {
	return ServerReply{
		V:      replyVersion,
		Status: uint8(st),
		Stage:  stage,
		Reason: reason,
		Msg:    msg,
		TsUnix: time.Now().Unix(),
		ReqID:  reqID,
	}
}
