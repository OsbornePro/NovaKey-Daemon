// cmd/novakey/main.go
// NovaKey – Quantum-resistant BLE password filler
// Works on Linux with go-ble/ble and crypto/mlkem (Go 1.24+)

package main

import (
	"context"
	"crypto/mlkem"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/kardianos/service"
	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"

	"golang.org/x/crypto/chacha20poly1305"
	"github.com/go-vgo/robotgo"
	"github.com/zalando/go-keyring"
	"github.com/sirupsen/logrus"
)

// Configuration
var (
	peripheralServiceUUID = ble.MustParse("0000c0de-0000-1000-8000-00805f9b34fb")
	unlockCharUUID        = ble.MustParse("0000c0df-0000-1000-8000-00805f9b34fb")

	kyberCtSize   = mlkem.CiphertextSize768
	nonceSize     = 8
	sessionKeyLen = 32

	advertiseInterval = 5 * time.Second
	busyCooldown      = 2 * time.Second
)

const (
	keyringService   = "NovaKey"
	keyringPubKey    = "clientKyberPublicKey" // stores our public (encapsulation) key (hex)
	keyringLastNonce = "lastSeenNonce"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		DisableColors:    true,
		QuoteEmptyFields: true,
	})
}

func zeroBytes(b []byte) {
	if b != nil {
		for i := range b {
			b[i] = 0
		}
	}
}

// Keyring helpers
func storeClientPubKeyBytes(pub []byte) error {
	return keyring.Set(keyringService, keyringPubKey, hex.EncodeToString(pub))
}

