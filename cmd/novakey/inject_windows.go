// inject_windows.go
//go:build windows

package main

import (
	"fmt"
	"log"
	"reflect"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procGetFocus                 = user32.NewProc("GetFocus")
	procSendMessageW             = user32.NewProc("SendMessageW")
    procGetClassNameW            = user32.NewProc("GetClassNameW")

	// clipboard-related
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")

	// global memory for clipboard
	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")

	// keyboard fallback
	procKeybdEvent     = user32.NewProc("keybd_event")
	procVkKeyScanW     = user32.NewProc("VkKeyScanW")
	procMapVirtualKeyW = user32.NewProc("MapVirtualKeyW")

	// NOTE: GetCurrentThreadId is in kernel32.dll
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
)

const (
	WM_SETTEXT       = 0x000C
	EM_REPLACESEL    = 0x00C2
	WM_GETTEXTLENGTH = 0x000E

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002

	VK_SHIFT        = 0x10
	KEYEVENTF_KEYUP = 0x0002
)

func getWindowClass(hwnd windows.Handle) (string, error) {
    var buf [256]uint16
    r1, _, err := procGetClassNameW.Call(
        uintptr(hwnd),
        uintptr(unsafe.Pointer(&buf[0])),
        uintptr(len(buf)),
    )
    if r1 == 0 {
        if err != syscall.Errno(0) {
            return "", fmt.Errorf("GetClassNameW: %v", err)
        }
        return "", fmt.Errorf("GetClassNameW returned 0")
    }
    return syscall.UTF16ToString(buf[:r1]), nil
}

// InjectPasswordToFocusedControl on Windows:
//
//   1) Copy password to clipboard (best-effort).
//   2) Get HWND of focused control.
//   3) Try EM_REPLACESEL / WM_SETTEXT.
//      - If text length changes (WM_GETTEXTLENGTH), treat as success.
//   4) If text length didn't change or anything failed, fall back to keybd_event typing.
func InjectPasswordToFocusedControl(password string) error {
    log.Printf("[windows] InjectPasswordToFocusedControl called; len=%d", len(password))

    // clipboard first (best-effort)
    if err := setClipboardText(password); err != nil {
        log.Printf("[windows] setClipboardText failed: %v", err)
    } else {
        log.Printf("[windows] password copied to clipboard")
    }

    hwnd, err := getFocusedControl()
    if err != nil {
        return fmt.Errorf("getFocusedControl: %w", err)
    }
    if hwnd == 0 {
        return fmt.Errorf("no focused control")
    }

    className, err := getWindowClass(hwnd)
    if err != nil {
        log.Printf("[windows] getWindowClass failed: %v", err)
        className = "<unknown>"
    }
    log.Printf("[windows] focused HWND=0x%X class=%q", uintptr(hwnd), className)

    // Only use direct messages on known-safe text controls
   	safeDirect := className == "Edit" ||
        className == "RichEdit20W" || className == "RichEdit20A"

    if safeDirect {
        beforeLen := getTextLength(hwnd)
        log.Printf("[windows] initial text length=%d", beforeLen)

        if err := injectViaMessages(hwnd, password); err == nil {
            afterLen := getTextLength(hwnd)
            log.Printf("[windows] post-message text length=%d", afterLen)

            if beforeLen >= 0 && afterLen >= 0 && afterLen != beforeLen {
                log.Printf("[windows] direct message injection succeeded (len %d -> %d)", beforeLen, afterLen)
                return nil
            }

            log.Printf("[windows] direct message injection uncertain/no change (len %d -> %d), falling back to keybd_event", beforeLen, afterLen)
        } else {
            log.Printf("[windows] direct message injection failed, falling back to keybd_event: %v", err)
        }
    } else {
        log.Printf("[windows] control class %q not in safe list; using keybd_event", className)
    }

    // Fallback typing path (works in ISE, browsers, etc.)
    if err := injectViaKeybdEvent(password); err != nil {
        log.Printf("[windows] keybd_event typing failed: %v", err)
        return fmt.Errorf("keybd_event typing failed: %w", err)
    }
    log.Printf("[windows] keybd_event typing fallback succeeded")
    return nil
}

