// cmd/novakey/pairing_proto.go
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"filippo.io/mlkem768"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

type pairHello struct {
	Op    string `json:"op"`
	V     int    `json:"v"`
	Token string `json:"token"`
}

type pairServerKey struct {
	Op          string `json:"op"`
	V           int    `json:"v"`
	KID         string `json:"kid"`
	KyberPubB64 string `json:"kyber_pub_b64"`
	FP16Hex     string `json:"fp16_hex"`
	ExpiresUnix int64  `json:"expires_unix"`
}

type pairRegister struct {
	Op           string `json:"op"`
	V            int    `json:"v"`
	DeviceID     string `json:"device_id"`
	DeviceKeyHex string `json:"device_key_hex"`
}

// --- per-IP hello limiter (in-memory, per-uptime) ---
var (
	pairHelloMu sync.Mutex
	pairHelloRL = map[string]rateWindow{} // reuse rateWindow from crypto.go
)

func allowPairHelloFromIP(ip string) bool {
	limit := cfg.PairHelloMaxPerMin
	if limit <= 0 {
		limit = 30
	}

	now := time.Now().Unix()
	pairHelloMu.Lock()
	defer pairHelloMu.Unlock()

	rw := pairHelloRL[ip]
	if rw.windowStart == 0 || now-rw.windowStart >= 60 {
		rw.windowStart = now
		rw.count = 0
	}
	rw.count++
	pairHelloRL[ip] = rw
	return rw.count <= limit
}

func remoteIP(conn net.Conn) string {
	ra := conn.RemoteAddr().String()
	ip, _, err := net.SplitHostPort(ra)
	if err == nil && ip != "" {
		return ip
	}
	return ra
}

func handlePairConn(conn net.Conn) error {
	if serverDecapKey == nil || len(serverEncapKey) == 0 {
		return fmt.Errorf("server keys not initialized")
	}

	// Hard timeout for pairing flow; clear before return.
	_ = conn.SetDeadline(time.Now().Add(25 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	ip := remoteIP(conn)
	if !allowPairHelloFromIP(ip) {
		return fmt.Errorf("pair hello rate limited for %s", ip)
	}

	br := bufio.NewReaderSize(conn, 8192)

	// Read hello JSON line
	line, err := br.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("read hello: %w", err)
	}

	var hello pairHello
	if err := json.Unmarshal(trimNL(line), &hello); err != nil {
		return fmt.Errorf("bad hello json: %w", err)
	}
	if hello.Op != "hello" || hello.V != 1 {
		return fmt.Errorf("unexpected hello op/v")
	}
	if hello.Token == "" {
		return fmt.Errorf("missing token")
	}

	// Validate + consume token (one-time)
	tokenBytes, err := consumePairToken(hello.Token)
	if err != nil {
		return err
	}

	// Send server key info (plaintext)
	fp16 := fp16Hex(serverEncapKey)
	resp := pairServerKey{
		Op:          "server_key",
		V:           1,
		KID:         "1",
		KyberPubB64: base64.StdEncoding.EncodeToString(serverEncapKey),
		FP16Hex:     fp16,
		ExpiresUnix: time.Now().Add(2 * time.Minute).Unix(),
	}
	b, _ := json.Marshal(resp)
	b = append(b, '\n')
	if _, err := conn.Write(b); err != nil {
		return fmt.Errorf("write server_key: %w", err)
	}

	// Now read binary encapsulated register frame.
	ct, nonce, ciphertext, err := readPairBinaryFrame(br)
	if err != nil {
		return err
	}
	if len(ct) != mlkem768.CiphertextSize {
		return fmt.Errorf("bad ct len: %d", len(ct))
	}

	sharedKem, err := mlkem768.Decapsulate(serverDecapKey, ct)
	if err != nil {
		return fmt.Errorf("decaps failed: %w", err)
	}

	aeadKey, err := derivePairAEADKey(sharedKem, tokenBytes)
	if err != nil {
		return err
	}
	aead, err := chacha20poly1305.NewX(aeadKey)
	if err != nil {
		return fmt.Errorf("NewX: %w", err)
	}

	aad := makePairAAD(ct, nonce)

	plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return fmt.Errorf("pair decrypt failed: %w", err)
	}

	var reg pairRegister
	if err := json.Unmarshal(plaintext, &reg); err != nil {
		return fmt.Errorf("bad register json: %w", err)
	}
	if reg.Op != "register" || reg.V != 1 {
		return fmt.Errorf("unexpected register op/v")
	}

	// If client didn't choose, server assigns.
	if reg.DeviceID == "" {
		reg.DeviceID = "ios-" + randHex(8)
	}

	// If re-pair and rotation enabled: force a fresh PSK.
	// (We detect "existing device" by checking the current in-memory map.)
	if cfg.RotateDevicePSKOnRepair {
		devicesMu.RLock()
		_, exists := devices[reg.DeviceID]
		devicesMu.RUnlock()
		if exists {
			reg.DeviceKeyHex = "" // force regeneration below
		}
	}

	if reg.DeviceKeyHex == "" {
		reg.DeviceKeyHex = randHex(32) // 32 bytes
	}

	if err := writeDevicesFile(cfg.DevicesFile, reg.DeviceID, reg.DeviceKeyHex); err != nil {
		return fmt.Errorf("write devices: %w", err)
	}
	if err := reloadDevicesFromDisk(); err != nil {
		return fmt.Errorf("reload devices: %w", err)
	}

	ack := map[string]any{
		"op":        "ok",
		"v":         1,
		"device_id": reg.DeviceID,
	}
	ackB, _ := json.Marshal(ack)

	ackNonce := make([]byte, aead.NonceSize())
	_, _ = rand.Read(ackNonce)
	ackCT := aead.Seal(nil, ackNonce, ackB, makePairAAD(ct, ackNonce))

	if err := writePairAck(conn, ackNonce, ackCT); err != nil {
		return fmt.Errorf("write ack: %w", err)
	}

	log.Printf("[pair] paired device_id=%s (saved + reloaded)", reg.DeviceID)
	return nil
}

