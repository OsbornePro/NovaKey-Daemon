package main

import (
	"encoding/binary"
	"net"
	"time"
)

const respVersion byte = 3

type RespStatus byte

const (
	StatusOK          RespStatus = 0x00
	StatusNotArmed    RespStatus = 0x01
	StatusNeedsApprove RespStatus = 0x02
	StatusNotPaired   RespStatus = 0x03
	StatusBadRequest  RespStatus = 0x04
	StatusBadTimestamp RespStatus = 0x05
	StatusReplay      RespStatus = 0x06
	StatusRateLimit   RespStatus = 0x07
	StatusCryptoFail  RespStatus = 0x08
	StatusInternal    RespStatus = 0x7F
)

func writeResp(conn net.Conn, st RespStatus, msg string) {
	_ = conn.SetWriteDeadline(time.Now().Add(1 * time.Second))

	b := []byte(msg)
	if len(b) > 65535 {
		b = b[:65535]
	}

	var hdr [4]byte
	hdr[0] = respVersion
	hdr[1] = byte(st)
	binary.BigEndian.PutUint16(hdr[2:], uint16(len(b)))

	_, _ = conn.Write(hdr[:])
	if len(b) > 0 {
		_, _ = conn.Write(b)
	}
}
