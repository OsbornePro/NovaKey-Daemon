package main

import "errors"

// extractDeviceID splits the decrypted payload into
// (deviceID, passwordBytes)
func extractDeviceID(data []byte) (string, []byte, error) {
	if len(data) < 1 {
		return "", nil, errors.New("payload too short")
	}

	idLen := int(data[0])
	if idLen == 0 {
		return "", nil, errors.New("empty device ID")
	}

	if len(data) < 1+idLen {
		return "", nil, errors.New("invalid device ID length")
	}

	deviceID := string(data[1 : 1+idLen])
	password := data[1+idLen:]

	if len(password) == 0 {
		return "", nil, errors.New("missing password data")
	}

	return deviceID, password, nil
}
