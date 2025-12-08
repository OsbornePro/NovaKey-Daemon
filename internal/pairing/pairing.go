package pairing

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type PairingPayload struct {
	Version      int    `json:"v"`
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
}

// GenerateDeviceID returns a random URL-safe device ID
func GenerateDeviceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// GenerateDeviceSecret returns a random 32-byte secret
func GenerateDeviceSecret() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return b, err
}

// BuildPairingPayload creates the QR JSON payload
func BuildPairingPayload(
	deviceID string,
	secret []byte,
	host string,
	port int,
) ([]byte, error) {
	p := PairingPayload{
		Version:      1,
		DeviceID:     deviceID,
		DeviceSecret: base64.StdEncoding.EncodeToString(secret),
		Host:         host,
		Port:         port,
	}
	return json.MarshalIndent(p, "", "  ")
}

// PrintPairingInfo prints user-friendly pairing info
func PrintPairingInfo(deviceID string, secret []byte, payload []byte) {
	fmt.Println("Device paired successfully")
	fmt.Println("Device ID :", deviceID)
	fmt.Println()
	fmt.Println("Save this JSON (QR payload):")
	fmt.Println(string(payload))
}