func fp16Hex(pub []byte) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:16])
}

func derivePairAEADKey(sharedKem []byte, token []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, sharedKem, token, []byte("NovaKey v4 Pair AEAD"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, fmt.Errorf("hkdf: %w", err)
	}
	return key, nil
}

func makePairAAD(ct []byte, nonce []byte) []byte {
	out := make([]byte, 0, 4+len(ct)+len(nonce))
	out = append(out, 'P', 'A', 'I', 'R')
	out = append(out, ct...)
	out = append(out, nonce...)
	return out
}

func readPairBinaryFrame(r *bufio.Reader) (ct []byte, nonce []byte, ciphertext []byte, err error) {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, nil, nil, fmt.Errorf("read ctLen: %w", err)
	}
	ctLen := int(binary.BigEndian.Uint16(hdr))
	if ctLen <= 0 || ctLen > 4096 {
		return nil, nil, nil, fmt.Errorf("invalid ctLen=%d", ctLen)
	}

	ct = make([]byte, ctLen)
	if _, err := io.ReadFull(r, ct); err != nil {
		return nil, nil, nil, fmt.Errorf("read ct: %w", err)
	}

	nonce = make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(r, nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("read nonce: %w", err)
	}

	ciphertext, err = io.ReadAll(io.LimitReader(r, 64*1024))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read ciphertext: %w", err)
	}
	if len(ciphertext) < 16 {
		return nil, nil, nil, fmt.Errorf("ciphertext too short")
	}
	return ct, nonce, ciphertext, nil
}

func writePairAck(w io.Writer, nonce []byte, ct []byte) error {
	if len(nonce) != chacha20poly1305.NonceSizeX {
		return fmt.Errorf("bad nonce size")
	}
	if _, err := w.Write(nonce); err != nil {
		return err
	}
	_, err := w.Write(ct)
	return err
}

func trimNL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}
