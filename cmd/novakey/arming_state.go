package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

const (
	// DefaultArmingDuration controls how long the service remains armed
	// after a successful ARM command. Adjust as desired.
	DefaultArmingDuration = 15 * time.Second
)

// armedUntil stores a Unix timestamp (seconds) until which the service
// is considered "armed". 0 means "not armed".
var armedUntil atomic.Int64

func isArmed() bool {
	until := armedUntil.Load()
	if until == 0 {
		return false
	}
	now := time.Now().Unix()
	return now <= until
}

// armOnce arms the service for the default duration.
func armOnce() {
	armFor(DefaultArmingDuration)
}

// armFor arms the service for the given duration.
func armFor(d time.Duration) {
	expiry := time.Now().Add(d).Unix()
	armedUntil.Store(expiry)
	LogInfo(fmt.Sprintf("NovaKey armed for %s: next valid payload will be typed", d))
}

func disarm() {
	armedUntil.Store(0)
	LogInfo("NovaKey disarmed")
}
