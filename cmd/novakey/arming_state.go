package main

import (
	"sync"
	"time"
)

var (
	armingMu   sync.Mutex
	armedUntil time.Time
)

// arm arms the service for the configured timeout window.
// If already armed, it extends the window.
func arm() {
	armingMu.Lock()
	defer armingMu.Unlock()

	timeout := time.Duration(settings.Arming.TimeoutSeconds) * time.Second
	if timeout <= 0 || timeout > 5*time.Minute {
		timeout = 30 * time.Second
	}

	armedUntil = time.Now().Add(timeout)
	LogInfo("Service armed for " + timeout.String())
}

// disarm immediately disarms the service.
func disarm() {
	armingMu.Lock()
	defer armingMu.Unlock()

	armedUntil = time.Time{}
	LogInfo("Service disarmed")
}

// isArmed returns true if the service is currently armed.
func isArmed() bool {
	armingMu.Lock()
	defer armingMu.Unlock()

	if armedUntil.IsZero() {
		return false
	}

	if time.Now().After(armedUntil) {
		armedUntil = time.Time{}
		LogInfo("Service auto-disarmed (timeout)")
		return false
	}

	return true
}

// ✅ NEW: centralized arming gate used immediately before typing
func acquireTypingPermission() bool {
	if !settings.Security.RequireArming {
		return true
	}

	return isArmed()
}
