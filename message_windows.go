//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	mbOK              = 0x00000000
	mbYesNo           = 0x00000004
	mbIconError       = 0x00000010
	mbIconInformation = 0x00000040
	mbIconQuestion    = 0x00000020
	mbSetForeground   = 0x00010000
	idYes             = 6
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

func messageBox(title, text string, flags uintptr) int {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags|mbSetForeground)
	return int(r)
}

func showInfo(title, text string) {
	messageBox(title, text, mbOK|mbIconInformation)
}

func showError(title, text string) {
	messageBox(title, text, mbOK|mbIconError)
}

func askYesNo(title, text string) bool {
	return messageBox(title, text, mbYesNo|mbIconQuestion) == idYes
}
