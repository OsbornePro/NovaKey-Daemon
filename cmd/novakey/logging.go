// logging.go
package main

import (
	"log"
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

