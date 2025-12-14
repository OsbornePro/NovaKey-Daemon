// cmd/novakey/arm_api_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestArmAPI_UnauthorizedRejected(t *testing.T) {
	// minimal cfg defaults needed by mux
	cfg.ArmTokenHeader = "X-NovaKey-Token"
	cfg.ArmDurationMs = 20000

	token := "goodtoken"
	mux := buildArmMux(token)

	req := httptest.NewRequest(http.MethodPost, "http://example/arm?ms=1000", nil)
	req.Header.Set(cfg.ArmTokenHeader, "badtoken")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%q", rr.Code, rr.Body.String())
	}
}

func TestArmAPI_ArmSetsGateAndStatus(t *testing.T) {
	cfg.ArmTokenHeader = "X-NovaKey-Token"
	cfg.ArmDurationMs = 20000

	token := "tok"
	mux := buildArmMux(token)

	// Arm for 1500ms
	req := httptest.NewRequest(http.MethodPost, "http://example/arm?ms=1500", nil)
	req.Header.Set(cfg.ArmTokenHeader, token)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "armed for") {
		t.Fatalf("expected 'armed for' in response, got %q", rr.Body.String())
	}

	// Status should show an armed_until in the future (best-effort check)
	req2 := httptest.NewRequest(http.MethodGet, "http://example/status", nil)
	req2.Header.Set(cfg.ArmTokenHeader, token)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rr2.Code, rr2.Body.String())
	}
	body := rr2.Body.String()
	if !strings.Contains(body, "armed_until=") {
		t.Fatalf("expected armed_until, got %q", body)
	}

	// Very lightweight sanity: ArmedUntil should be >= now (allow tiny clock/format variance)
	until := armGate.ArmedUntil()
	if until.Before(time.Now().Add(-50 * time.Millisecond)) {
		t.Fatalf("expected armed_until in the future-ish, got %s", until)
	}
}

func TestArmGate_ConsumeTrueConsumes(t *testing.T) {
	// Use a fresh local gate instance (no globals).
	var g ArmGate

	g.Arm(2 * time.Second)
	if ok := g.Consume(true); !ok {
		t.Fatalf("expected first consume(true) to succeed")
	}
	if ok := g.Consume(true); ok {
		t.Fatalf("expected second consume(true) to fail (should be disarmed)")
	}
}

func TestArmGate_BlockedWhenNotArmed(t *testing.T) {
	var g ArmGate
	// Not armed
	if ok := g.Consume(true); ok {
		t.Fatalf("expected consume(true) to fail when not armed")
	}
}

