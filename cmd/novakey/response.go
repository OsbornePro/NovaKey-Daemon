package novakey

import (
	"encoding/binary"
	"net"
	"time"
)

type RespStatus byte

const (
	StatusOK            RespStatus = 0x00
	StatusNotArmed      RespStatus = 0x01
	StatusNeedsApprove  RespStatus = 0x02
	StatusNotPaired     RespStatus = 0x03
	StatusBadTimestamp  RespStatus = 0x04
	StatusReplay        RespStatus = 0x05
	StatusRateLimit     RespStatus = 0x06
	StatusDecryptFailed RespStatus = 0x07
	StatusInternal      RespStatus = 0x7F
)

func writeResponse(conn net.Conn, status RespStatus, msg string) {
	// Best-effort: old clients may close immediately; ignore write errors.
	_ = conn.SetWriteDeadline(time.Now().Add(750 * time.Millisecond))

	b := []byte(msg)
	if len(b) > 65535 {
		b = b[:65535]
	}

	var hdr [3]byte
	hdr[0] = byte(status)
	binary.BigEndian.PutUint16(hdr[1:], uint16(len(b)))

	_, _ = conn.Write(hdr[:])
	if len(b) > 0 {
		_, _ = conn.Write(b)
	}
}