func loadClientPubKeyBytes() ([]byte, error) {
	s, err := keyring.Get(keyringService, keyringPubKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return nil, err
	}
	if s == "" {
		return nil, nil
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func loadLastNonce() uint64 {
	s, _ := keyring.Get(keyringService, keyringLastNonce)
	if s == "" {
		return 0
	}
	// stored as raw varint bytes
	v, _ := binary.Varint([]byte(s))
	if v < 0 {
		return 0
	}
	return uint64(v)
}

func storeLastNonce(n uint64) {
	buf := make([]byte, binary.MaxVarintLen64)
	nbytes := binary.PutUvarint(buf, n)
	_ = keyring.Set(keyringService, keyringLastNonce, string(buf[:nbytes]))
}

// generateKeyPair returns a decapsulation (private) key and its corresponding
// encapsulation (public) key. We persist the public key in the keyring as hex.
func generateKeyPair() (*mlkem.DecapsulationKey768, *mlkem.EncapsulationKey768, error) {
	// Always generate a fresh private (decapsulation) key on startup.
	priv, err := mlkem.GenerateKey768()
	if err != nil {
		return nil, nil, err
	}
	pub := priv.EncapsulationKey()

	// store the public key bytes so clients can read it from keyring if needed
	if err := storeClientPubKeyBytes(pub.Bytes()); err != nil {
		// non-fatal: still return the keypair but surface the error
		logrus.WithError(err).Warn("failed to persist Kyber public key to keyring")
	}

	logrus.Info("Generated new Kyber (ML-KEM-768) key pair")
	return priv, pub, nil
}

func decryptPayload(key, payload []byte) ([]byte, error) {
	if len(payload) < chacha20poly1305.NonceSizeX {
		return nil, errors.New("payload too short")
	}
	aead, _ := chacha20poly1305.NewX(key)
	nonce := payload[:chacha20poly1305.NonceSizeX]
	ct := payload[chacha20poly1305.NonceSizeX:]
	return aead.Open(nil, nonce, ct, nil)
}

func autoTypeSecret(secret string) {
	logrus.Infof("Typing %d-character secret", len(secret))
	if err := robotgo.WriteAll(secret); err == nil {
		time.Sleep(50 * time.Millisecond)
		robotgo.KeyTap("enter")
		logrus.Info("Typed secret + Enter")
		return
	}
	logrus.Warn("WriteAll failed, using slow fallback")
	for _, r := range secret {
		robotgo.TypeStr(string(r))
		time.Sleep(30 * time.Millisecond)
	}
	robotgo.KeyTap("enter")
}

// Write handler
type unlockHandler struct{ priv *mlkem.DecapsulationKey768 }

func (h *unlockHandler) ServeWrite(req ble.Request, rsp ble.ResponseWriter) {
	data := req.Data()
	// check minimal length: ciphertext + nonce
	if len(data) < kyberCtSize+nonceSize {
		rsp.SetStatus(ble.ErrSuccess)
		return
	}
	payload := append([]byte(nil), data...) // copy

	go func(pl []byte) {
		// parse fields
		ct := pl[:kyberCtSize]
		nonce := binary.BigEndian.Uint64(pl[kyberCtSize : kyberCtSize+nonceSize])
		enc := pl[kyberCtSize+nonceSize:]

		if nonce <= loadLastNonce() {
			logrus.Warn("Replay attack blocked")
			return
		}

		// decapsulate to get shared secret
		shared, err := h.priv.Decapsulate(ct)
		if err != nil {
			logrus.WithError(err).Error("Decapsulation failed")
			return
		}

		// truncate/pad to session key length
		if len(shared) > sessionKeyLen {
			shared = shared[:sessionKeyLen]
		} else if len(shared) < sessionKeyLen {
			// unlikely (mlkem.SharedKeySize == 32), but ensure length
			tmp := make([]byte, sessionKeyLen)
			copy(tmp, shared)
			shared = tmp
		}

		plain, err := decryptPayload(shared, enc)
		zeroBytes(shared) // wipe derived shared key
		if err != nil {
			logrus.WithError(err).Error("Decryption failed")
			return
		}

		secret := string(plain)
		zeroBytes(plain)
		storeLastNonce(nonce)
		autoTypeSecret(secret)
		time.Sleep(busyCooldown)
	}(payload)

	rsp.Write(nil)
	rsp.SetStatus(ble.ErrSuccess)
}

// Service
type program struct{ quit chan struct{} }

func (p *program) Start(s service.Service) error {
	p.quit = make(chan struct{})
	go p.run()
	return nil
}

func (p *program) run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dev, err := linux.NewDevice()
	if err != nil {
		logrus.Fatalf("BLE init failed: %v", err)
	}
	ble.SetDefaultDevice(dev)

	priv, _, err := generateKeyPair()
	if err != nil {
		logrus.Fatalf("Key gen failed: %v", err)
	}
	// can't zero internals of mlkem DecapsulationKey768, so don't attempt; trust runtime lifetime

	char := ble.NewCharacteristic(unlockCharUUID)
	char.Property = ble.CharWriteNR
	char.HandleWrite(&unlockHandler{priv: priv})

	svc := ble.NewService(peripheralServiceUUID)
	svc.AddCharacteristic(char)
	ble.AddService(svc)

	go func() {
		for {
			select {
			case <-p.quit:
				return
			default:
				if err := ble.AdvertiseNameAndServices(ctx, "NovaKey", peripheralServiceUUID); err != nil {
					logrus.WithError(err).Error("Advertising failed")
					time.Sleep(advertiseInterval)
				}
			}
		}
	}()

	logrus.Info("NovaKey started – advertising as 'NovaKey'")
	<-p.quit
	logrus.Info("NovaKey stopped")
}

func (p *program) Stop(s service.Service) error {
	close(p.quit)
	time.Sleep(500 * time.Millisecond)
	return nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("NovaKey %s (built %s)\n", version, buildDate)
		os.Exit(0)
	}

	svcConfig := &service.Config{
		Name:        "NovaKey",
		DisplayName: "NovaKey Agent",
		Description: "Quantum-resistant BLE password filler",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logrus.Fatal(err)
	}

	if len(os.Args) > 1 {
		service.Control(s, os.Args[1])
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; s.Stop() }()

	if err := s.Run(); err != nil {
		logrus.Fatal(err)
	}
}

