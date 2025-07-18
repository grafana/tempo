package work

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

// isRaceEnabled returns true if the race detector is enabled
func isRaceEnabled() bool {
	b, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}

	for _, s := range b.Settings {
		if s.Key == "-race" && s.Value == "true" {
			return true
		}
	}
	return false
}

// createTestWork creates a Work instance with many jobs to simulate production load
func createTestWork(tb testing.TB, numJobs int) *Work {
	cfg := Config{
		PruneAge:       time.Hour,
		DeadJobTimeout: time.Hour,
	}
	w := New(cfg)

	for i := 0; i < numJobs; i++ {
		job := &Job{
			ID:   fmt.Sprintf("job-%d", i),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: fmt.Sprintf("tenant-%d", i%100), // 100 different tenants
				Compaction: &tempopb.CompactionDetail{
					Input: []string{
						fmt.Sprintf("block-%d-1", i),
						fmt.Sprintf("block-%d-2", i),
						fmt.Sprintf("block-%d-3", i),
					},
				},
			},
		}
		err := w.AddJob(job)
		require.NoError(tb, err, "Failed to add job %d", i)
	}

	return w
}

// BenchmarkWorkContention tests the lock contention performance under realistic workload
func BenchmarkWorkContention(b *testing.B) {
	testCases := []struct {
		name    string
		numJobs int
	}{
		{"jobs_100", 100},
		{"jobs_1000", 1000},
		{"jobs_5000", 5000},
		{"jobs_10000", 10000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			w := createTestWork(b, tc.numJobs)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				workerID := fmt.Sprintf("worker-%d", time.Now().UnixNano())

				for pb.Next() {
					// Simulate realistic backend scheduler workload
					// Mix of read and write operations that happen in production

					// 1. Check for existing job (read-heavy)
					w.GetJobForWorker(ctx, workerID)

					// 2. Get job count (read-heavy)
					w.Len()

					// 3. List jobs (read-heavy)
					jobs := w.ListJobs()

					// 4. Occasionally get a specific job (read-heavy)
					if len(jobs) > 0 {
						w.GetJob(jobs[0].ID)
					}

					// 5. Simulate marshal operation (read-heavy)
					_, _ = w.Marshal()
				}
			})
		})
	}
}

// BenchmarkWorkMarshal tests just the marshal operation which is called during flushWorkCache
func BenchmarkWorkMarshal(b *testing.B) {
	testCases := []struct {
		name    string
		numJobs int
	}{
		{"jobs_100", 100},
		{"jobs_1000", 1000},
		{"jobs_5000", 5000},
		{"jobs_10000", 10000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			w := createTestWork(b, tc.numJobs)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := w.Marshal()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestLockContentionScenario demonstrates the lock contention scenario
func TestLockContentionScenario(t *testing.T) {
	w := createTestWork(t, 1000)

	// Simulate high contention scenario
	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			ctx := context.Background()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix of operations that happen in production
				w.GetJobForWorker(ctx, workerID)
				w.Len()
				w.ListJobs()
				_, _ = w.Marshal()
			}
		}(fmt.Sprintf("worker-%d", i))
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Adjust timeout based on race detector overhead
	timeout := 5 * time.Second
	if isRaceEnabled() {
		timeout = 40 * time.Second // Race detector adds ~13x overhead
	}

	t.Logf("Lock contention test completed in %v", elapsed)
	require.True(t, elapsed < timeout, "Lock contention test took too long: %v (timeout: %v, race detector: %v)", elapsed, timeout, isRaceEnabled())
}
