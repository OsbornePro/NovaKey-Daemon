//go:build windows

package main

import (
	"unsafe"
)

var registerHotKey = user32.NewProc("RegisterHotKey")
var unregisterHotKey = user32.NewProc("UnregisterHotKey")
var getMessage = user32.NewProc("GetMessageW")

const (
	MOD_ALT   = 0x0001
	MOD_CTRL  = 0x0002
	MOD_SHIFT = 0x0004
	MOD_WIN   = 0x0008

	WM_HOTKEY = 0x0312
)

type msg struct {
	hwnd   uintptr
	msg    uint32
	wParam uintptr
	lParam uintptr
	time   uint32
	pt     struct{ x, y int32 }
}

func startHotkeyListener() {
	mod := uint(0)
	if settings.Arming.Hotkey.Ctrl {
		mod |= MOD_CTRL
	}
	if settings.Arming.Hotkey.Alt {
		mod |= MOD_ALT
	}
	if settings.Arming.Hotkey.Shift {
		mod |= MOD_SHIFT
	}
	if settings.Arming.Hotkey.Win {
		mod |= MOD_WIN
	}

	if len(settings.Arming.Hotkey.Key) != 1 {
		LogError("Hotkey must be a single character", nil)
		return
	}

	key := uintptr(settings.Arming.Hotkey.Key[0])

	ok, _, err := registerHotKey.Call(
		0,
		1,
		uintptr(mod),
		key,
	)
	if ok == 0 {
		LogError("Failed to register hotkey", err)
		return
	}
	defer unregisterHotKey.Call(0, 1)

	LogInfo("Hotkey listener started")

	var m msg
	for {
		ret, _, _ := getMessage.Call(
			uintptr(unsafe.Pointer(&m)),
			0,
			0,
			0,
		)
		if ret == 0 {
			return
		}

		if m.msg == WM_HOTKEY {
			LogInfo("Service armed via hotkey")
			arm()
		}
	}
}
