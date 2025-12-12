// cmd/nvclient/main.go
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
)

var (
	addr     = flag.String("addr", "127.0.0.1:60768", "NovaKey server address")
	password = flag.String("password", "hello", "password to send")

	deviceIDFlag = flag.String("device-id", "roberts-phone", "device ID to use")
	keyHexFlag   = flag.String("key-hex", "", "hex-encoded 32-byte key for this device")
)

func main() {
	flag.Parse()

	if *keyHexFlag == "" {
		log.Fatal("must provide -key-hex (hex-encoded 32-byte key matching server devices.json)")
	}

	if err := initCryptoClient(*deviceIDFlag, *keyHexFlag); err != nil {
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

	fmt.Println("sent encrypted password frame")
}

