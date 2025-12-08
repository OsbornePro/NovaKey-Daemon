//go:build windows
// +build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	MOD_ALT     = 0x0001
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004
	VK_N        = 0x4E
)

func registerHotkey() error {
	var mod uint = 0

	if settings.Arming.Hotkey.Ctrl {
		mod |= MOD_CONTROL
	}
	if settings.Arming.Hotkey.Alt {
		mod |= MOD_ALT
	}
	if settings.Arming.Hotkey.Shift {
		mod |= MOD_SHIFT
	}

	user32 := syscall.NewLazyDLL("user32.dll")
	registerHotKey := user32.NewProc("RegisterHotKey")

	_, _, err := registerHotKey.Call(
		0,
		1,
		uintptr(mod),
		uintptr(VK_N),
	)
	return err
}

func hotkeyLoop() {
	user32 := syscall.NewLazyDLL("user32.dll")
	getMessage := user32.NewProc("GetMessageW")

	var msg struct {
		hwnd   uintptr
		msg    uint32
		wparam uintptr
		lparam uintptr
		time   uint32
		ptx    int32
		pty    int32
	}

	for {
		getMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)

		if msg.msg == 0x0312 { // WM_HOTKEY
			armOnce()
		}
	}
}
