// cmd/novakey/arm_gate.go
package main

import (
	"sync"
	"time"
)

// ArmGate tracks whether the daemon is locally "armed" until a deadline.
type ArmGate struct {
	mu         sync.Mutex
	armedUntil time.Time
}

// Global gate instance (used by all platforms)
var armGate ArmGate

// ArmFor arms the gate until now+dur and returns the expiry time.
func (g *ArmGate) ArmFor(dur time.Duration) time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.armedUntil = time.Now().Add(dur)
	return g.armedUntil
}

// Disarm immediately clears the armed state.
func (g *ArmGate) Disarm() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.armedUntil = time.Time{}
}

// ArmedUntil returns the current armed-until time (zero if disarmed).
func (g *ArmGate) ArmedUntil() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.armedUntil
}

// Consume returns true if armed and not expired.
// If consume==true, it disarms after allowing this call.
func (g *ArmGate) Consume(consume bool) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.armedUntil.IsZero() {
		return false
	}
	now := time.Now()
	if now.After(g.armedUntil) {
		// expired
		g.armedUntil = time.Time{}
		return false
	}

	if consume {
		g.armedUntil = time.Time{}
	}
	return true
}
