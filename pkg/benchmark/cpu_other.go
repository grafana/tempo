//go:build !unix

package benchmark

import "time"

func cpuTime() time.Duration { return 0 }
