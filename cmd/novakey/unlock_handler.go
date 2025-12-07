package main

import (
	"fmt"
	"io"
	"net"

	"github.com/cloudflare/circl/kem/kyber/kyber768"
)

// kyberCtSize is defined in crypto_shared.go — DO NOT redeclare here
// const kyberCtSize = 1088   ← DELETE THIS LINE

func handleConn(conn net.Conn, priv *kyber768.PrivateKey) {
	defer conn.Close()

	data, err := io.ReadAll(conn)
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

	sharedSecret, err := Decapsulate(priv, ct)
	if err != nil {
		LogError("Decapsulation failed", err)
		return
	}

	plain, err := DecryptPayload(sharedSecret, encPayload)
	if err != nil {
		LogError("DecryptPayload failed", err)
		return
	}

	HandlePayload(plain)
}

func HandlePayload(data []byte) {
	password := string(data)
	fmt.Printf("Auto-typing password (%d chars)...\n", len(password))
	TypeString(password)
}
