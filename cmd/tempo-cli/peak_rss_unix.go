//go:build unix

package main

import (
	"runtime"
	"syscall"
)

// peakRSSBytes returns the peak resident memory of the process, or 0 if it
// cannot be read.
func peakRSSBytes() uint64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil || ru.Maxrss < 0 {
		return 0
	}
	// Darwin reports bytes, the other Unix systems kilobytes.
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return uint64(ru.Maxrss)
	}
	return uint64(ru.Maxrss) * 1024
}
