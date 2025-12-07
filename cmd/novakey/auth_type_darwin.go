//go:build darwin
// +build darwin

package main

import (
	"time"
	"unsafe"
)

/*
#include <ApplicationServices/ApplicationServices.h>
static void postKey(uint32_t key, bool down) {
    CGEventRef e = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)key, down);
    CGEventPost(kCGSessionEventTap, e);
    CFRelease(e);
}
*/
import "C"

func TypeString(s string) {
	time.Sleep(600 * time.Millisecond)
	for _, r := range s {
		keyCode := runeToMacKeyCode(r)
		shift := needsShift(r)

		if shift {
			C.postKey(56, true) // Left Shift
		}
		C.postKey(C.uint32_t(keyCode), true)
		C.postKey(C.uint32_t(keyCode), false)
		if shift {
			C.postKey(56, false)
		}
		time.Sleep(15 * time.Millisecond)
	}
	C.postKey(36, true)  // Return
	C.postKey(36, false) // Release
	time.Sleep(100 * time.Millisecond)
}

func needsShift(r rune) bool {
	return (r >= 'A' && r <= 'Z') || "!@#$%^&*()_+{}|:\"<>?~".Contains(string(r))
}

func runeToMacKeyCode(r rune) uint32 {
	m := map[rune]uint32{
		'a': 0x00, 'b': 0x0B, 'c': 0x08, 'd': 0x02, 'e': 0x0E, 'f': 0x03,
		'g': 0x05, 'h': 0x04, 'i': 0x22, 'j': 0x26, 'k': 0x28, 'l': 0x25,
		'm': 0x2E, 'n': 0x2D, 'o': 0x1F, 'p': 0x23, 'q': 0x0C, 'r': 0x0F,
		's': 0x01, 't': 0x11, 'u': 0x20, 'v': 0x09, 'w': 0x0D, 'x': 0x07,
		'y': 0x10, 'z': 0x06,
		'0': 0x1D, '1': 0x12, '2': 0x13, '3': 0x14, '4': 0x15, '5': 0x17,
		'6': 0x16, '7': 0x1A, '8': 0x1C, '9': 0x19,
		' ': 0x31, '\n': 0x24, '\t': 0x30, '-': 0x1B, '=': 0x18, '[': 0x21, ']': 0x1E,
		';': 0x29, "'": 0x27, ',': 0x2B, '.': 0x2F, '/': 0x2C, '\\': 0x2A,
	}
	if code, ok := m[r]; ok {
		return code
	}
	if r >= 'A' && r <= 'Z' {
		return m[r-'A'+'a']
	}
	return 0x31 // space as fallback
}
