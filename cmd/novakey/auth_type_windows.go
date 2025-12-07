//go:build windows
// +build windows

package main

import (
	"syscall"
	"time"
	"unicode/utf8"
)

var (
	moduser32      = syscall.NewLazyDLL("user32.dll")
	procKeybdEvent = moduser32.NewProc("keybd_event")
)

const (
	KEYEVENTF_KEYUP = 0x0002
	VK_RETURN       = 0x0D
	VK_SHIFT        = 0x10
)

// SecureType sends the given UTF-8 bytes as synthetic key presses.
func SecureType(b []byte) {
	time.Sleep(600 * time.Millisecond)

	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			// Skip invalid byte and continue.
			b = b[1:]
			continue
		}
		b = b[size:]

		if r == '\n' || r == '\r' {
			keyDownUp(VK_RETURN)
			continue
		}

		vk, needShift := runeToVK(r)

		if needShift {
			procKeybdEvent.Call(uintptr(VK_SHIFT), 0, 0, 0)
		}
		keyDownUp(vk)
		if needShift {
			procKeybdEvent.Call(uintptr(VK_SHIFT), 0, KEYEVENTF_KEYUP, 0)
		}

		time.Sleep(12 * time.Millisecond)
	}

	keyDownUp(VK_RETURN)
}

func keyDownUp(vk int) {
	procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vk), 0, KEYEVENTF_KEYUP, 0)
}

func runeToVK(r rune) (vk int, shift bool) {
	if r >= 'a' && r <= 'z' {
		return int(r - 32), false
	}
	if r >= 'A' && r <= 'Z' {
		return int(r), true
	}
	if r >= '0' && r <= '9' {
		return int(r), false
	}

	m := map[rune]struct {
		vk    int
		shift bool
	}{
		' ': {0x20, false},

		'!': {'1', true}, '@': {'2', true}, '#': {'3', true}, '$': {'4', true},
		'%': {'5', true}, '^': {'6', true}, '&': {'7', true}, '*': {'8', true},
		'(': {'9', true}, ')': {'0', true},

		'-': {0xBD, false}, '_': {0xBD, true},
		'=': {0xBB, false}, '+': {0xBB, true},

		'[': {0xDB, false}, '{': {0xDB, true},
		']': {0xDD, false}, '}': {0xDD, true},

		';': {0xBA, false}, ':': {0xBA, true},

		'\'': {0xDE, false}, '"': {0xDE, true},

		',': {0xBC, false}, '<': {0xBC, true},
		'.': {0xBE, false}, '>': {0xBE, true},
		'/': {0xBF, false}, '?': {0xBF, true},

		'\\': {0xDC, false}, '|': {0xDC, true},

		'`': {0xC0, false}, '~': {0xC0, true},
	}

	if v, ok := m[r]; ok {
		return v.vk, v.shift
	}

	// Fallback to space
	return 0x20, false
}
