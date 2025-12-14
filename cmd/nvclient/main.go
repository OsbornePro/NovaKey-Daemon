package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

var (
	addr     = flag.String("addr", "127.0.0.1:60768", "NovaKey server address (host:port)")
	password = flag.String("password", "hello", "password/secret to send")

	deviceIDFlag          = flag.String("device-id", "roberts-phone", "device ID to use")
	keyHexFlag            = flag.String("key-hex", "", "hex-encoded 32-byte per-device key (matches devices.json)")
	serverKyberPubB64Flag = flag.String("server-kyber-pub-b64", "", "base64 ML-KEM-768 public key (kyber768_public from server_keys.json or pairing)")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  nvclient arm [--addr 127.0.0.1:60769] [--token_file arm_token.txt] [--ms 20000]\n")
	fmt.Fprintf(os.Stderr, "  nvclient [flags]   (send encrypted password frame)\n\n")
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

	frame, err := encryptPasswordFrame(*password)
	if err != nil {
		log.Fatalf("encryptPasswordFrame: %v", err)
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

	fmt.Println("sent v3 Kyber+XChaCha encrypted password frame")
}

