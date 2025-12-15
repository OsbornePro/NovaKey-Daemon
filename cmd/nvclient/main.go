// cmd/nvclient/main.go
package main

import (
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"filippo.io/mlkem768"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	// IMPORTANT: Outer v3 msgType must remain 1 so the server accepts the frame.
	// The "approve vs inject" distinction is carried in the INNER typed message frame.
	outerMsgTypePassword = 1

	innerFrameVersionV1 = 1
	innerMsgTypeInject  = 1
	innerMsgTypeApprove = 2
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  nvclient arm [--addr 127.0.0.1:60769] [--token_file arm_token.txt] [--ms 20000]\n")
	fmt.Fprintf(os.Stderr, "  nvclient approve [flags]            (send typed APPROVE control message)\n")
	fmt.Fprintf(os.Stderr, "  nvclient [flags]                    (send typed INJECT/password message)\n\n")
	fmt.Fprintf(os.Stderr, "Common flags:\n")
	fmt.Fprintf(os.Stderr, "  -addr                 NovaKey server address (host:port)\n")
	fmt.Fprintf(os.Stderr, "  -device-id            device ID to use\n")
	fmt.Fprintf(os.Stderr, "  -key-hex              hex-encoded 32-byte per-device key (matches devices.json)\n")
	fmt.Fprintf(os.Stderr, "  -server-kyber-pub-b64 base64 ML-KEM-768 public key (kyber768_public)\n")
	fmt.Fprintf(os.Stderr, "  -password             secret to send (inject only)\n\n")
}

type commonArgs struct {
	addr              string
	deviceID          string
	keyHex            string
	serverKyberPubB64 string
	password          string
}

func parseCommon(fs *flag.FlagSet) *commonArgs {
	c := &commonArgs{}
	fs.StringVar(&c.addr, "addr", "127.0.0.1:60768", "NovaKey server address (host:port)")
	fs.StringVar(&c.password, "password", "hello", "password/secret to send (inject only)")
	fs.StringVar(&c.deviceID, "device-id", "roberts-phone", "device ID to use")
	fs.StringVar(&c.keyHex, "key-hex", "", "hex-encoded 32-byte per-device key (matches devices.json)")
	fs.StringVar(&c.serverKyberPubB64, "server-kyber-pub-b64", "", "base64 ML-KEM-768 public key (kyber768_public from server_keys.json or pairing)")
	return c
}

func requireCryptoInputs(c *commonArgs) {
	if c.keyHex == "" {
		log.Fatal("must provide -key-hex (hex-encoded 32-byte key matching server devices.json)")
	}
	if c.serverKyberPubB64 == "" {
		log.Fatal("must provide -server-kyber-pub-b64 (base64 kyber768_public from server_keys.json / pairing)")
	}
	if c.deviceID == "" {
		log.Fatal("must provide -device-id (non-empty)")
	}
}

func main() {
	// Subcommand dispatch BEFORE parsing default flags.
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "arm":
			code := cmdArm(os.Args[2:])
			os.Exit(code)
			return
		case "approve":
			os.Exit(cmdApprove(os.Args[2:]))
			return
		}
	}

	fs := flag.NewFlagSet("nvclient", flag.ExitOnError)
	fs.Usage = usage

	// Make "--help" probes succeed (test scripts often do this)
	help := fs.Bool("h", false, "show help")
	help2 := fs.Bool("help", false, "show help")

	c := parseCommon(fs)
	_ = fs.Parse(os.Args[1:])

	if *help || *help2 {
		usage()
		os.Exit(0)
		return
	}

	requireCryptoInputs(c)

	if err := initCryptoClient(c.deviceID, c.keyHex, c.serverKyberPubB64); err != nil {
		log.Fatalf("initCryptoClient failed: %v", err)
	}

	// Always send typed inner frame (no legacy mode).
	inner, err := encodeInnerMessageFrame(c.deviceID, innerMsgTypeInject, []byte(c.password))
	if err != nil {
		log.Fatalf("encodeInnerMessageFrame: %v", err)
	}

	if err := sendV3OuterFrame(c.addr, inner); err != nil {
		log.Fatalf("send failed: %v", err)
	}
	fmt.Printf("sent v3 Kyber+XChaCha encrypted password frame (inner msgType=%d)\n", innerMsgTypeInject)
}

