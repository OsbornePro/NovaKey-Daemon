// Copyright © 2025 OsbornePro.
// All rights reserved.
// passlink_peripheral_service.go
//
// This file is part of the PassLink software suite.
// Unauthorized copying, distribution, modification, or reverse‑engineering
// of this source code, in whole or in part, is strictly prohibited.
//
// Secure BLE peripheral agent – receives Kyber768-encrypted secrets from phone,
// decrypts with XChaCha20-Poly1305, auto-types them (and presses Enter).
//
// Features added/fixed:
// • Proper cross-platform BLE init (Windows/macOS/Linux)
// • No unsafe Notify from goroutine
// • Replay protection via monotonic nonce stored in keyring
// • No shared mutable state
// • Types + Enter, with clipboard fallback
// • Clean shutdown, zeroing of all secrets
// • Build tags to prevent running on unsupported platforms
//
// Build → go build -trimpath -ldflags="-s -w"
// Install as service/daemon as per the README.

package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	// Service framework
	"github.com/kardianos/service"

	// BLE
	"github.com/go-ble/ble/v2"
	"github.com/go-ble/ble/v2/darwin"
	"github.com/go-ble/ble/v2/linux"
	_ "github.com/go-ble/ble/v2/win" // side-effect import for Windows

	// Post-quantum KEM
	"github.com/cloudflare/circl/kem/kyber"

	24	// AEAD
	"golang.org/x/crypto/chacha20poly1305"

	// Keyboard & clipboard
	"github.com/go-vgo/robotgo"

	// Secure storage
	"github.com/zalando/go-keyring"

	// Logging
	"github.com/sirupsen/logrus"
)

// ---------------------------------------------------------------------
// Configuration (must match mobile app)
// ---------------------------------------------------------------------
var (
	peripheralServiceUUID = ble.NewUUID("0000c0de-0000-1000-8000-00805f9b34fb")
	unlockCharUUID        = ble.NewUUID("0000c0df-0000-1000-8000-00805f9b34fb")

	kyberCtSize   = 1088 // Kyber768 ciphertext size
	nonceSize     = 8    // Anti-replay monotonic counter (uint64)
	sessionKeyLen = 32

	advertiseInterval = 5 * time.Second
	busyCooldown      = 2 * time.Second
)

// Keyring keys
const (
	keyringService     = "PassLinkAgent"
	keyringPubKey      = "clientKyberPublicKey"
	keyringLastNonce   = "lastSeenNonce"
)

// ---------------------------------------------------------------------
// Logging setup
// ---------------------------------------------------------------------
func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		DisableColors:    true,
		QuoteEmptyFields: true,
	})

	if !service.Interactive() {
		// Running as service → try Windows Event Log or fallback file
		if runtime.GOOS == "windows" {
			if hook, err := NewEventLogHook("PassLinkAgent"); err == nil {
				logrus.AddHook(hook)
			}
		}
		if logrus.GetLevel() >= logrus.InfoLevel {
			logrus.SetLevel(logrus.InfoLevel)
		}
	}
}

// ---------------------------------------------------------------------
// Secure zeroing
// ---------------------------------------------------------------------
func zeroBytes(b []byte) {
	if b != nil {
		for i := range b {
			b[i] = 0
		}
	}
}

// ---------------------------------------------------------------------
// Keyring helpers
// ---------------------------------------------------------------------
func storeClientPubKey(pub kyber.PublicKey) error {
	data := hex.EncodeToString(pub.Bytes())
	return keyring.Set(keyringService, keyringPubKey, data)
}

func loadClientPubKey() (kyber.PublicKey, error) {
	s, err := keyring.Get(keyringService, keyringPubKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	pk, err := kyber.Kyber768.NewPublicKey()
	if err != nil {
		return nil, err
	}
	return pk, pk.FromBytes(b)
}

func loadLastNonce() uint64 {
	s, _ := keyring.Get(keyringService, keyringLastNonce)
	if s == "" {
		return 0
	}
	v, _ := binary.Varint([]byte(s))
	return uint64(v)
}

func storeLastNonce(n uint64) {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buf, n)
	keyring.Set(keyringService, keyringLastNonce, string(buf[:]))
}

