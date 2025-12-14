// cmd/novakey/two_man.go
package main

import (
	"sync"
	"time"
)

// twoManGate tracks per-device approval windows.
type twoManGate struct {
	mu    sync.Mutex
	until map[string]time.Time
}

func newTwoManGate() *twoManGate {
	return &twoManGate{
		until: make(map[string]time.Time),
	}
}

func (g *twoManGate) Approve(deviceID string, d time.Duration) time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cleanupLocked(time.Now())
	u := time.Now().Add(d)
	g.until[deviceID] = u
	return u
}

// Consume returns true if device is currently approved.
// If consume==true, it clears approval after use.
func (g *twoManGate) Consume(deviceID string, consume bool) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	g.cleanupLocked(now)

	u, ok := g.until[deviceID]
	if !ok || now.After(u) {
		return false
	}
	if consume {
		delete(g.until, deviceID)
	}
	return true
}

func (g *twoManGate) ApprovedUntil(deviceID string) time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	u := g.until[deviceID]
	return u
}

func (g *twoManGate) cleanupLocked(now time.Time) {
	for id, u := range g.until {
		if now.After(u) {
			delete(g.until, id)
		}
	}
}

// Global gate instance (used by all platforms)
var approvalGate = newTwoManGate()

// isApproveControlPayload returns true if this decrypted “password” is actually an approval control message.
func isApproveControlPayload(pw string) bool {
	// Default magic string if not configured.
	magic := cfg.ApproveMagic
	if magic == "" {
		magic = "__NOVAKEY_APPROVE__"
	}
	return pw == magic
}

// approveWindow returns the approval TTL duration.
func approveWindow() time.Duration {
	ms := cfg.ApproveWindowMs
	if ms <= 0 {
		ms = 15000
	}
	return time.Duration(ms) * time.Millisecond
}
