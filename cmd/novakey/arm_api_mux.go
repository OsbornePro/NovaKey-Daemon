// cmd/novakey/arm_api_mux.go
package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// armMuxForTests returns the exact mux used by startArmAPI(), but without listening.
// Keep this small and deterministic so CI can test it with httptest.
func armMuxForTests() *http.ServeMux {
	mux := http.NewServeMux()

	// Require token for everything (including /status) so it’s not an oracle.
	mux.HandleFunc("/status", requireArmToken(handleArmStatus))
	mux.HandleFunc("/arm", requireArmToken(handleArmArm))
	mux.HandleFunc("/disarm", requireArmToken(handleArmDisarm))

	return mux
}

func handleArmStatus(w http.ResponseWriter, r *http.Request) {
	// Non-consuming check.
	if armGate.Consume(false) {
		_, _ = fmt.Fprintln(w, "armed=true")
		return
	}
	_, _ = fmt.Fprintln(w, "armed=false")
}

func handleArmArm(w http.ResponseWriter, r *http.Request) {
	// ms can override default duration
	ms := cfg.ArmDurationMs
	if q := r.URL.Query().Get("ms"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			ms = n
		}
	}

	armGate.ArmFor(time.Duration(ms) * time.Millisecond)

	// Keep response stable for scripts.
	_, _ = fmt.Fprintf(w, "armed_for_ms=%d\n", ms)
}

func handleArmDisarm(w http.ResponseWriter, r *http.Request) {
	armGate.Disarm()
	_, _ = fmt.Fprintln(w, "disarmed=true")
}