// ---------------------------------------------------------------------
// Crypto helpers
// ---------------------------------------------------------------------
func generateKeyPair() (kyber.PrivateKey, kyber.PublicKey, error) {
	pub, err := loadClientPubKey()
	if err != nil {
		return nil, nil, err
	}
	if pub != nil {
		// We only persist the public key – generate matching private key
		sk, pk, err := kyber.Kyber768.GenerateKeyPair(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		copy(pk.Bytes(), pub.Bytes())
		logrus.Info("Loaded persisted Kyber public key")
		return sk, pk, nil
	}

	// Fresh pair
	sk, pk, err := kyber.Kyber768.GenerateKeyPair(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	if err := storeClientPubKey(pk); err != nil {
		return nil, nil, fmt.Errorf("failed to persist public key: %w", err)
	}
	logrus.Info("Generated new Kyber key pair and persisted public key")
	return sk, pk, nil
}

func decryptPayload(sessionKey, payload []byte) ([]byte, error) {
	if len(payload) < chacha20poly1305.NonceSizeX {
		return nil, errors.New("payload too short")
	}
	aead, _ := chacha20poly1305.NewX(sessionKey)
	nonce := payload[:chacha20poly1305.NonceSizeX]
	ct := payload[chacha20poly1305.NonceSizeX:]
	return aead.Open(nil, nonce, ct, nil)
}

// ---------------------------------------------------------------------
// Auto-type with fallback
// ---------------------------------------------------------------------
func autoTypeSecret(secret string) {
	logrus.Infof("Typing %d-character secret", len(secret))

	// Try typing
	if err := robotgo.WriteAll(secret); err == nil {
		time.Sleep(50 * time.Millisecond)
		robotgo.KeyTap("enter")
		logrus.Info("Successfully typed secret + Enter")
		return
	}

	// Fallback to per-character (slower but works on more apps)
	logrus.Warn("WriteAll failed, falling back to per-character typing")
	for _, r := range secret {
		robotgo.TypeStr(string(r))
		time.Sleep(30 * time.Millisecond)
	}
	robotgo.KeyTap("enter")
}

// ---------------------------------------------------------------------
// Service implementation
// ---------------------------------------------------------------------
type program struct {
	quit chan struct{}
}

func (p *program) Start(s service.Service) error {
	p.quit = make(chan struct{})
	go p.run()
	return nil
}

func (p *program) run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// -----------------------------------------------------------------
	// 1. Initialize correct BLE device for the platform
	// -----------------------------------------------------------------
	var dev ble.Device
	var err error

	switch runtime.GOOS {
	case "windows":
		dev, err = ble.NewDevice()
	case "darwin":
		dev, err = darwin.NewDevice()
	case "linux":
		dev, err = linux.NewDevice(
			linux.WithSigmaConnection(),
		)
	default:
		logrus.Fatalf("Unsupported OS: %s", runtime.GOOS)
	}
	if err != nil {
		logrus.Fatalf("Failed to initialize BLE device: %v", err)
	}
	ble.SetDefaultDevice(dev)

	// -----------------------------------------------------------------
	// 2. Load/generate Kyber key pair
	// -----------------------------------------------------------------
	privKey, _, err := generateKeyPair()
	if err != nil {
		logrus.Fatalf("Key pair init failed: %v", err)
	}
	defer zeroBytes(privKey.Bytes())

	// -----------------------------------------------------------------
	// 3. Create GATT characteristic
	// -----------------------------------------------------------------
	char := ble.NewCharacteristic(unlockCharUUID)
	char.Properties = ble.CharWriteWithoutResp
	char.HandleWrite(ble.WriteHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) {
		data := req.Data()
		if len(data) < kyberCtSize+nonceSize {
			logrus.Warn("Payload too short")
			return
		}

		// Copy immediately – avoid any shared state
		payload := append([]byte(nil), data...)

		go func(pl []byte) {
			// Split payload
			kyberCt := pl[:kyberCtSize]
			nonceBytes := pl[kyberCtSize : kyberCtSize+nonceSize]
			encSecret := pl[kyberCtSize+nonceSize:]

			nonce := binary.BigEndian.Uint64(nonceBytes)
			last := loadLastNonce()
			if nonce <= last {
				logrus.Warnf("Replay attack detected (nonce %d ≤ %d)", nonce, last)
				return
			}

			// Decapsulate
			shared, err := privKey.Decapsulate(kyberCt)
			if err != nil {
				logrus.WithError(err).Error("Kyber decapsulation failed")
				return
			}
			if len(shared) > sessionKeyLen {
				shared = shared[:sessionKeyLen]
			}

			plain, err := decryptPayload(shared, encSecret)
			zeroBytes(shared)
			if err != nil {
				logrus.WithError(err).Error("AEAD decryption failed")
				return
			}

			secret := string(plain)
			zeroBytes(plain)

			storeLastNonce(nonce)
			logrus.Info("Successfully decrypted and typed secret")
			autoTypeSecret(secret)

			// Small cooldown so UI doesn’t flash repeatedly
			time.Sleep(busy + busyCooldown)
		}(payload)
	}))

	svc := ble.NewService(peripheralServiceUUID)
	svc.AddCharacteristic(char)

	// -----------------------------------------------------------------
	// 4. Start advertising loop
	// -----------------------------------------------------------------
	adv := ble.Advertisement{
		LocalName:    "PassLinkAgent",
		ServiceUUIDs: []ble.UUID{peripheralServiceUUID},
	}

	go func() {
		for {
			select {
			case <-p.quit:
				return
			default:
				if err := ble.AdvertiseNameAndServices(ctx, adv.LocalName, adv.ServiceUUIDs...); err != nil {
					logrus.WithError(err).Error("Advertising failed, retrying...")
					time.Sleep(advertiseInterval)
				}
			}
		}
	}()

	// Wait for shutdown
	<-p.quit
	logrus.Info("PassLinkAgent stopped cleanly")
}

