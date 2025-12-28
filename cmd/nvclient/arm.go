// cmd/nvclient/arm.go
package main

import (
	"flag"
	"fmt"
	"os"
)

func cmdArm(args []string) int {
	fs := flag.NewFlagSet("arm", flag.ContinueOnError)
	fs.Usage = usage
	fs.SetOutput(os.Stdout)

	help := fs.Bool("h", false, "show help")
	help2 := fs.Bool("help", false, "show help")

	c := parseCommon(fs)
	ms := fs.Int("ms", 15000, "arm duration in ms")

	if err := fs.Parse(args); err != nil {
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

	// payload is optional JSON: {"ms":15000}
	payload := []byte(fmt.Sprintf(`{"ms":%d}`, *ms))

	inner, err := encodeInnerMessageFrame(c.deviceID, innerMsgTypeArm, payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encodeInnerMessageFrame failed: %v\n", err)
		return 1
	}

	replyLine, err := sendV3OuterFrame(c.addr, inner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
		return 1
	}

	// Print the raw reply line (scripts/security researchers love this)
	fmt.Print(replyLine)

	// Optional: treat non-success as failure exit code (production behavior)
	if st, ok := parseReplyStatus(replyLine); ok && !st.isSuccess() {
		return 1
	}

	return 0
}
