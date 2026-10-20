//go:build unix

package metrics

import (
	"syscall"
	"time"
)

// CPUTime returns the process's user plus system time.
func CPUTime() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
