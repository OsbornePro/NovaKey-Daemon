// cmd/novakey/pair_bootstrap_tcp.go
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type tcpBootstrapReq struct {
	Op    string `json:"op"`
	V     int    `json:"v"`
	Token string `json:"token"`
}

type tcpBootstrapResp struct {
	V               int    `json:"v"`
	DeviceID        string `json:"device_id"`
	DeviceKeyHex    string `json:"device_key_hex"`
	ServerAddr      string `json:"server_addr"`
	ServerKyberPub  string `json:"server_kyber768_pub"`
	ExpiresAtUnix   int64  `json:"expires_at_unix"`
}

type tcpCompleteResp struct {
	Ok bool `json:"ok"`
}

func handlePairConnWithRoute(route string, conn net.Conn) error {
	switch route {
	case "/pair":
		return handlePairConn(conn) // existing Kyber/XChaCha flow
	case "/pair/bootstrap":
		return handlePairBootstrapTCP(conn)
	case "/pair/complete":
		return handlePairCompleteTCP(conn)
	default:
		return handlePairConn(conn)
	}
}

func handlePairBootstrapTCP(conn net.Conn) error {
	if len(serverEncapKey) == 0 {
		return fmt.Errorf("server keys not initialized")
	}

	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	br := bufio.NewReaderSize(conn, 8192)

	line, err := br.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("read bootstrap json: %w", err)
	}

	var req tcpBootstrapReq
	if err := json.Unmarshal(trimNL(line), &req); err != nil {
		return fmt.Errorf("bad bootstrap json: %w", err)
	}
	if req.Op != "bootstrap" || req.V != 1 {
		return fmt.Errorf("unexpected bootstrap op/v")
	}
	if req.Token == "" {
		return fmt.Errorf("missing token")
	}

	// ✅ Bootstrap consumes the one-time token (this is the *bootstrap* flow).
	// If you intended to also run the Kyber /pair hello flow with the same token,
	// do NOT use /pair/bootstrap; use /pair directly.
	_, err = consumePairToken(req.Token)
	if err != nil {
		return err
	}

	deviceID := "ios-" + randHex(8)
	deviceKeyHex := randHex(32) // 32 bytes -> 64 hex chars

	if err := writeDevicesFile(cfg.DevicesFile, deviceID, deviceKeyHex); err != nil {
		return fmt.Errorf("write devices: %w", err)
	}
	if err := reloadDevicesFromDisk(); err != nil {
		return fmt.Errorf("reload devices: %w", err)
	}

	resp := tcpBootstrapResp{
		V:              1,
		DeviceID:       deviceID,
		DeviceKeyHex:   deviceKeyHex,
		ServerAddr:     cfg.ListenAddr, // host:port; your iOS will parse this
		ServerKyberPub: base64.StdEncoding.EncodeToString(serverEncapKey),
		ExpiresAtUnix:  time.Now().Add(2 * time.Minute).Unix(),
	}

	b, _ := json.Marshal(resp)
	b = append(b, '\n')
	_, err = conn.Write(b)
	return err
}

func handlePairCompleteTCP(conn net.Conn) error {
	_ = conn.SetDeadline(time.Now().Add(6 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	br := bufio.NewReaderSize(conn, 4096)

	line, err := br.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("read complete json: %w", err)
	}

	var req tcpBootstrapReq
	if err := json.Unmarshal(trimNL(line), &req); err != nil {
		return fmt.Errorf("bad complete json: %w", err)
	}
	if req.Op != "complete" || req.V != 1 {
		return fmt.Errorf("unexpected complete op/v")
	}
	// token is optional here; iOS sends it. We don't need it server-side now.
	ack := tcpCompleteResp{Ok: true}
	b, _ := json.Marshal(ack)
	b = append(b, '\n')
	_, err = conn.Write(b)
	return err
}

