//go:build windows

package lh

import (
	"os"
	"syscall"
	"unsafe"
)

func init() {
	enableWindowsANSI()
}

// enableWindowsANSI enables virtual terminal processing on Windows 10+.
func enableWindowsANSI() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")

	const enableVirtualTerminalProcessing = 0x0004
	handles := []syscall.Handle{syscall.Stdout, syscall.Stderr}

	for _, handle := range handles {
		var mode uint32
		if r, _, _ := getConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode))); r != 0 {
			if mode&enableVirtualTerminalProcessing == 0 {
				newMode := mode | enableVirtualTerminalProcessing
				setConsoleMode.Call(uintptr(handle), uintptr(newMode))
			}
		}
	}
}

// isWindowsTerminal checks Windows-specific terminal indicators.
func (h *ColorizedHandler) isWindowsTerminal() bool {
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	if os.Getenv("ConEmuANSI") == "ON" {
		return true
	}
	if os.Getenv("ANSICON") != "" {
		return true
	}
	return false
}
