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

var (
	logInitOnce sync.Once

	redactMu sync.RWMutex
	secrets  = map[string]struct{}{}

	secretReplacer atomic.Value // stores *strings.Replacer
)

func initLoggingFromConfig() {
	logInitOnce.Do(func() {
		outs := selectLogOutputs()

		var dst io.Writer
		switch {
		case outs.writer != nil && outs.toStderr:
			dst = io.MultiWriter(os.Stderr, outs.writer)
		case outs.writer != nil:
			dst = outs.writer
		default:
			dst = os.Stderr
		}

		log.SetOutput(newLineSanitizingWriter(dst))
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

//		seedSecretsFromConfig()
		rebuildSecretReplacerLocked()
	})
}

func loggingRedactEnabled() bool {
	if cfg.LogRedact == nil {
		return true
	}
	return *cfg.LogRedact
}

func addSecret(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}

	redactMu.Lock()
	if _, ok := secrets[s]; ok {
		redactMu.Unlock()
		return
	}
	secrets[s] = struct{}{}
	rebuildSecretReplacerLocked()
	redactMu.Unlock()
}

func rebuildSecretReplacerLocked() {
	if len(secrets) == 0 {
		secretReplacer.Store(strings.NewReplacer())
		return
	}
	pairs := make([]string, 0, len(secrets)*2)
	for sec := range secrets {
		if sec == "" {
			continue
		}
		pairs = append(pairs, sec, "[REDACTED]")
	}
	secretReplacer.Store(strings.NewReplacer(pairs...))
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
		line := string(b[:i+1])
		w.buf.Next(i + 1)

		out := redactLine(line)
		if _, err := io.WriteString(w.dst, out); err != nil {
			return n, err
		}
	}
	return n, nil
}

func redactLine(line string) string {
	if !loggingRedactEnabled() {
		return line
	}

	out := line

	if v := secretReplacer.Load(); v != nil {
		if r, ok := v.(*strings.Replacer); ok {
			out = r.Replace(out)
		}
	}

	out = redactLongBlobs(out)
	out = redactKeyValueHints(out)
	return out
}

func redactLongBlobs(s string) string {
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

// Redact obvious key/value hints (all occurrences, now URL-safe).
func redactKeyValueHints(s string) string {
	keys := []string{
		"password=", "pass=", "secret=", "token=", "key_hex=", "kyber=", "aead=",
	}
	out := s

	for _, k := range keys {
		for {
			lo := strings.ToLower(out)
			idx := strings.Index(lo, k)
			if idx < 0 {
				break
			}
			start := idx + len(k)
			end := start
			for end < len(out) {
				ch := out[end]
				// Expanded delimiters to catch URL query params:
				// token=AAA&fp=...  token=AAA#...  token=AAA?...
				if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' ||
					ch == ',' || ch == '"' || ch == '\'' ||
					ch == '&' || ch == '?' || ch == '#' || ch == ';' ||
					ch == ')' || ch == ']' || ch == '}' {
					break
				}
				end++
			}
			if start < end {
				out = out[:start] + "[REDACTED]" + out[end:]
			} else {
				break
			}
		}
	}
	return out
}
