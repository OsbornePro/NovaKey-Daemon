// cmd/novakey/focused_target_windows.go
//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func getFocusedTarget() (string, string, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return "", "", fmt.Errorf("GetForegroundWindow returned NULL")
	}

	// Window title
	title := ""
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if n > 0 {
		buf := make([]uint16, n+1)
		procGetWindowTextW.Call(
			hwnd,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(n+1),
		)
		title = windows.UTF16ToString(buf)
	}

	// PID -> process image name
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return "", title, fmt.Errorf("GetWindowThreadProcessId returned pid=0")
	}

	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", title, fmt.Errorf("OpenProcess: %w", err)
	}
	defer windows.CloseHandle(h)

	var size uint32 = 4096
	buf := make([]uint16, size)
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", title, fmt.Errorf("QueryFullProcessImageName: %w", err)
	}
	full := windows.UTF16ToString(buf[:size])

	// Return just the exe name
	exe := full
	for i := len(exe) - 1; i >= 0; i-- {
		if exe[i] == '\\' || exe[i] == '/' {
			exe = exe[i+1:]
			break
		}
	}
	if exe == "" {
		exe = full
	}
	return exe, title, nil
}
