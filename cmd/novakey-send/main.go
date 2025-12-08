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

type PairingQRPayload struct {
	Version      int    `json:"v"`
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
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

	pubKey, err := loadServerPublicKey()
	if err != nil {
		die("Failed to load server public key:", err)
	}

	// Build plaintext (replay header + deviceID + password + HMAC)
	plaintext, err := buildPlaintext(qr.DeviceID, []byte(*password), secret)
	if err != nil {
		die("Failed to build plaintext:", err)
	}

	// Kyber encapsulation via the KEM Scheme interface
	scheme := kyber768.Scheme()
	ct, ss, err := scheme.Encapsulate(pubKey)
	if err != nil {
		die("Kyber encapsulation failed:", err)
	}

	// Encrypt payload with XChaCha20-Poly1305 using shared secret
	payload, err := encryptPayload(ss, plaintext)
	if err != nil {
		die("AEAD encrypt failed:", err)
	}

	// Send [ct || payload] to service
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

	fmt.Println("✅ Payload sent successfully")
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

func buildPlaintext(deviceID string, password []byte, secret []byte) ([]byte, error) {
	ts := time.Now().Unix()

	// 16-byte replay nonce
	replay := make([]byte, 16)
	if _, err := rand.Read(replay); err != nil {
		return nil, err
	}

	// 8-byte timestamp (big-endian) + 16-byte nonce
	header := make([]byte, 8+16)
	binary.BigEndian.PutUint64(header[:8], uint64(ts))
	copy(header[8:], replay)

	// body = [deviceIDLen (1 byte) || deviceID || password]
	body := []byte{byte(len(deviceID))}
	body = append(body, []byte(deviceID)...)
	body = append(body, password...)

	// HMAC over (header || deviceID || password)
	mac := hmac.New(sha256.New, secret)
	mac.Write(header)
	mac.Write([]byte(deviceID))
	mac.Write(password)
	sum := mac.Sum(nil)

	// final plaintext = header || body || hmac
	return append(append(header, body...), sum...), nil
}

func encryptPayload(ss []byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(ss)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	cipher := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, cipher...), nil
}

func loadServerPublicKey() (kem.PublicKey, error) {
	// For now, read raw public key bytes from a file:
	// future: expose via pairing QR or control API
	b, err := os.ReadFile("server.pub")
	if err != nil {
		return nil, err
	}
	scheme := kyber768.Scheme()
	return scheme.UnmarshalBinaryPublicKey(b)
}

func die(msg string, err error) {
	fmt.Println("❌", msg)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(1)
}
