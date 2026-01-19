// cmd/novakey/inject_method.go
package main

type InjectMethod string

const (
	InjectMethodDirect   InjectMethod = "direct"
	InjectMethodTyping   InjectMethod = "typing"
	InjectMethodClipboard InjectMethod = "clipboard"
)

