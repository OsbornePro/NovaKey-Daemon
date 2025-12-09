package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

const (
	deviceMACLen  = 32 // HMAC-SHA256 size
	deviceMACInfo = "NovaKey-device-mac-v1"
)

// parseDevicePayload splits the payload into (deviceID, password, mac).
// Layout (after header):
//
//	[idLen (1 byte)] [deviceID (idLen)] [password] [mac (32 bytes)]
func parseDevicePayload(data []byte) (string, []byte, []byte, error) {
	if len(data) < 1+deviceMACLen {
		return "", nil, nil, errors.New("payload too short for device+MAC")
	}

	// Last 32 bytes are MAC, everything before is body.
	macStart := len(data) - deviceMACLen
	body := data[:macStart]
	mac := data[macStart:]

	if len(body) < 1 {
		return "", nil, nil, errors.New("no room for device ID length")
	}

	idLen := int(body[0])
	if idLen <= 0 {
		return "", nil, nil, errors.New("invalid device ID length")
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

// verifyDeviceMAC checks:
//
//	HMAC-SHA256(deviceMACInfo || header || deviceID || password)
//
// using the per-device secret from config.
//
// If cfg.Secret is empty, MAC enforcement is skipped (for initial bring-up).
func verifyDeviceMAC(header []byte, deviceID string, password []byte, mac []byte, cfg DeviceConfig) bool {
	if cfg.Secret == "" {
		// No per-device secret configured; skip MAC enforcement for now.
		return true
	}

	h := hmac.New(sha256.New, []byte(cfg.Secret))

	// Scope this MAC to the NovaKey device-auth protocol/version.
	h.Write([]byte(deviceMACInfo))
	h.Write(header)
	h.Write([]byte(deviceID))
	h.Write(password)

	expected := h.Sum(nil)
	return hmac.Equal(expected, mac)
}
