// cmd/novakey/dpapi_windows.go
//go:build windows

package main

import (
	"encoding/base64"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DPAPI wrapper format stored on disk.
type dpapiFile struct {
	V        int    `json:"v"`
	DPAPIB64 string `json:"dpapi_b64"`
}

func dpapiProtect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("dpapiProtect: empty plaintext")
	}

	in := windows.DataBlob{Size: uint32(len(plaintext)), Data: &plaintext[0]}
	var out windows.DataBlob

	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, fmt.Errorf("CryptProtectData: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))

	buf := unsafe.Slice(out.Data, out.Size)
	cp := make([]byte, len(buf))
	copy(cp, buf)
	return cp, nil
}

func dpapiUnprotect(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("dpapiUnprotect: empty ciphertext")
	}

	in := windows.DataBlob{Size: uint32(len(ciphertext)), Data: &ciphertext[0]}
	var out windows.DataBlob

	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, fmt.Errorf("CryptUnprotectData: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))

	buf := unsafe.Slice(out.Data, out.Size)
	cp := make([]byte, len(buf))
	copy(cp, buf)
	return cp, nil
}

func dpapiEncode(b []byte) string            { return base64.StdEncoding.EncodeToString(b) }
func dpapiDecode(s string) ([]byte, error)   { return base64.StdEncoding.DecodeString(s) }
