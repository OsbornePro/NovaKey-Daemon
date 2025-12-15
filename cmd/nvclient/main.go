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
	msgTypeInject  = 1
	msgTypeApprove = 2
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  nvclient arm [--addr 127.0.0.1:60769] [--token_file arm_token.txt] [--ms 20000]\n")
	fmt.Fprintf(os.Stderr, "  nvclient approve [flags] [--legacy_magic] [--magic __NOVAKEY_APPROVE__]\n")
	fmt.Fprintf(os.Stderr, "  nvclient [flags]   (send inject/password frame)\n\n")
	fmt.Fprintf(os.Stderr, "Common flags:\n")
	fmt.Fprintf(os.Stderr, "  -addr                 NovaKey server address (host:port)\n")
	fmt.Fprintf(os.Stderr, "  -device-id            device ID to use\n")
	fmt.Fprintf(os.Stderr, "  -key-hex              hex-encoded 32-byte per-device key (matches devices.json)\n")
	fmt.Fprintf(os.Stderr, "  -server-kyber-pub-b64 base64 ML-KEM-768 public key (kyber768_public)\n")
	fmt.Fprintf(os.Stderr, "  -password             secret to send (inject only)\n\n")
}

type commonArgs struct {
	addr                 string
	deviceID             string
	keyHex               string
	serverKyberPubB64    string
	password             string
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
}

func main() {
	// Subcommand dispatch BEFORE flag.Parse()
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
	c := parseCommon(fs)
	fs.Parse(os.Args[1:])

	requireCryptoInputs(c)

	if err := initCryptoClient(c.deviceID, c.keyHex, c.serverKyberPubB64); err != nil {
		log.Fatalf("initCryptoClient failed: %v", err)
	}

	if err := sendV3Frame(c.addr, msgTypeInject, []byte(c.password)); err != nil {
		log.Fatalf("send failed: %v", err)
	}
	fmt.Printf("sent v3 Kyber+XChaCha encrypted password frame (msgType=%d)\n", msgTypeInject)
}

func cmdApprove(args []string) int {
	fs := flag.NewFlagSet("approve", flag.ContinueOnError)
	fs.Usage = usage

	c := parseCommon(fs)

	legacyMagic := fs.Bool("legacy_magic", false, "send legacy approve as msgType=1 with payload==magic (for older servers)")
	magic := fs.String("magic", "__NOVAKEY_APPROVE__", "legacy approve magic payload (used only with --legacy_magic)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	requireCryptoInputs(c)

	if err := initCryptoClient(c.deviceID, c.keyHex, c.serverKyberPubB64); err != nil {
		fmt.Fprintf(os.Stderr, "initCryptoClient failed: %v\n", err)
		return 1
	}

	mt := uint8(msgTypeApprove)
	payload := []byte{}
	label := "APPROVE control"

	if *legacyMagic {
		mt = uint8(msgTypeInject)
		payload = []byte(*magic)
		label = "LEGACY APPROVE magic"
	}

	if err := sendV3Frame(c.addr, mt, payload); err != nil {
		fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
		return 1
	}

	fmt.Printf("sent v3 Kyber+XChaCha encrypted %s frame (msgType=%d)\n", label, mt)
	return 0
}

// sendV3Frame sends a single v3 frame to the daemon: [u16 length][payload] over TCP4.
func sendV3Frame(addr string, msgType uint8, payload []byte) error {
	frame, err := encryptV3Frame(msgType, payload)
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

// encryptV3Frame builds the v3 payload:
//   header = version || msgType || idLen || deviceID || kemCtLen || kemCt   (AAD)
//   plaintext = timestamp(u64be) || payloadBytes
//   out = header || nonce || aead(ciphertext)
//
// NOTE: Relies on globals initialized by initCryptoClient():
//   clientDeviceID, deviceStaticKey, serverEncapKey, deriveAEADKey()
func encryptV3Frame(msgType uint8, payload []byte) ([]byte, error) {
	if deviceStaticKey == nil || len(deviceStaticKey) == 0 {
		return nil, fmt.Errorf("device static key not initialized")
	}
	if serverEncapKey == nil || len(serverEncapKey) == 0 {
		return nil, fmt.Errorf("server Kyber public key not initialized")
	}
	if msgType != msgTypeInject && msgType != msgTypeApprove {
		return nil, fmt.Errorf("invalid msgType=%d", msgType)
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

	// AAD header: version || msgType || idLen || deviceID || kemCtLen || kemCt
	header := make([]byte, 0, 3+len(idBytes)+2+len(kemCt))
	header = append(header, protocolVersion)
	header = append(header, byte(msgType))
	header = append(header, idLen)
	header = append(header, idBytes...)

	var kemLenBuf [2]byte
	binary.BigEndian.PutUint16(kemLenBuf[:], uint16(len(kemCt)))
	header = append(header, kemLenBuf[:]...)
	header = append(header, kemCt...)

	// plaintext = timestamp || payload
	now := time.Now().Unix()
	plaintext := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint64(plaintext[:8], uint64(now))
	copy(plaintext[8:], payload)

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

