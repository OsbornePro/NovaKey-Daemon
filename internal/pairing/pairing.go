package pairing

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"
)

const (
	PairingVersion = 1
	DefaultKeyID   = "kyber768-v1"
)

type PairingPayload struct {
	Version      int    `json:"v"`
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`

	ServerPubKey string `json:"server_pub"`
	KeyID        string `json:"key_id"`
	IssuedAt     int64  `json:"iat"`

	Host string `json:"host"`
	Port int    `json:"port"`
}

// GenerateDeviceID returns a random URL-safe device ID
func GenerateDeviceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// GenerateDeviceSecret returns a random 32-byte secret
func GenerateDeviceSecret() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return b, err
}

// BuildPairingPayload creates a signed JSON pairing payload
func BuildPairingPayload(
	deviceID string,
	deviceSecret []byte,
	serverPub []byte,
	host string,
	port int,
) ([]byte, error) {

	p := PairingPayload{
		Version:      PairingVersion,
		DeviceID:     deviceID,
		DeviceSecret: base64.StdEncoding.EncodeToString(deviceSecret),

		ServerPubKey: base64.StdEncoding.EncodeToString(serverPub),
		KeyID:        DefaultKeyID,
		IssuedAt:     time.Now().Unix(),

		Host: host,
		Port: port,
	}

	return json.MarshalIndent(p, "", "  ")
}
