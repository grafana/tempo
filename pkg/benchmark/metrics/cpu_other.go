//go:build !unix

package metrics

import "time"

func CPUTime() time.Duration { return 0 }
