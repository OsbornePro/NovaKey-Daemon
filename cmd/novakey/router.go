// cmd/novakey/router.go
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const (
	routerMagic  = "NOVAK/1"
	routerMaxHdr = 1024 // max bytes to read for the first route line
)

// startUnifiedListener starts a single TCP listener (cfg.ListenAddr, default :60768)
// and routes each connection based on the first line:
//
//   "NOVAK/1 /pair\n"  -> handlePairConn
//   "NOVAK/1 /msg\n"   -> handleMsgConn (your existing message flow)
//   (anything else)    -> fallback to /msg (backwards compatible)
//
// Drop-in usage:
//   - call startUnifiedListener() from main after initCrypto()
//   - implement handleMsgConn(conn) to wrap your current 60768 handler.
func startUnifiedListener() error {
	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.ListenAddr, err)
	}
	log.Printf("[net] listening on %s (routes: /pair, /msg)", cfg.ListenAddr)

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				log.Printf("[net] accept: %v", err)
				continue
			}
			go routeConn(c)
		}
	}()
	return nil
}

func routeConn(conn net.Conn) {
	defer conn.Close()

	// Avoid hangs if client connects and never sends.
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	br := bufio.NewReaderSize(conn, 4096)

	// Peek up to routerMaxHdr, but do not block forever; ReadString reads until '\n'.
	line, err := readRouteLine(br)
	if err != nil {
		// If we couldn't read a route line, try treating it as a raw /msg client.
		_ = conn.SetReadDeadline(time.Time{})
		if err := handleMsgConn(newPreReadConn(conn, br)); err != nil {
			log.Printf("[net] /msg fallback error: %v", err)
		}
		return
	}

	// Clear read deadline for handler.
	_ = conn.SetReadDeadline(time.Time{})

	route := parseRoute(line)
	switch route {
	case "/pair":
		if err := handlePairConn(newPreReadConn(conn, br)); err != nil {
			log.Printf("[pair] conn error: %v", err)
		}
	case "/msg":
		if err := handleMsgConn(newPreReadConn(conn, br)); err != nil {
			log.Printf("[msg] conn error: %v", err)
		}
	default:
		// Backwards compatible: if a legacy phone client connects and immediately
		// starts writing frames, it won’t send NOVAK/1 line. But if it sends a line
		// we don’t understand, also treat as /msg.
		if err := handleMsgConn(newPreReadConn(conn, br)); err != nil {
			log.Printf("[msg] default route error: %v", err)
		}
	}
}

func readRouteLine(br *bufio.Reader) (string, error) {
	// Look ahead: if first bytes don't match "NOVAK/1", do not consume anything.
	peek, err := br.Peek(min(len(routerMagic), routerMaxHdr))
	if err != nil && err != io.EOF {
		return "", err
	}
	if len(peek) < len(routerMagic) || string(peek[:len(routerMagic)]) != routerMagic {
		return "", fmt.Errorf("no route magic")
	}

	// Consume a full line (bounded).
	line, err := br.ReadString('\n')
	if err != nil {
		return "", err
	}
	if len(line) > routerMaxHdr {
		return "", fmt.Errorf("route line too long")
	}
	return line, nil
}

func parseRoute(line string) string {
	line = strings.TrimSpace(line)
	// Expected: "NOVAK/1 /pair"
	if !strings.HasPrefix(line, routerMagic) {
		return ""
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, routerMagic))
	if rest == "" {
		return ""
	}
	// allow extra tokens, e.g. "NOVAK/1 /pair foo=bar"
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// preReadConn lets handlers keep reading from the same buffered reader
// (which may already contain bytes after the route line).
type preReadConn struct {
	net.Conn
	br *bufio.Reader
}

func newPreReadConn(c net.Conn, br *bufio.Reader) net.Conn {
	return &preReadConn{Conn: c, br: br}
}

func (p *preReadConn) Read(b []byte) (int, error) {
	return p.br.Read(b)
}

func (p *preReadConn) Write(b []byte) (int, error) {
	return p.Conn.Write(b)
}

// Optional helper: allows handler to "unread" bytes by re-wrapping a reader.
// Not currently used, but handy if you want to preserve exact framing.
func unreadBytes(br *bufio.Reader, data []byte) *bufio.Reader {
	out := bufio.NewReader(io.MultiReader(bytes.NewReader(data), br))
	return out
}

// You MUST provide this in your codebase by adapting your existing 60768 handler.
// func handleMsgConn(conn net.Conn) error { ... }
