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
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	psapi    = syscall.NewLazyDLL("psapi.dll")

	getForegroundWindow      = user32.NewProc("GetForegroundWindow")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	openProcess              = kernel32.NewProc("OpenProcess")
	closeHandle              = kernel32.NewProc("CloseHandle")
	getModuleFileNameExW     = psapi.NewProc("GetModuleFileNameExW")
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
		return "", errors.New("failed to resolve foreground PID")
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
	return strings.ToLower(filepath.Base(fullPath)), nil
}

func foregroundAppAllowed() (bool, string, error) {
	exe, err := getForegroundExe()
	if err != nil {
		return false, "", err
	}

	// --- HARD DENY (testing / safety) ---
	if exe == "cmd.exe" {
		return false, exe, nil
	}

	// --- FOR TESTING: ALWAYS ALLOW POWERSHELL ---
	if exe == "powershell.exe" || exe == "pwsh.exe" {
		return true, exe, nil
	}

	// --- Normal allowlist enforcement ---
	for _, allowed := range settings.Allowlist.Windows.Browsers {
		if exe == strings.ToLower(allowed) {
			return true, exe, nil
		}
	}

	for _, allowed := range settings.Allowlist.Windows.PasswordManagers {
		if exe == strings.ToLower(allowed) {
			return true, exe, nil
		}
	}

	return false, exe, nil
}
