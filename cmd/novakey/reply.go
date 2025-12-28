// cmd/novakey/reply.go
package main

import (
	"encoding/json"
	"net"
)

type ReplyReason string

const (
	ReasonOK                     ReplyReason = "ok"
	ReasonClipboardFallback       ReplyReason = "clipboard_fallback"
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
	Status  uint8              `json:"status"`
	Msg     string             `json:"msg,omitempty"`     // keep for backwards compatibility
	Stage   string             `json:"stage,omitempty"`   // "inject" | "approve" | "arm" | "disarm"
	Reason  ReplyReason        `json:"reason,omitempty"`  // machine readable
	Details map[string]any     `json:"details,omitempty"` // optional
}

func writeReplyLine(conn net.Conn, r ServerReply) {
	// Always return ONE JSON line terminated by '\n' to match your current client behavior.
	b, _ := json.Marshal(r)
	b = append(b, '\n')
	_, _ = conn.Write(b)
}
