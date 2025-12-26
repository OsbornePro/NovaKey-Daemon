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

func (g *twoManGate) ClearForTests() {
	*g = *newTwoManGate()
}

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
	return g.until[deviceID]
}

func (g *twoManGate) cleanupLocked(now time.Time) {
	for id, u := range g.until {
		if now.After(u) {
			delete(g.until, id)
		}
	}
}

// Global gate instance
var approvalGate = newTwoManGate()

func approveWindow() time.Duration {
	ms := cfg.ApproveWindowMs
	if ms <= 0 {
		ms = 15000
	}
	return time.Duration(ms) * time.Millisecond
}
