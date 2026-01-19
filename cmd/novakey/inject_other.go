// cmd/novakey/inject_other.go
//go:build !windows && !darwin && !linux

package main

import "fmt"

func InjectPasswordToFocusedControl(password string) (InjectMethod, error) {
	return "", fmt.Errorf("injection not supported on this OS")
}

