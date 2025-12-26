// cmd/novakey/inject_other.go
//go:build !linux && !darwin && !windows

package main

import "fmt"

func InjectPasswordToFocusedControl(password string) error {
	return fmt.Errorf("InjectPasswordToFocusedControl not implemented on this OS yet")
}
