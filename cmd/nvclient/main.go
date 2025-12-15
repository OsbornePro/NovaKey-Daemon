package main

import (
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"filippo.io/mlkem768"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	addr     = flag.String("addr", "127.0.0.1:60768", "NovaKey server address (host:port)")
	password = flag.String("password", "hello", "password/secret to send (or approve magic if using two-man)")

	deviceIDFlag          = flag.String("device-id", "roberts-phone", "device ID to use")
	keyHexFlag            = flag.String("key-hex", "", "hex-encoded 32-byte per-device key (matches devices.json)")
	serverKyberPubB64Flag = flag.String("server-kyber-pub-b64", "", "base64 ML-KEM-768 public key (kyber768_public from server_keys.json or pairing)")

	// Two-man approve support
	approve       = flag.Bool("approve", false, "send an approve control frame (msgType=2) instead of a password (msgType=1)")
	approveMagic  = flag.String("approve-magic", "__NOVAKEY_APPROVE__", "if -password equals this and -force-password is false, nvclient sends msgType=2 approve")
	forcePassword = flag.Bool("force-password", false, "if true, never auto-convert approve-magic into msgType=2; always send as a normal password")
)

const (
	// Server-side expects msgTypePassword=1 and msgTypeApprove=2 in v3 framing.
	msgTypeApprove = 2
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  nvclient arm [--addr 127.0.0.1:60769] [--token_file arm_token.txt] [--ms 20000]\n")
	fmt.Fprintf(os.Stderr, "  nvclient [flags]   (send encrypted frame)\n\n")
	fmt.Fprintf(os.Stderr, "Two-man approve:\n")
	fmt.Fprintf(os.Stderr, "  nvclient -password \"__NOVAKEY_APPROVE__\" ...      (auto-sends msgType=2 approve)\n")
	fmt.Fprintf(os.Stderr, "  nvclient -approve ...                             (explicit msgType=2 approve)\n")
	fmt.Fprintf(os.Stderr, "  nvclient -force-password -password \"__NOVAKEY_APPROVE__\" ... (send as normal password)\n\n")
	flag.PrintDefaults()
}

func main() {
	// Subcommand dispatch BEFORE flag.Parse()
	if len(os.Args) >= 2 && os.Args[1] == "arm" {
		code := cmdArm(os.Args[2:])
		os.Exit(code)
		return
	}

	flag.Usage = usage
	flag.Parse()

	if *keyHexFlag == "" {
		log.Fatal("must provide -key-hex (hex-encoded 32-byte key matching server devices.json)")
	}
	if *serverKyberPubB64Flag == "" {
		log.Fatal("must provide -server-kyber-pub-b64 (base64 kyber768_public from server_keys.json / pairing)")
	}

	if err := initCryptoClient(*deviceIDFlag, *keyHexFlag, *serverKyberPubB64Flag); err != nil {
		log.Fatalf("initCryptoClient failed: %v", err)
	}

	// Decide whether we're sending approve or password.
	wantApprove := *approve
	if !wantApprove && !*forcePassword && *password == *approveMagic {
		// Backward-compatible behavior: your scripts already send approve_magic as "password".
		// We auto-upgrade that into a real msgType=Approve control frame.
		wantApprove = true
	}

	var (
		frame []byte
		err   error
	)

	if wantApprove {
		frame, err = encryptMessageFrame(byte(msgTypeApprove), nil)
		if err != nil {
			log.Fatalf("encrypt approve frame: %v", err)
		}
	} else {
		// Normal password frame
		frame, err = encryptMessageFrame(byte(msgTypePassword), []byte(*password))
		if err != nil {
			log.Fatalf("encrypt password frame: %v", err)
		}
	}

	if len(frame) > 0xFFFF {
		log.Fatalf("frame too large: %d", len(frame))
	}

	conn, err := net.Dial("tcp4", *addr)
	if err != nil {
		log.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	var hdr [2]byte
	binary.BigEndian.PutUint16(hdr[:], uint16(len(frame)))

	if _, err := conn.Write(hdr[:]); err != nil {
		log.Fatalf("write length: %v", err)
	}
	if _, err := conn.Write(frame); err != nil {
		log.Fatalf("write frame: %v", err)
	}

	if wantApprove {
		fmt.Println("sent v3 Kyber+XChaCha encrypted APPROVE control frame (msgType=2)")
	} else {
		fmt.Println("sent v3 Kyber+XChaCha encrypted password frame (msgType=1)")
	}
}

// encryptMessageFrame builds the v3 frame with a specified msgType.
// Plaintext is always: timestamp(uint64 BE) || payloadBytes
func encryptMessageFrame(msgType byte, payload []byte) ([]byte, error) {
	if deviceStaticKey == nil || len(deviceStaticKey) == 0 {
		return nil, fmt.Errorf("device static key not initialized")
	}
	if serverEncapKey == nil || len(serverEncapKey) == 0 {
		return nil, fmt.Errorf("server Kyber public key not initialized")
	}

	idBytes := []byte(clientDeviceID)
	if len(idBytes) == 0 || len(idBytes) > 255 {
		return nil, fmt.Errorf("deviceID length invalid: %d (must be 1..255)", len(idBytes))
	}
	idLen := byte(len(idBytes))

	// 1) KEM encapsulation to get (kemCt, sharedKem)
	kemCt, sharedKem, err := mlkem768.Encapsulate(serverEncapKey)
	if err != nil {
		return nil, fmt.Errorf("mlkem768.Encapsulate: %w", err)
	}

	// 2) Derive AEAD key from sharedKem + deviceStaticKey (same as server)
	aeadKey, err := deriveAEADKey(deviceStaticKey, sharedKem)
	if err != nil {
		return nil, err
	}

	aead, err := chacha20poly1305.NewX(aeadKey)
	if err != nil {
		return nil, fmt.Errorf("NewX with derived key failed: %w", err)
	}

	// 3) Build header (AAD)
	// header = version || msgType || idLen || deviceID || kemCtLen || kemCt
	header := make([]byte, 0, 3+len(idBytes)+2+len(kemCt))
	header = append(header, protocolVersion)
	header = append(header, msgType)
	header = append(header, idLen)
	header = append(header, idBytes...)

	var kemLenBuf [2]byte
	binary.BigEndian.PutUint16(kemLenBuf[:], uint16(len(kemCt)))
	header = append(header, kemLenBuf[:]...)
	header = append(header, kemCt...)

	// 4) Plaintext = timestamp || payload
	now := time.Now().Unix()
	plaintext := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint64(plaintext[:8], uint64(now))
	copy(plaintext[8:], payload)

	// 5) Nonce + AEAD
	nonceLen := aead.NonceSize()
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("rand nonce: %w", err)
	}

	ct := aead.Seal(nil, nonce, plaintext, header)

	// 6) Final frame
	out := make([]byte, 0, len(header)+len(nonce)+len(ct))
	out = append(out, header...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