// injectViaMessages uses EM_REPLACESEL then WM_SETTEXT on the control.
func injectViaMessages(hwnd windows.Handle, password string) error {
	log.Printf("[windows] injectViaMessages start")
	pwUTF16, err := utf16FromString(password)
	if err != nil {
		return fmt.Errorf("utf16FromString: %w", err)
	}
	ptr := uintptr(unsafe.Pointer(&pwUTF16[0]))

	// Try EM_REPLACESEL to insert at caret / replace selection
	r1, _, _ := procSendMessageW.Call(
		uintptr(hwnd),
		uintptr(EM_REPLACESEL),
		uintptr(1), // TRUE
		ptr,
	)
	if r1 != 0 {
		return nil
	}

	// Fallback: replace entire content with WM_SETTEXT
	r1, _, _ = procSendMessageW.Call(
		uintptr(hwnd),
		uintptr(WM_SETTEXT),
		0,
		ptr,
	)
	if r1 == 0 {
		return fmt.Errorf("WM_SETTEXT returned 0 (likely failed)")
	}
	return nil
}

// getTextLength uses WM_GETTEXTLENGTH to read current length.
// Returns -1 on failure.
func getTextLength(hwnd windows.Handle) int {
	r1, _, _ := procSendMessageW.Call(
		uintptr(hwnd),
		uintptr(WM_GETTEXTLENGTH),
		0,
		0,
	)
	// r1 is LRESULT (signed). We'll clamp to int.
	if r1 == 0xFFFFFFFF {
		return -1
	}
	return int(int32(r1))
}

// injectViaKeybdEvent sends real keystrokes for each character of the password.
// This is the last-resort fallback: keyloggers will see each character.
func injectViaKeybdEvent(password string) error {
	log.Printf("[windows] injectViaKeybdEvent start, len=%d", len(password))
	for _, r := range password {
		if r > 0x7f {
			// For now, only handle basic ASCII; expand as needed.
			return fmt.Errorf("keybd_event fallback does not support non-ASCII char %q", r)
		}

		vk, shiftState, err := charToVk(byte(r))
		if err != nil {
			return fmt.Errorf("charToVk(%q): %w", r, err)
		}

		shiftNeeded := (shiftState & 0x01) != 0

		if shiftNeeded {
			keyEvent(VK_SHIFT, true)
		}

		keyEvent(vk, true)
		keyEvent(vk, false)

		if shiftNeeded {
			keyEvent(VK_SHIFT, false)
		}
	}

	return nil
}

// charToVk maps an ASCII character to a virtual-key code and shift state using VkKeyScanW.
func charToVk(ch byte) (byte, byte, error) {
	r1, _, err := procVkKeyScanW.Call(uintptr(ch))
	if r1 == ^uintptr(0) { // -1
		return 0, 0, fmt.Errorf("VkKeyScanW returned -1 for %q: %v", ch, err)
	}
	v := uint16(r1)
	vk := byte(v & 0xFF)
	shiftState := byte((v >> 8) & 0xFF)
	return vk, shiftState, nil
}

// keyEvent sends a key press or release via keybd_event.
func keyEvent(vk byte, down bool) {
	var flags uint32
	if !down {
		flags |= KEYEVENTF_KEYUP
	}

	r1, _, _ := procMapVirtualKeyW.Call(uintptr(vk), 0)
	scan := byte(r1 & 0xFF)

	procKeybdEvent.Call(
		uintptr(vk),
		uintptr(scan),
		uintptr(flags),
		0,
	)
}

