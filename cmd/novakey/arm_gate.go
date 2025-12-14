package main

import (
	"sync"
	"time"
)

type ArmGate struct {
	mu     sync.Mutex
	until  time.Time
	lastAt time.Time
}

func (g *ArmGate) Arm(d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.until = time.Now().Add(d)
	g.lastAt = time.Now()
}

func (g *ArmGate) ArmedUntil() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.until
}

// Consume returns true if armed right now, and (optionally) disarms immediately.
func (g *ArmGate) Consume(consume bool) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if time.Now().Before(g.until) {
		if consume {
			g.until = time.Time{}
		}
		return true
	}
	return false
}

var armGate ArmGate

