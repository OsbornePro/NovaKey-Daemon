//go:build windows
// +build windows

package main

import (
	"runtime"
	"time"
	"unicode/utf8"
	"unsafe"
)

var sendInput = user32.NewProc("SendInput")

const (
	INPUT_KEYBOARD    = 1
	KEYEVENTF_KEYUP   = 0x0002
	KEYEVENTF_UNICODE = 0x0004
)

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// INPUT MUST be 40 bytes on amd64
type INPUT struct {
	Type uint32
	Ki   KEYBDINPUT
	_    [8]byte // REQUIRED padding
}

func SecureType(password []byte) {
	// 🔒 REQUIRED: must stay on one Windows thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	LogInfo("SecureType: injecting keystrokes")

	time.Sleep(500 * time.Millisecond)

	if len(password) == 0 {
		LogError("SecureType: empty password", nil)
		return
	}

	inputs := make([]INPUT, 0, len(password)*2+2)

	for len(password) > 0 {
		r, size := utf8.DecodeRune(password)
		if r == utf8.RuneError && size == 1 {
			password = password[1:]
			continue
		}
		password = password[size:]

		// Key down
		inputs = append(inputs, INPUT{
			Type: INPUT_KEYBOARD,
			Ki: KEYBDINPUT{
				WVk:     0,
				WScan:   uint16(r),
				DwFlags: KEYEVENTF_UNICODE,
			},
		})

		// Key up
		inputs = append(inputs, INPUT{
			Type: INPUT_KEYBOARD,
			Ki: KEYBDINPUT{
				WVk:     0,
				WScan:   uint16(r),
				DwFlags: KEYEVENTF_UNICODE | KEYEVENTF_KEYUP,
			},
		})
	}

	// Press Enter
	inputs = append(inputs,
		INPUT{
			Type: INPUT_KEYBOARD,
			Ki: KEYBDINPUT{
				WVk:     0,
				WScan:   '\r',
				DwFlags: KEYEVENTF_UNICODE,
			},
		},
		INPUT{
			Type: INPUT_KEYBOARD,
			Ki: KEYBDINPUT{
				WVk:     0,
				WScan:   '\r',
				DwFlags: KEYEVENTF_UNICODE | KEYEVENTF_KEYUP,
			},
		},
	)

	ret, _, err := sendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(unsafe.Sizeof(INPUT{})),
	)

	if ret == 0 {
		LogError("SendInput failed", err)
		return
	}

	LogInfo("SendInput succeeded (keystrokes injected)")
}
