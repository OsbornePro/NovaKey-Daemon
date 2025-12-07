//go:build windows
// +build windows

package main

import (
	"syscall"
	"time"
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

func TypeString(s string) {
	time.Sleep(600 * time.Millisecond)

	for _, r := range s {
		if r == '\n' || r == '\r' {
			keyDownUp(VK_RETURN)
			continue
		}

		vk, needShift := runeToVK(r)

		if needShift {
			procKeybdEvent.Call(uintptr(VK_SHIFT), 0, 0, 0)
		}
		keyDownUp(int(vk))
		if needShift {
			procKeybdEvent.Call(uintptr(VK_SHIFT), 0, KEYEVENTF_KEYUP, 0)
		}
		time.Sleep(12 * time.Millisecond)
	}

	// Final Enter
	keyDownUp(VK_RETURN)
}

func keyDownUp(vk int) {
	procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vk), 0, KEYEVENTF_KEYUP, 0)
}

func runeToVK(r rune) (vk int, shift bool) {
	if r >= 'a' && r <= 'z' {
		return int(r - 32), false // uppercase VK code
	}
	if r >= 'A' && r <= 'Z' {
		return int(r), true
	}
	if r >= '0' && r <= '9' {
		return int(r), false
	}

	m := map[rune]int{
		' ': 0x20, '!': '1', '@': '2', '#': '3', '$': '4', '%': '5',
		'^': '6', '&': '7', '*': '8', '(': '9', ')': '0',
		'-': 0xBD, '_': 0xBD, '=': 0xBB, '+': 0xBB,
		'[': 0xDB, '{': 0xDB, ']': 0xDD, '}': 0xDD,
		';': 0xBA, ':': 0xBA, '\'': 0xDE, '"': 0xDE,
		',': 0xBC, '<': 0xBC, '.': 0xBE, '>': 0xBE,
		'/': 0xBF, '?': 0xBF, '\\': 0xDC, '|': 0xDC,
		'`': 0xC0, '~': 0xC0,
	}
	if v, ok := m[r]; ok {
		return v, true
	}
	return 0x20, false // fallback space
}
