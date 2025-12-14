package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func buildArmAPIMux(token string) *http.ServeMux {
	mux := http.NewServeMux()

	// POST /arm?ms=20000
	mux.HandleFunc("/arm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get(cfg.ArmTokenHeader) != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ms := cfg.ArmDurationMs
		if q := r.URL.Query().Get("ms"); q != "" {
			if v, err := strconv.Atoi(q); err == nil && v > 0 && v <= 120000 {
				ms = v
			}
		}

		armGate.Arm(time.Duration(ms) * time.Millisecond)
		cfg.ArmEnabled = true // arming implies gate is active (even if normally off for testing)

		fmt.Fprintf(w, "armed_for_ms=%d\n", ms)
	})

	// GET /status
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get(cfg.ArmTokenHeader) != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		until := armGate.ArmedUntil()
		now := time.Now()

		armed := false
		if !until.IsZero() && now.Before(until) {
			armed = true
		}

		// Keep it dead-simple and greppable.
		fmt.Fprintf(w, "armed=%t\narmed_until=%s\nnow=%s\n",
			armed,
			until.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
		)
	})

	return mux
}

