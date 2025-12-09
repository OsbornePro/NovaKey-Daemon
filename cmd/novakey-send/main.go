package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/kyber/kyber768"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	ProtocolVersion = 1

	// Must match server-side device_auth.go
	deviceMACInfo = "NovaKey-device-mac-v1"
)

type PairingQRPayload struct {
	Version      int    `json:"v"`
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
	ServerPubKey string `json:"server_pub"`
	KeyID        string `json:"key_id"`
	IssuedAt     int64  `json:"iat"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
}

func main() {
	var (
		qrPath   = flag.String("qr", "", "Path to pairing QR JSON")
		password = flag.String("password", "", "Password to send (optional; will prompt if empty)")
	)
	flag.Parse()

	if *qrPath == "" {
		fmt.Println("Usage: novakey-send --qr pairing.json [--password XXXX]")
		os.Exit(1)
	}

	if *password == "" {
		fmt.Print("Password: ")
		reader := bufio.NewReader(os.Stdin)
		pw, _ := reader.ReadString('\n')
		*password = strings.TrimSpace(pw)
	}

	qr, err := loadQR(*qrPath)
	if err != nil {
		die("Invalid QR payload:", err)
	}

	secret, err := base64.StdEncoding.DecodeString(qr.DeviceSecret)
	if err != nil {
		die("Invalid base64 device secret:", err)
	}

	pubKey, err := loadServerPublicKeyFromQR(qr)
	if err != nil {
		die("Failed to load server public key from pairing JSON:", err)
	}

	// Build plaintext: header || [idLen || deviceID || password] || HMAC
	plaintext, err := buildPlaintextV1(qr.DeviceID, []byte(*password), secret)
	if err != nil {
		die("Failed to build plaintext:", err)
	}

	scheme := kyber768.Scheme()
	ct, ss, err := scheme.Encapsulate(pubKey)
	if err != nil {
		die("Kyber encapsulation failed:", err)
	}

	// Encrypt payload with XChaCha20-Poly1305
	payload, err := encryptPayload(ss, plaintext)
	if err != nil {
		die("AEAD encrypt failed:", err)
	}

	// Send [ct || payload]
	addr := fmt.Sprintf("%s:%d", qr.Host, qr.Port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		die("Failed to connect:", err)
	}
	defer conn.Close()

	if _, err := conn.Write(ct); err != nil {
		die("Failed to write ciphertext:", err)
	}
	if _, err := conn.Write(payload); err != nil {
		die("Failed to write payload:", err)
	}

	fmt.Println("Payload sent successfully (v1 protocol)")
}

// buildPlaintextV1 — aligned with server v1 MAC layout:
//
//	header = [version(1) || timestamp(8) || nonce(16)]
//	body   = [idLen(1) || deviceID || password]
//	mac    = HMAC-SHA256(deviceMACInfo || header || deviceID || password, key=deviceSecret)
//
//	plaintext = header || body || mac
func buildPlaintextV1(deviceID string, password []byte, secret []byte) ([]byte, error) {
	if len(deviceID) == 0 || len(deviceID) > 255 {
		return nil, fmt.Errorf("device ID length must be 1..255")
	}

	ts := time.Now().Unix()
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Header: [version(1) || timestamp(8) || nonce(16)]
	header := make([]byte, 1+8+16)
	header[0] = ProtocolVersion
	binary.BigEndian.PutUint64(header[1:9], uint64(ts))
	copy(header[9:], nonce)

	// Body for transport: [idLen || deviceID || password]
	body := []byte{byte(len(deviceID))}
	body = append(body, []byte(deviceID)...)
	body = append(body, password...)

	// HMAC over deviceMACInfo || header || deviceID || password
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(deviceMACInfo))
	h.Write(header)
	h.Write([]byte(deviceID))
	h.Write(password)
	mac := h.Sum(nil)

	// Final plaintext: header || body || mac
	plaintext := append(header, body...)
	plaintext = append(plaintext, mac...)
	return plaintext, nil
}

func encryptPayload(key []byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

func loadQR(path string) (*PairingQRPayload, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p PairingQRPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func loadServerPublicKeyFromQR(qr *PairingQRPayload) (kem.PublicKey, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(qr.ServerPubKey)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 server_pub: %w", err)
	}
	scheme := kyber768.Scheme()
	pk, err := scheme.UnmarshalBinaryPublicKey(pubBytes)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

func die(msg string, err error) {
	fmt.Println("❌", msg)
	if err != nil {
		fmt.Println("   ", err)
	}
	os.Exit(1)
}
