//go:build darwin
// +build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

const char* frontmostBundleID() {
	NSRunningApplication *app =
		[[NSWorkspace sharedWorkspace] frontmostApplication];
	if (!app || !app.bundleIdentifier) {
		return NULL;
	}
	return strdup([app.bundleIdentifier UTF8String]);
}

const char* frontmostExecutable() {
	NSRunningApplication *app =
		[[NSWorkspace sharedWorkspace] frontmostApplication];
	if (!app || !app.executableURL) {
		return NULL;
	}
	return strdup([[app.executableURL lastPathComponent] UTF8String]);
}
*/
import "C"

import (
	"errors"
	"strings"
	"unsafe"
)

func foregroundAppAllowed() (bool, string, error) {
	bundlePtr := C.frontmostBundleID()
	if bundlePtr == nil {
		return false, "", errors.New("no foreground bundle ID")
	}
	defer C.free(unsafe.Pointer(bundlePtr))

	exePtr := C.frontmostExecutable()
	if exePtr == nil {
		return false, "", errors.New("no foreground executable")
	}
	defer C.free(unsafe.Pointer(exePtr))

	bundleID := strings.ToLower(C.GoString(bundlePtr))
	exe := strings.ToLower(C.GoString(exePtr))

	// Match against allowlist
	for _, allowed := range settings.Allowlist.Darwin.BundleIDs {
		if bundleID == strings.ToLower(allowed) {
			return true, bundleID, nil
		}
	}

	for _, allowed := range settings.Allowlist.Darwin.Executables {
		if exe == strings.ToLower(allowed) {
			return true, exe, nil
		}
	}

	return false, exe, nil
}
