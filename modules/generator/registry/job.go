package registry

import (
	"context"
	"time"
)

// job executes f every getInterval.
func job(ctx context.Context, f func(context.Context), getInterval func() time.Duration) {
	currentInterval := getInterval()
	ticker := time.NewTicker(currentInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			jobCtx, cancel := context.WithTimeout(ctx, currentInterval)
			f(jobCtx)
			cancel()
		}

		newInterval := getInterval()
		if currentInterval != newInterval {
			ticker = time.NewTicker(newInterval)
			currentInterval = newInterval
		}
	}
}

func constantInterval(interval time.Duration) func() time.Duration {
	return func() time.Duration {
		return interval
	}
}
