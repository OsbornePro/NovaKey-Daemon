// cmd/novakey/logging.go
package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

var globalReqID uint64

func nextReqID() uint64 {
	return atomic.AddUint64(&globalReqID, 1)
}

func logReqf(id uint64, format string, args ...any) {
	prefix := "[" + formatReqID(id) + "] "
	log.Printf(prefix+format, args...)
}

// Optional: use for explicitly-auditable events (blocks/denies/arming/etc.)
func logSecurityf(id uint64, format string, args ...any) {
	prefix := "[" + formatReqID(id) + "] SECURITY: "
	log.Printf(prefix+format, args...)
}

func formatReqID(id uint64) string {
	return "req:" + itoa64(id)
}

// cheap uint64 -> string, to avoid pulling fmt in hot paths
func itoa64(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// safePreview returns a short, non-secret preview of the text for logs.
func safePreview(s string) string {
	const max = 3
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return `""`
	}
	if n > max {
		return `"` + string(runes[:max]) + `..." (len=` + itoa64(uint64(n)) + ")"
	}
	return `"` + string(runes) + `" (len=` + itoa64(uint64(n)) + ")"
}

// --------------------
// Global log redaction
// --------------------
//
// Call initLoggingFromConfig() once after loadConfig() in each OS main().
// This installs a writer that scrubs secrets from ALL log output.

var (
	logInitOnce sync.Once

	redactMu sync.RWMutex
	redactOn = true
	secrets  = map[string]struct{}{}
)

// initLoggingFromConfig installs a log writer that redacts sensitive values.
// It is safe to call multiple times (only first call takes effect).
func initLoggingFromConfig() {
	logInitOnce.Do(func() {
		// Default: redact on (can disable for local debugging)
		redactOn = envBoolDefault("NOVAKEY_LOG_REDACT", true)

		// Scrub output line-by-line so partial writes don’t bypass redaction.
		log.SetOutput(newLineSanitizingWriter(os.Stderr))

		// Keep your existing flags if you want; this adds microseconds for debugging.
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		// Seed known secrets (best-effort, no failures if absent)
		seedSecretsFromConfig()
	})
}

func seedSecretsFromConfig() {
	// Two-man magic (treat as sensitive-adjacent)
	magic := cfg.ApproveMagic
	if magic == "" {
		magic = "__NOVAKEY_APPROVE__"
	}
	addSecret(magic)

	// Arm token (real secret)
	if cfg.ArmTokenFile != "" {
		if b, err := os.ReadFile(cfg.ArmTokenFile); err == nil {
			t := strings.TrimSpace(string(b))
			if t != "" {
				addSecret(t)
			}
		}
	}

	// If you add other secrets later (device keys, etc.), add them here.
}

func addSecret(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	redactMu.Lock()
	secrets[s] = struct{}{}
	redactMu.Unlock()
}

type lineSanitizingWriter struct {
	dst io.Writer
	mu  sync.Mutex
	buf bytes.Buffer
}

func newLineSanitizingWriter(dst io.Writer) *lineSanitizingWriter {
	return &lineSanitizingWriter{dst: dst}
}

func (w *lineSanitizingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := len(p)
	_, _ = w.buf.Write(p)

	for {
		b := w.buf.Bytes()
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			break
		}

		line := string(b[:i+1]) // include newline
		w.buf.Next(i + 1)

		out := redactLine(line)
		if _, err := io.WriteString(w.dst, out); err != nil {
			return n, err
		}
	}

	return n, nil
}

func redactLine(line string) string {
	redactMu.RLock()
	on := redactOn
	// snapshot secrets so we don't hold lock while processing
	localSecrets := make([]string, 0, len(secrets))
	for k := range secrets {
		localSecrets = append(localSecrets, k)
	}
	redactMu.RUnlock()

	if !on {
		return line
	}

	out := line

	// 1) Exact known secrets
	for _, sec := range localSecrets {
		if sec == "" {
			continue
		}
		if strings.Contains(out, sec) {
			out = strings.ReplaceAll(out, sec, "[REDACTED]")
		}
	}

	// 2) Heuristic: redact huge base64-ish blobs (Kyber keys, etc.)
	out = redactLongBlobs(out)

	// 3) Defensive: scrub common key=value hints (future-proofing)
	out = redactKeyValueHints(out)

	return out
}

func redactLongBlobs(s string) string {
	// Replace long runs of base64-ish characters.
	// Intentionally heuristic: better to over-redact than leak.
	const minLen = 120
	const blobChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=_-"

	var b strings.Builder
	b.Grow(len(s))

	runStart := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.ContainsRune(blobChars, rune(c)) {
			if runStart == -1 {
				runStart = i
			}
			continue
		}

		if runStart != -1 {
			runLen := i - runStart
			if runLen >= minLen {
				b.WriteString("[REDACTED_BLOB]")
			} else {
				b.WriteString(s[runStart:i])
			}
			runStart = -1
		}
		b.WriteByte(c)
	}

	if runStart != -1 {
		runLen := len(s) - runStart
		if runLen >= minLen {
			b.WriteString("[REDACTED_BLOB]")
		} else {
			b.WriteString(s[runStart:])
		}
	}

	return b.String()
}

func redactKeyValueHints(s string) string {
	keys := []string{
		"password=", "pass=", "secret=", "token=", "key_hex=", "kyber=", "aead=",
	}
	out := s
	lo := strings.ToLower(out)

	for _, k := range keys {
		idx := strings.Index(lo, k)
		if idx < 0 {
			continue
		}
		start := idx + len(k)
		end := start
		for end < len(out) {
			ch := out[end]
			if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' || ch == ',' || ch == '"' {
				break
			}
			end++
		}
		if start < end {
			out = out[:start] + "[REDACTED]" + out[end:]
			lo = strings.ToLower(out)
		}
	}
	return out
}

func envBoolDefault(name string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