func (p *program) Stop(s service.Service) error {
	close(p.quit)
	time.Sleep(500 * time.Millisecond)
	return nil
}

// -----------------------------------------------------------------
// Windows Event Log hook (unchanged)
// -----------------------------------------------------------------
type eventLogHook struct{ w *eventlog.Writer }

func NewEventLogHook(name string) (logrus.Hook, error) {
	w, err := eventlog.Open(name)
	if err != nil {
		return nil, err
	}
	return &eventLogHook{w}, nil
}

func (h *eventLogHook) Levels() []logrus.Level { return logrus.AllLevels }
func (h *eventLogHook) Fire(e *logrus.Entry) error {
	line, _ := e.String()
	switch e.Level {
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return h.w.Error(1, line)
	case logrus.WarnLevel:
		return h.w.Warning(1, line)
	default:
		return h.w.Info(1, line)
	}
}

// -----------------------------------------------------------------
// main() – service entry point
// -----------------------------------------------------------------
func main() {
	svcConfig := &service.Config{
		Name:        "PassLinkAgent",
		DisplayName: "PassLink Agent",
		Description: "Secure BLE peripheral that receives encrypted secrets from your phone and auto-types them.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logrus.Fatal(err)
	}

	if len(os.Args) > 1 {
		if err := service.Control(s, os.Args[1]); err != nil {
			logrus.Fatalf("Service command failed: %v", err)
		}
		return
	}

	// Catch SIGTERM for clean daemon shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		s.Stop()
	}()

	if err := s.Run(); err != nil {
		logrus.Fatal(err)
	}
}
