//go:build windows
// +build windows

package main

import (
	"errors"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32                   = syscall.NewLazyDLL("user32.dll")
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	psapi                    = syscall.NewLazyDLL("psapi.dll")

	getForegroundWindow       = user32.NewProc("GetForegroundWindow")
	getWindowThreadProcessId  = user32.NewProc("GetWindowThreadProcessId")
	openProcess               = kernel32.NewProc("OpenProcess")
	closeHandle               = kernel32.NewProc("CloseHandle")
	getModuleFileNameExW      = psapi.NewProc("GetModuleFileNameExW")
)

const (
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
)

func getForegroundExe() (string, error) {
	hwnd, _, _ := getForegroundWindow.Call()
	if hwnd == 0 {
		return "", errors.New("no foreground window")
	}

	var pid uint32
	getWindowThreadProcessId.Call(
		hwnd,
		uintptr(unsafe.Pointer(&pid)),
	)
	if pid == 0 {
		return "", errors.New("unable to resolve foreground PID")
	}

	hProc, _, err := openProcess.Call(
		uintptr(PROCESS_QUERY_LIMITED_INFORMATION),
		0,
		uintptr(pid),
	)
	if hProc == 0 {
		return "", err
	}
	defer closeHandle.Call(hProc)

	buf := make([]uint16, syscall.MAX_PATH)
	ret, _, err := getModuleFileNameExW.Call(
		hProc,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return "", err
	}

	fullPath := syscall.UTF16ToString(buf)
	exe := strings.ToLower(filepath.Base(fullPath))
	return exe, nil
}
