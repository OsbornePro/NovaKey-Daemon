package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func cmdArm(args []string) int {
	fs := flag.NewFlagSet("arm", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:60769", "arm API address")
	tokenFile := fs.String("token_file", "arm_token.txt", "path to arm token file")
	ms := fs.Int("ms", 20000, "arm duration in ms")
	header := fs.String("header", "X-NovaKey-Token", "token header name")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	b, err := os.ReadFile(*tokenFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read token file: %v\n", err)
		return 1
	}
	token := strings.TrimSpace(string(b))
	if token == "" {
		fmt.Fprintf(os.Stderr, "token file is empty: %s\n", *tokenFile)
		return 1
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/arm?ms=%d", *addr, *ms), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build request failed: %v\n", err)
		return 1
	}
	req.Header.Set(*header, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("%s", body)

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

