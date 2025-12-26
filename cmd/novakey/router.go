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
//   "NOVAK/1 /msg\n"   -> handleMsgConn
//   (anything else)    -> fallback to /msg (backwards compatible)
//
// Drop-in usage:
//   - call startUnifiedListener() from main after initCrypto()
//   - implement handleMsgConn(conn) to wrap your existing message flow.
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
	// Router does NOT close the connection.
	// Ownership belongs to the selected handler, which must Close().
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	br := bufio.NewReaderSize(conn, 4096)

	line, err := readRouteLine(br)
	if err != nil {
		// No route line: treat as raw /msg client (backwards compatible).
		_ = conn.SetReadDeadline(time.Time{})
		if err := handleMsgConn(newPreReadConn(conn, br)); err != nil {
			log.Printf("[net] /msg fallback error: %v", err)
		}
		return
	}

	_ = conn.SetReadDeadline(time.Time{})

	switch parseRoute(line) {
	case "/pair":
		if err := handlePairConn(newPreReadConn(conn, br)); err != nil {
			log.Printf("[pair] conn error: %v", err)
		}
	case "/msg":
		if err := handleMsgConn(newPreReadConn(conn, br)); err != nil {
			log.Printf("[msg] conn error: %v", err)
		}
	default:
		// Unknown route token → treat as /msg.
		if err := handleMsgConn(newPreReadConn(conn, br)); err != nil {
			log.Printf("[msg] default route error: %v", err)
		}
	}
}

func readRouteLine(br *bufio.Reader) (string, error) {
	peek, err := br.Peek(min(len(routerMagic), routerMaxHdr))
	if err != nil && err != io.EOF {
		return "", err
	}
	if len(peek) < len(routerMagic) || string(peek[:len(routerMagic)]) != routerMagic {
		return "", fmt.Errorf("no route magic")
	}

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
	if !strings.HasPrefix(line, routerMagic) {
		return ""
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, routerMagic))
	if rest == "" {
		return ""
	}
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

type preReadConn struct {
	net.Conn
	br *bufio.Reader
}

func newPreReadConn(c net.Conn, br *bufio.Reader) net.Conn {
	return &preReadConn{Conn: c, br: br}
}

func (p *preReadConn) Read(b []byte) (int, error)  { return p.br.Read(b) }
func (p *preReadConn) Write(b []byte) (int, error) { return p.Conn.Write(b) }

// Optional helper: wrap already-peeked bytes back onto the reader.
func unreadBytes(br *bufio.Reader, data []byte) *bufio.Reader {
	return bufio.NewReader(io.MultiReader(bytes.NewReader(data), br))
}
