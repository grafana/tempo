package registry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_job(t *testing.T) {
	interval := durationPtr(200 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	var jobTimes []time.Duration
	lastRun := time.Now()

	go job(
		ctx,
		func(_ context.Context) {
			diff := time.Since(lastRun)
			lastRun = time.Now()

			jobTimes = append(jobTimes, diff)
			fmt.Println(diff)

			*interval = *interval + (20 * time.Millisecond)
		},
		func() time.Duration {
			return *interval
		},
	)

	time.Sleep(1 * time.Second)

	cancel()

	require.Len(t, jobTimes, 4)
	require.InDelta(t, 200*time.Millisecond, jobTimes[0], float64(10*time.Millisecond))
	require.InDelta(t, 220*time.Millisecond, jobTimes[1], float64(10*time.Millisecond))
	require.InDelta(t, 240*time.Millisecond, jobTimes[2], float64(10*time.Millisecond))
	require.InDelta(t, 260*time.Millisecond, jobTimes[3], float64(10*time.Millisecond))
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}
