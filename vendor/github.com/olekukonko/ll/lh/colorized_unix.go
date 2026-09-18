//go:build !windows

package lh

// No-op for Unix systems - ANSI is native.
func enableWindowsANSI() {}

func (h *ColorizedHandler) isWindowsTerminal() bool {
	return false
}
