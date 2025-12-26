// cmd/novakey/winapi_windows.go
//go:build windows

package main

import "syscall"

// Shared WinAPI DLL/proc handles used across Windows files.
// Define them ONCE to avoid redeclare errors and missing-proc errors.
var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	// Foreground/focus + PID
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procGetFocus                 = user32.NewProc("GetFocus")

	// Window title
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")

	// Control class + message injection
	procGetClassNameW      = user32.NewProc("GetClassNameW")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
)