// setClipboardText sets the Windows clipboard to the given UTF-16 text (CF_UNICODETEXT).
// We intentionally do NOT restore previous contents; user asked to keep password there.
func setClipboardText(text string) error {
	u16, err := utf16FromString(text)
	if err != nil {
		return err
	}

	dataSize := uintptr(len(u16) * 2) // 2 bytes per uint16
	hMem, _, err := procGlobalAlloc.Call(
		uintptr(GMEM_MOVEABLE),
		dataSize,
	)
	if hMem == 0 {
		return fmt.Errorf("GlobalAlloc failed: %v", err)
	}

	ptr, _, err := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return fmt.Errorf("GlobalLock failed: %v", err)
	}
	defer procGlobalUnlock.Call(hMem)

	var hdr reflect.SliceHeader
	hdr.Data = ptr
	hdr.Len = int(dataSize)
	hdr.Cap = int(dataSize)
	dst := *(*[]byte)(unsafe.Pointer(&hdr))

	for i, v := range u16 {
		dst[2*i] = byte(v)
		dst[2*i+1] = byte(v >> 8)
	}

	r1, _, err := procOpenClipboard.Call(0)
	if r1 == 0 {
		return fmt.Errorf("OpenClipboard failed: %v", err)
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	r1, _, err = procSetClipboardData.Call(
		uintptr(CF_UNICODETEXT),
		hMem,
	)
	if r1 == 0 {
		return fmt.Errorf("SetClipboardData failed: %v", err)
	}

	// ownership of hMem is transferred to the system
	return nil
}

// getFocusedControl finds the HWND of the control that currently has keyboard focus
// in the foreground window, by briefly attaching our input thread to the foreground
// thread so GetFocus() returns its focused control.
func getFocusedControl() (windows.Handle, error) {
	log.Printf("[windows] getFocusedControl start")
	// HWND GetForegroundWindow(void)
	r1, _, err := procGetForegroundWindow.Call()
	if r1 == 0 {
		if err != syscall.Errno(0) {
			return 0, fmt.Errorf("GetForegroundWindow: %v", err)
		}
		return 0, fmt.Errorf("GetForegroundWindow returned 0")
	}
	fg := windows.Handle(r1)

	// DWORD GetWindowThreadProcessId(HWND hWnd, LPDWORD lpdwProcessId)
	var pid uint32
	r1, _, _ = procGetWindowThreadProcessId.Call(
		uintptr(fg),
		uintptr(unsafe.Pointer(&pid)),
	)
	if r1 == 0 {
		return 0, fmt.Errorf("GetWindowThreadProcessId returned 0")
	}
	fgThread := uint32(r1)

	// DWORD GetCurrentThreadId(void) (from kernel32.dll)
	r1, _, err = procGetCurrentThreadId.Call()
	thisThread := uint32(r1)

	// BOOL AttachThreadInput(DWORD idAttach, DWORD idAttachTo, BOOL fAttach)
	r1, _, err = procAttachThreadInput.Call(
		uintptr(thisThread),
		uintptr(fgThread),
		uintptr(1), // TRUE
	)
	if r1 == 0 {
		if err != syscall.Errno(0) {
			return 0, fmt.Errorf("AttachThreadInput(TRUE): %v", err)
		}
		return 0, fmt.Errorf("AttachThreadInput(TRUE) returned 0")
	}
	defer procAttachThreadInput.Call(
		uintptr(thisThread),
		uintptr(fgThread),
		uintptr(0), // FALSE
	)

	// HWND GetFocus(void)
	r1, _, err = procGetFocus.Call()
	if r1 == 0 {
		if err != syscall.Errno(0) {
			return 0, fmt.Errorf("GetFocus: %v", err)
		}
		return 0, fmt.Errorf("GetFocus returned 0")
	}

	return windows.Handle(r1), nil
}

func utf16FromString(s string) ([]uint16, error) {
	u := utf16.Encode([]rune(s + "\x00"))
	if len(u) == 0 {
		return nil, fmt.Errorf("empty UTF-16 string")
	}
	return u, nil
}

