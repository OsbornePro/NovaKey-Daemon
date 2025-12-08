package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

type PairingQRPayload struct {
	Version      int    `json:"v"`
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
}

// generateDeviceSecret returns 32 random bytes, base64-encoded.
func generateDeviceSecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(secret), nil
}

// generateRandomDeviceID can be used if the user doesn't supply one.
func generateRandomDeviceID() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	length := 12

	out := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		out[i] = charset[n.Int64()]
	}
	return "device-" + string(out), nil
}

// buildPairingPayload returns a JSON string to embed into a QR code.
func buildPairingPayload(deviceID, deviceSecret, host string, port int) (string, error) {
	p := PairingQRPayload{
		Version:      1,
		DeviceID:     deviceID,
		DeviceSecret: deviceSecret,
		Host:         host,
		Port:         port,
	}

	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// printPairingInfo prints human-readable instructions and the QR payload.
func printPairingInfo(deviceID, deviceSecret, payload string) {
	fmt.Println("NovaKey Pairing Information")
	fmt.Println("---------------------------")
	fmt.Println("Device ID:     ", deviceID)
	fmt.Println("Device Secret: ", deviceSecret)
	fmt.Println()
	fmt.Println("QR Payload (embed this in a QR code for the phone app):")
	fmt.Println(payload)
	fmt.Println()
	fmt.Println("On the phone, scan this QR and store device_id + device_secret.")
}
