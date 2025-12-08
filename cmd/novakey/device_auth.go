package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

const deviceMACLen = 32 // HMAC-SHA256 size

// parseDevicePayload splits the payload into (deviceID, password, mac).
// Layout (after header):
//   [1 byte deviceID length][deviceID][password][32-byte MAC]
func parseDevicePayload(data []byte) (string, []byte, []byte, error) {
	if len(data) < 1+deviceMACLen {
		return "", nil, nil, errors.New("payload too short for device+MAC")
	}

	macStart := len(data) - deviceMACLen
	body := data[:macStart]
	mac := data[macStart:]

	if len(body) < 1 {
		return "", nil, nil, errors.New("no room for device ID length")
	}

	idLen := int(body[0])
	if idLen == 0 {
		return "", nil, nil, errors.New("empty device ID")
	}
	if len(body) < 1+idLen {
		return "", nil, nil, errors.New("truncated device ID")
	}

	deviceID := string(body[1 : 1+idLen])
	password := body[1+idLen:]

	if len(password) == 0 {
		return "", nil, nil, errors.New("empty password")
	}

	return deviceID, password, mac, nil
}

// verifyDeviceMAC checks HMAC-SHA256 over (header || deviceID || password)
// using the per-device secret from config.
func verifyDeviceMAC(header []byte, deviceID string, password []byte, mac []byte, cfg DeviceConfig) bool {
	if cfg.Secret == "" {
		// Misconfigured device; fail closed.
		return false
	}

	h := hmac.New(sha256.New, []byte(cfg.Secret))
	h.Write(header)
	h.Write([]byte(deviceID))
	h.Write(password)

	expected := h.Sum(nil)
	return hmac.Equal(expected, mac)
}