func cmdApprove(args []string) int {
	fs := flag.NewFlagSet("approve", flag.ContinueOnError)
	fs.Usage = usage
	fs.SetOutput(os.Stdout)

	// Accept -h/--help so test_send.sh can probe support reliably.
	help := fs.Bool("h", false, "show help")
	help2 := fs.Bool("help", false, "show help")

	c := parseCommon(fs)
	if err := fs.Parse(args); err != nil {
		// If they asked for help, treat as success
		if *help || *help2 {
			usage()
			return 0
		}
		return 2
	}

	if *help || *help2 {
		usage()
		return 0
	}

	requireCryptoInputs(c)

	if err := initCryptoClient(c.deviceID, c.keyHex, c.serverKyberPubB64); err != nil {
		fmt.Fprintf(os.Stderr, "initCryptoClient failed: %v\n", err)
		return 1
	}

	inner, err := encodeInnerMessageFrame(c.deviceID, innerMsgTypeApprove, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encodeInnerMessageFrame failed: %v\n", err)
		return 1
	}

	if err := sendV3OuterFrame(c.addr, inner); err != nil {
		fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
		return 1
	}

	fmt.Printf("sent v3 Kyber+XChaCha encrypted APPROVE control frame (inner msgType=%d)\n", innerMsgTypeApprove)
	return 0
}

// sendV3OuterFrame sends a single v3 frame to the daemon: [u16 length][payload] over TCP4.
func sendV3OuterFrame(addr string, innerBody []byte) error {
	frame, err := encryptV3OuterFrame(innerBody)
	if err != nil {
		return err
	}
	if len(frame) > 0xFFFF {
		return fmt.Errorf("frame too large: %d", len(frame))
	}

	conn, err := net.Dial("tcp4", addr)
	if err != nil {
		return fmt.Errorf("Dial: %w", err)
	}
	defer conn.Close()

	var hdr [2]byte
	binary.BigEndian.PutUint16(hdr[:], uint16(len(frame)))

	if _, err := conn.Write(hdr[:]); err != nil {
		return fmt.Errorf("write length: %w", err)
	}
	if _, err := conn.Write(frame); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	return nil
}

// encryptV3OuterFrame builds the v3 payload:
//
//   header = version || outerMsgType(=1) || idLen || deviceID || kemCtLen || kemCt   (AAD)
//   plaintext = timestamp(u64be) || innerBody
//   out = header || nonce || aead(ciphertext)
func encryptV3OuterFrame(innerBody []byte) ([]byte, error) {
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

	kemCt, sharedKem, err := mlkem768.Encapsulate(serverEncapKey)
	if err != nil {
		return nil, fmt.Errorf("mlkem768.Encapsulate: %w", err)
	}
	if len(kemCt) != mlkem768.CiphertextSize {
		return nil, fmt.Errorf("internal: kemCt length %d != CiphertextSize %d", len(kemCt), mlkem768.CiphertextSize)
	}

	aeadKey, err := deriveAEADKey(deviceStaticKey, sharedKem)
	if err != nil {
		return nil, err
	}

	aead, err := chacha20poly1305.NewX(aeadKey)
	if err != nil {
		return nil, fmt.Errorf("NewX with derived key failed: %w", err)
	}

	header := make([]byte, 0, 3+len(idBytes)+2+len(kemCt))
	header = append(header, protocolVersion)
	header = append(header, byte(outerMsgTypePassword))
	header = append(header, idLen)
	header = append(header, idBytes...)

	var kemLenBuf [2]byte
	binary.BigEndian.PutUint16(kemLenBuf[:], uint16(len(kemCt)))
	header = append(header, kemLenBuf[:]...)
	header = append(header, kemCt...)

	now := time.Now().Unix()
	plaintext := make([]byte, 8+len(innerBody))
	binary.BigEndian.PutUint64(plaintext[:8], uint64(now))
	copy(plaintext[8:], innerBody)

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("rand.Read nonce: %w", err)
	}

	ct := aead.Seal(nil, nonce, plaintext, header)

	out := make([]byte, 0, len(header)+len(nonce)+len(ct))
	out = append(out, header...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func encodeInnerMessageFrame(deviceID string, msgType uint8, payload []byte) ([]byte, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("deviceID required")
	}
	if msgType != innerMsgTypeInject && msgType != innerMsgTypeApprove {
		return nil, fmt.Errorf("invalid inner msgType=%d", msgType)
	}
	if payload == nil {
		payload = []byte{}
	}

	dev := []byte(deviceID)
	if len(dev) > 0xFFFF {
		return nil, fmt.Errorf("deviceID too long")
	}

	out := make([]byte, 0, 1+1+2+4+len(dev)+len(payload))
	out = append(out, byte(innerFrameVersionV1))
	out = append(out, byte(msgType))

	var tmp2 [2]byte
	binary.BigEndian.PutUint16(tmp2[:], uint16(len(dev)))
	out = append(out, tmp2[:]...)

	var tmp4 [4]byte
	binary.BigEndian.PutUint32(tmp4[:], uint32(len(payload)))
	out = append(out, tmp4[:]...)

	out = append(out, dev...)
	out = append(out, payload...)
	return out, nil
}

