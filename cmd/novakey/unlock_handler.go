package main

import (
	"unicode/utf8"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"github.com/cloudflare/circl/kem/kyber/kyber768"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	maxPayloadSize  = 16 * 1024       // 16 KB max total payload
	maxPasswordSize = 4096            // 4 KB max password
	replayWindow    = 5 * time.Minute // how long nonces are remembered

	rateWindow       = 1 * time.Minute // per-IP window
	maxRequestsPerIP = 30              // max connections per IP per window

	headerTimestampLen = 8  // int64 seconds
	headerNonceLen     = 16 // 16-byte nonce
	headerLen          = headerTimestampLen + headerNonceLen
)

var (
	rateMu      sync.Mutex
	clientRates = make(map[string]*clientRate)

	replayMu   sync.Mutex
	seenNonces = make(map[[headerNonceLen]byte]int64)
)

type clientRate struct {
	windowStart time.Time
	count       int
}

func HandlePayload(password []byte) {
	if !utf8.Valid(password) {
		LogError("Typing denied: password payload is not valid UTF-8", nil)
		zeroBytes(password)
		return
	}

	// Require explicit arming if enabled
	if settings.Security.RequireArming && !isArmed() {
		LogError("Typing denied: service is not armed", nil)
		return
	}

	// Enforce foreground application allowlist if enabled
	if settings.Security.EnforceAllowlist {
		allowed, exe, err := foregroundAppAllowed()
		if err != nil {
			LogError("Typing denied: failed to determine foreground application", err)
			return
		}
		if !allowed {
			LogError("Typing denied: foreground app not allowed ("+exe+")", nil)
			return
		}
	}

	LogInfo("Typing allowed; injecting keystrokes")

	SecureType(password)
	zeroBytes(password)

	if settings.Security.RequireArming {
		disarm()
	}
}

func handleConn(conn net.Conn, priv *kyber768.PrivateKey) {
	defer conn.Close()

	ip := clientIP(conn)
	if !allowClient(ip) {
		LogError("Rate limit exceeded for "+ip, nil)
		return
	}

	limitedReader := &io.LimitedReader{R: conn, N: maxPayloadSize}
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		LogError("Read failed", err)
		return
	}

	if len(data) < kyberCtSize {
		LogError("Payload too short", nil)
		return
	}

	ct := data[:kyberCtSize]
	encPayload := data[kyberCtSize:]

	// Must contain AEAD nonce + header + device+password+MAC.
	if len(encPayload) < chacha20poly1305.NonceSizeX+headerLen+1+deviceMACLen {
		LogError("Encrypted payload too short", nil)
		return
	}
	if len(encPayload) > maxPayloadSize {
		LogError("Encrypted payload too large", nil)
		return
	}

	sharedSecret, err := Decapsulate(priv, ct)
	if err != nil {
		LogError("Decapsulation failed", err)
		return
	}
	defer zeroBytes(sharedSecret)

	// We decrypt first, then validate the replay/nonce header inside.
	plain, err := DecryptPayload(sharedSecret, encPayload, nil) // temporary AAD = nil, will re-check below
	if err != nil {
		LogError("DecryptPayload failed", err)
		return
	}
	defer zeroBytes(plain)

	if len(plain) < headerLen+1+deviceMACLen {
		LogError("Decrypted payload too short for header+device+MAC", nil)
		return
	}

	// --- Replay protection header ---
	header := plain[:headerLen]

	ts := int64(binary.BigEndian.Uint64(plain[:headerTimestampLen]))
	var nonce [headerNonceLen]byte
	copy(nonce[:], plain[headerTimestampLen:headerLen])

	now := time.Now()
	nowUnix := now.Unix()

	if ts > nowUnix+60 || ts < nowUnix-int64(replayWindow.Seconds()) {
		LogError("Decrypted payload timestamp out of acceptable window", nil)
		return
	}

	if isReplay(nonce, ts) {
		LogError("Replay detected; dropping payload", nil)
		return
	}

	// At this point we know header is sane; reconstruct the AAD and
	// re-verify the AEAD with proper associated data.
	//
	// NOTE: To avoid double-decryption in a future protocol revision,
	// you could instead pass the header as AAD from the sender side.
	aad := make([]byte, 0, headerLen+len(transportContext))
	aad = append(aad, header...)
	aad = append(aad, []byte(transportContext)...)

	// Re-decrypt with AAD to ensure integrity is bound to header.
	plainWithAAD, err := DecryptPayload(sharedSecret, encPayload, aad)
	if err != nil {
		LogError("AEAD re-check with AAD failed", err)
		return
	}
	defer zeroBytes(plainWithAAD)

	if len(plainWithAAD) < headerLen+1+deviceMACLen {
		LogError("Decrypted(AAD) payload too short for header+device+MAC", nil)
		return
	}

	// --- Device ID + password + MAC ---
	payload := plainWithAAD[headerLen:]

	deviceID, password, mac, err := parseDevicePayload(payload)
	if err != nil {
		LogError("Invalid payload format (device ID / MAC)", err)
		return
	}

	if len(password) > maxPasswordSize {
		LogError("Decrypted password too large", nil)
		return
	}

	// Enforce known-device policy + HMAC verification if enabled.
	if settings.Devices.RequireKnownDevice {
		devCfg, ok := settings.Devices.PairedDevices[deviceID]
		if !ok {
			LogError("Typing denied: unknown device ("+deviceID+")", nil)
			return
		}

		if !verifyDeviceMAC(header, deviceID, password, mac, devCfg) {
			LogError("Typing denied: invalid device MAC ("+deviceID+")", nil)
			return
		}
	}

	HandlePayload(password)
}

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
	now := time.Now()
	cutoff := now.Add(-replayWindow).Unix()

	replayMu.Lock()
	defer replayMu.Unlock()

	// Garbage collect expired nonces
	for k, v := range seenNonces {
		if v < cutoff {
			delete(seenNonces, k)
		}
	}

	if oldTs, ok := seenNonces[nonce]; ok && oldTs == ts {
		return true
	}

	seenNonces[nonce] = ts
	return false
}
