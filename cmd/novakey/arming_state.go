package main

import "sync/atomic"

var armedFlag atomic.Bool

func isArmed() bool {
	return armedFlag.Load()
}

func armOnce() {
	armedFlag.Store(true)
	LogInfo("NovaKey armed: next valid payload will be typed")
}

func disarm() {
	armedFlag.Store(false)
	LogInfo("NovaKey disarmed")
}
