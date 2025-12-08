package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/OsbornePro/NovaKey/internal/pairing"
)

func main() {
	var (
		serverPubPath = flag.String("server-pub", "", "Path to server Kyber public key")
		outPath       = flag.String("out", "pairing.json", "Output file")
		host          = flag.String("host", "127.0.0.1", "NovaKey host")
		port          = flag.Int("port", 60768, "NovaKey port")
	)
	flag.Parse()

	if *serverPubPath == "" {
		fatal("You must specify --server-pub")
	}

	pubBytes, err := os.ReadFile(*serverPubPath)
	if err != nil {
		fatal("Failed to read server public key", err)
	}

	// Allow base64 or raw
	if decoded, err := base64.StdEncoding.DecodeString(string(pubBytes)); err == nil {
		pubBytes = decoded
	}

	deviceID := pairing.GenerateDeviceID()
	secret, err := pairing.GenerateDeviceSecret()
	if err != nil {
		fatal("Failed to generate device secret", err)
	}

	payload, err := pairing.BuildPairingPayload(
		deviceID,
		secret,
		pubBytes,
		*host,
		*port,
	)
	if err != nil {
		fatal("Failed to build pairing payload", err)
	}

	// ✅ Write bytes directly (UTF-8, no BOM, no shell interference)
	if err := os.WriteFile(*outPath, payload, 0600); err != nil {
		fatal("Failed to write pairing file", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Pairing file written to %s\n", *outPath)
}

func fatal(msg string, err ...error) {
	fmt.Fprintln(os.Stderr, "❌", msg)
	if len(err) > 0 && err[0] != nil {
		fmt.Fprintln(os.Stderr, err[0])
	}
	os.Exit(1)
}
