// cmd/novakey/arm_api_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestArmAPI_UnauthorizedRejected(t *testing.T) {
	tmp := t.TempDir()
	cfg.ArmTokenFile = tmp + "/arm_token.txt"
	cfg.ArmTokenHeader = "X-NovaKey-Token"
	cfg.ArmDurationMs = 20000

	// Ensure token exists.
	if err := initArmTokenFile(cfg.ArmTokenFile); err != nil {
		t.Fatalf("initArmTokenFile: %v", err)
	}

	armGate.Disarm()

	mux := armMuxForTests()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// No token header => 401
	req, _ := http.NewRequest("GET", srv.URL+"/status", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestArmAPI_ArmSetsGate_ConsumeConsumes(t *testing.T) {
	tmp := t.TempDir()
	cfg.ArmTokenFile = tmp + "/arm_token.txt"
	cfg.ArmTokenHeader = "X-NovaKey-Token"
	cfg.ArmDurationMs = 20000

	if err := initArmTokenFile(cfg.ArmTokenFile); err != nil {
		t.Fatalf("initArmTokenFile: %v", err)
	}
	tokenBytes, err := os.ReadFile(cfg.ArmTokenFile)
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	token := string(tokenBytes)
	token = trimSpace(token)

	armGate.Disarm()

	mux := armMuxForTests()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Arm with correct token
	req, _ := http.NewRequest("POST", srv.URL+"/arm?ms=250", nil)
	req.Header.Set(cfg.ArmTokenHeader, token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("arm request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	// Non-consuming check should succeed
	if ok := armGate.Consume(false); !ok {
		t.Fatalf("expected armed gate to be open")
	}

	// Consuming check should succeed once, then fail
	if ok := armGate.Consume(true); !ok {
		t.Fatalf("expected Consume(true) to succeed first time")
	}
	if ok := armGate.Consume(false); ok {
		t.Fatalf("expected gate to be consumed/closed after Consume(true)")
	}

	// Also sanity-check a short arm expires quickly
	armGate.ArmFor(50 * time.Millisecond)
	time.Sleep(80 * time.Millisecond)
	if ok := armGate.Consume(false); ok {
		t.Fatalf("expected arm to expire")
	}
}

func TestApprovalGate_Basic(t *testing.T) {
	approvalGate.ClearForTests()

	until := approvalGate.Approve("dev1", 100*time.Millisecond)
	if until.IsZero() {
		t.Fatalf("expected non-zero until")
	}
	if ok := approvalGate.Consume("dev1", false); !ok {
		t.Fatalf("expected approval present")
	}
	if ok := approvalGate.Consume("dev1", true); !ok {
		t.Fatalf("expected consume ok")
	}
	if ok := approvalGate.Consume("dev1", false); ok {
		t.Fatalf("expected approval gone after consume")
	}
}

// small helper to avoid importing strings just for tests
func trimSpace(s string) string {
	// remove CRLF/whitespace around token
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\n' || s[i] == '\r' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\n' || s[j-1] == '\r' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

