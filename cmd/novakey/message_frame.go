// cmd/novakey/message_frame.go
package main

import (
	"encoding/binary"
	"fmt"
)

const (
	frameVersionV1 = 1
	MsgTypeInject  = 1
    MsgTypeApprove = 2
    MsgTypeArm     = 3
)

// Frame format (plaintext BEFORE encryption):
//
//	[0]   = version (uint8) = 1
//	[1]   = msgType (uint8) = 1 inject, 2 approve
//	[2:4] = deviceIDLen (uint16, big endian)
//	[4:8] = payloadLen  (uint32, big endian)
//	[..]  = deviceID bytes (UTF-8)
//	[..]  = payload bytes  (UTF-8)
//
// Notes:
// - payload for MsgTypeApprove can be empty.
// - payload for MsgTypeInject is the secret string.
func encodeMessageFrame(deviceID string, msgType uint8, payload []byte) ([]byte, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("deviceID required")
	}
	if msgType != MsgTypeInject && msgType != MsgTypeApprove {
		return nil, fmt.Errorf("invalid msgType=%d", msgType)
	}

	dev := []byte(deviceID)
	if len(dev) > 0xFFFF {
		return nil, fmt.Errorf("deviceID too long")
	}

	out := make([]byte, 0, 1+1+2+4+len(dev)+len(payload))
	out = append(out, byte(frameVersionV1))
	out = append(out, byte(msgType))

	tmp2 := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp2, uint16(len(dev)))
	out = append(out, tmp2...)

	tmp4 := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp4, uint32(len(payload)))
	out = append(out, tmp4...)

	out = append(out, dev...)
	out = append(out, payload...)
	return out, nil
}

func decodeMessageFrame(b []byte) (deviceID string, msgType uint8, payload []byte, err error) {
	if len(b) < 1+1+2+4 {
		return "", 0, nil, fmt.Errorf("frame too short")
	}
	ver := b[0]
	if ver != frameVersionV1 {
		return "", 0, nil, fmt.Errorf("unsupported frame version=%d", ver)
	}
	msgType = b[1]
	if msgType != MsgTypeInject && msgType != MsgTypeApprove {
		return "", 0, nil, fmt.Errorf("invalid msgType=%d", msgType)
	}

	devLen := int(binary.BigEndian.Uint16(b[2:4]))
	plLen := int(binary.BigEndian.Uint32(b[4:8]))

	if devLen < 1 {
		return "", 0, nil, fmt.Errorf("deviceIDLen invalid")
	}
	if devLen > 0xFFFF {
		return "", 0, nil, fmt.Errorf("deviceIDLen too large")
	}
	if plLen < 0 {
		return "", 0, nil, fmt.Errorf("payloadLen invalid")
	}

	want := 1 + 1 + 2 + 4 + devLen + plLen
	if len(b) != want {
		return "", 0, nil, fmt.Errorf("length mismatch: have=%d want=%d", len(b), want)
	}

	devStart := 8
	devEnd := devStart + devLen
	deviceID = string(b[devStart:devEnd])

	plStart := devEnd
	plEnd := plStart + plLen
	payload = b[plStart:plEnd]
	return deviceID, msgType, payload, nil
}
