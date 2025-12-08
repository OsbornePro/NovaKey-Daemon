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
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/kyber/kyber768"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	// Must match the server-side constants.
	deviceMACInfo    = "NovaKey-device-mac-v1"
	transportContext = "NovaKey-transport-v1"
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

	// Derive the transport session key via HKDF-SHA256 with a fixed context,
	// matching the server's Decapsulate().
	sessionKey, err := deriveSessionKey(ss)
	if err != nil {
		die("Failed to derive session key:", err)
	}
	// ss is no longer needed after deriving the key
	for i := range ss {
		ss[i] = 0
	}

	// Encrypt payload with XChaCha20-Poly1305 using the derived session key.
	payload, err := encryptPayload(sessionKey, plaintext)
	if err != nil {
		die("AEAD encrypt failed:", err)
	}
	for i := range sessionKey {
		sessionKey[i] = 0
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

// buildPlaintext constructs:
//
//	header  = [8-byte big-endian timestamp || 16-byte random nonce]
//	body    = [1-byte deviceID length || deviceID || password]
//	mac     = HMAC-SHA256 over (deviceMACInfo || header || deviceID || password)
//	final   = header || body || mac
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
	if len(deviceID) > 255 {
		return nil, fmt.Errorf("device ID too long")
	}
	body := []byte{byte(len(deviceID))}
	body = append(body, []byte(deviceID)...)
	body = append(body, password...)

	// HMAC over (deviceMACInfo || header || deviceID || password)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(deviceMACInfo))
	mac.Write(header)
	mac.Write([]byte(deviceID))
	mac.Write(password)
	sum := mac.Sum(nil)

	// final plaintext = header || body || hmac
	plaintext := make([]byte, 0, len(header)+len(body)+len(sum))
	plaintext = append(plaintext, header...)
	plaintext = append(plaintext, body...)
	plaintext = append(plaintext, sum...)
	return plaintext, nil
}

// deriveSessionKey mirrors the server-side HKDF derivation in Decapsulate().
func deriveSessionKey(ss []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, ss, nil, []byte(transportContext))
	key := make([]byte, 32)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, err
	}
	return key, nil
}

// encryptPayload encrypts the plaintext with XChaCha20-Poly1305 using
// the provided session key and returns [nonce || ciphertext].
func encryptPayload(sessionKey []byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(sessionKey)
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

// loadServerPublicKeyFromQR decodes the server's Kyber public key
// from the pairing JSON.
func loadServerPublicKeyFromQR(qr *PairingQRPayload) (kem.PublicKey, error) {
	if qr.ServerPubKey == "" {
		return nil, fmt.Errorf("pairing payload missing server_pub")
	}

	pubBytes, err := base64.StdEncoding.DecodeString(qr.ServerPubKey)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 server_pub: %w", err)
	}

	scheme := kyber768.Scheme()
	return scheme.UnmarshalBinaryPublicKey(pubBytes)
}

func die(msg string, err error) {
	fmt.Println("❌", msg)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(1)
}
