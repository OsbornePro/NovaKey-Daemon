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
		serverPubPath = flag.String("server-pub", "", "Path to server Kyber public key (Option A)")
		serverPubB64  = flag.String("server-pub-base64", "", "Base64 server Kyber public key (Option B)")
		outPath       = flag.String("out", "pairing.json", "Output file")
		host          = flag.String("host", "127.0.0.1", "NovaKey host")
		port          = flag.Int("port", 60768, "NovaKey port")
	)
	flag.Parse()

	// ---- argument validation ----

	if *serverPubPath == "" && *serverPubB64 == "" {
		fatal("You must specify either --server-pub or --server-pub-base64")
	}

	if *serverPubPath != "" && *serverPubB64 != "" {
		fatal("Specify only one of --server-pub or --server-pub-base64")
	}

	// ---- load server public key ----

	var pubBytes []byte
	var err error

	if *serverPubB64 != "" {
		// ✅ OPTION B: Base64 string (preferred)
		pubBytes, err = base64.StdEncoding.DecodeString(*serverPubB64)
		if err != nil {
			fatal("Invalid base64 server public key", err)
		}
	}

	/*
		// ⛔ OPTION A (kept for rollback, currently disabled):
		if *serverPubPath != "" {
			pubBytes, err = os.ReadFile(*serverPubPath)
			if err != nil {
				fatal("Failed to read server public key file", err)
			}
		}
	*/

	// ---- generate pairing data ----

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

	// ---- write output (binary, no BOM, no shell encoding nonsense) ----

	if err := os.WriteFile(*outPath, payload, 0600); err != nil {
		fatal("Failed to write pairing file", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Pairing file written to %s\n", *outPath)
}

// ------------------------------------------------------------

func fatal(msg string, err ...error) {
	fmt.Fprintln(os.Stderr, "❌", msg)
	if len(err) > 0 && err[0] != nil {
		fmt.Fprintln(os.Stderr, err[0])
	}
	os.Exit(1)
}
