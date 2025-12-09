package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/cloudflare/circl/kem/kyber/kyber768"
)

const (
	maxPayloadSize  = 16 * 1024
	maxPasswordSize = 4096
	replayWindow    = 5 * time.Minute

	rateWindow       = 1 * time.Minute
	maxRequestsPerIP = 30

	protocolVersion    = 1
	headerVersionLen   = 1
	headerTimestampLen = 8
	headerNonceLen     = 16
	headerLen          = headerVersionLen + headerTimestampLen + headerNonceLen

	CommandTypePassword = 0x01
)

var (
	rateMu      sync.Mutex
	clientRates = map[string]*clientRate{}

	replayMu   sync.Mutex
	seenNonces = map[[headerNonceLen]byte]int64{}
)

type clientRate struct {
	windowStart time.Time
	count       int
}

func handleConn(conn net.Conn, priv *kyber768.PrivateKey) {
	defer conn.Close()

	ip := clientIP(conn)
	if !allowClient(ip) {
		LogError("Rate limit exceeded for "+ip, nil)
		return
	}

	data, err := io.ReadAll(&io.LimitedReader{R: conn, N: maxPayloadSize})
	if err != nil {
		LogError("Read failed", err)
		return
	}
	if len(data) < kyberCtSize {
		LogError("Payload too short", nil)
		return
	}

	ct := data[:kyberCtSize]
	enc := data[kyberCtSize:]

	shared, err := Decapsulate(priv, ct)
	if err != nil {
		LogError("Decapsulation failed", err)
		return
	}
	defer zeroBytes(shared)

	plain, err := DecryptPayload(shared, enc)
	if err != nil {
		LogError("DecryptPayload failed", err)
		return
	}
	defer zeroBytes(plain)

	if len(plain) < headerLen+3+deviceMACLen {
		LogError("Decrypted payload too short", nil)
		return
	}

	// ---------------- Header ----------------
	header := plain[:headerLen]

	if header[0] != protocolVersion {
		LogError("Unsupported protocol version", nil)
		return
	}

	ts := int64(binary.BigEndian.Uint64(header[1:9]))
	var nonce [headerNonceLen]byte
	copy(nonce[:], header[9:headerLen])

	now := time.Now().Unix()
	if ts < now-int64(replayWindow.Seconds()) || ts > now+60 {
		LogError("Timestamp outside replay window", nil)
		return
	}
	if isReplay(nonce, ts) {
		LogError("Replay detected", nil)
		return
	}

	// ---------------- Body ----------------
	body := plain[headerLen:]

	cmd := body[0]
	if cmd != CommandTypePassword {
		LogError(fmt.Sprintf("Unsupported command type: %d", cmd), nil)
		return
	}

	idLen := int(body[1])
	if idLen <= 0 || len(body) < 2+idLen+deviceMACLen {
		LogError("Invalid device ID length", nil)
		return
	}

	deviceID := string(body[2 : 2+idLen])
	password := body[2+idLen : len(body)-deviceMACLen]
	mac := body[len(body)-deviceMACLen:]

	if len(password) == 0 {
		LogError("Empty password received", nil)
		return
	}
	if len(password) > maxPasswordSize {
		LogError("Password too long", nil)
		return
	}
	if !utf8.Valid(password) {
		LogError("Password not UTF-8", nil)
		return
	}

	LogInfo(fmt.Sprintf(
		"Payload received from device=%s len=%d",
		deviceID, len(password),
	))

	// ---------------- Device policy ----------------
	settingsMu.RLock()
	devCfg, known := settings.Devices.PairedDevices[deviceID]
	requireKnown := settings.Devices.RequireKnownDevice
	autoRegister := settings.Devices.AutoRegister
	settingsMu.RUnlock()

	if !known {
		if !autoRegister && len(settings.Devices.PairedDevices) > 0 {
			LogError("Typing denied: unknown device", nil)
			return
		}

		LogInfo("Auto-registering new device: " + deviceID)

		settingsMu.Lock()
		if settings.Devices.PairedDevices == nil {
			settings.Devices.PairedDevices = make(map[string]DeviceConfig)
		}
		settings.Devices.PairedDevices[deviceID] = DeviceConfig{}
		settings.Devices.AutoRegister = false
		_ = saveSettings("config.yaml")
		settingsMu.Unlock()
	}

	if requireKnown {
		if !verifyDeviceMAC(header, deviceID, password, mac, devCfg) {
			LogError("Invalid device MAC", nil)
			return
		}
		LogInfo("Device MAC verified")
	}

	// ✅✅✅ THIS MUST EXECUTE ✅✅✅
	LogInfo("Invoking HandlePayload")
	HandlePayload(password)
}

// ---------------- Typing ----------------

func HandlePayload(password []byte) {
	LogInfo(fmt.Sprintf("HandlePayload called (len=%d)", len(password)))

	if settings.Security.RequireArming && !isArmed() {
		LogError("Typing denied: service not armed", nil)
		return
	}

	LogInfo("Calling SecureType")
	SecureType(password)
	LogInfo("SecureType returned")

	zeroBytes(password)

	if settings.Security.RequireArming {
		disarm()
		LogInfo("Service disarmed after typing")
	}
}

// ---------------- Infrastructure ----------------

func clientIP(conn net.Conn) string {
	if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		if ip4 := addr.IP.To4(); ip4 != nil {
			return ip4.String()
		}
		return addr.IP.String()
	}
	return conn.RemoteAddr().String()
}

func allowClient(ip string) bool {
	now := time.Now()

	rateMu.Lock()
	defer rateMu.Unlock()

	cr, ok := clientRates[ip]
	if !ok || now.Sub(cr.windowStart) > rateWindow {
		clientRates[ip] = &clientRate{
			windowStart: now,
			count:       1,
		}
		return true
	}

	if cr.count >= maxRequestsPerIP {
		return false
	}

	cr.count++
	return true
}

func isReplay(nonce [headerNonceLen]byte, ts int64) bool {
	cutoff := time.Now().Add(-replayWindow).Unix()

	replayMu.Lock()
	defer replayMu.Unlock()

	for k, v := range seenNonces {
		if v < cutoff {
			delete(seenNonces, k)
		}
	}

	if old, ok := seenNonces[nonce]; ok && old == ts {
		return true
	}

	seenNonces[nonce] = ts
	return false
}
