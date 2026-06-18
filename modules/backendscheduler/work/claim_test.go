package work

import (
	"flag"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func newTestWork(t *testing.T) *Work {
	t.Helper()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	return New(cfg).(*Work)
}

func compactionJob(tenant string, inputs ...string) *Job {
	return &Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:     tenant,
			Compaction: &tempopb.CompactionDetail{Input: inputs},
		},
	}
}

func redactionBatch(id string) *tempopb.RedactionBatch {
	return &tempopb.RedactionBatch{BatchId: id, TenantId: "t1", TraceIds: [][]byte{[]byte("trace")}}
}

func toSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

// Ordering A: a compaction job registered BEFORE the claim is seen by the claim's
// busy snapshot, so its input block is recorded for rescan (skipped), never redacted
// directly.
func TestClaimRedactionBatch_RegisterThenClaim(t *testing.T) {
	w := newTestWork(t)
	j := compactionJob("t1", "B")
	require.True(t, w.TryRegisterJob(j), "no batch active yet → job registers")
	require.True(t, w.IsBlockBusy("t1", "B"))

	free, skipped, ok := w.ClaimRedactionBatch(redactionBatch("batch"), []string{"B", "C"}, int64(12345))
	require.True(t, ok)
	require.Equal(t, []string{j.ID}, skipped, "block under active compaction recorded for rescan")
	require.Equal(t, []string{"C"}, free, "only the free block is redacted directly")
}

// Ordering B: once the claim has installed the barrier, a compaction job for the same
// tenant is refused — so the block stays put and the redaction job (built from `free`)
// covers it.
func TestClaimRedactionBatch_ClaimThenRegisterRefused(t *testing.T) {
	w := newTestWork(t)
	free, skipped, ok := w.ClaimRedactionBatch(redactionBatch("batch"), []string{"B"}, int64(12345))
	require.True(t, ok)
	require.Equal(t, []string{"B"}, free)
	require.Empty(t, skipped)

	j := compactionJob("t1", "B")
	require.False(t, w.TryRegisterJob(j), "compaction must be refused while a batch is active")
	require.False(t, w.IsBlockBusy("t1", "B"), "refused job must not be registered")
}

func TestClaimRedactionBatch_RejectsConcurrentBatch(t *testing.T) {
	w := newTestWork(t)
	_, _, ok := w.ClaimRedactionBatch(redactionBatch("b1"), []string{"B"}, 0)
	require.True(t, ok)

	free, skipped, ok2 := w.ClaimRedactionBatch(redactionBatch("b2"), []string{"C"}, 0)
	require.False(t, ok2, "second batch for the same tenant must be rejected")
	require.Nil(t, free)
	require.Nil(t, skipped)
}

// The core safety property under concurrency: for every compaction job racing the
// claim, EITHER it registered (and the claim recorded it for rescan) OR it was refused
// (and the claim left its block free to redact). Never registered-AND-free, which would
// leave a compaction output uncovered → trace survives redaction. Run with -race.
func TestClaimRedactionBatch_RegisterRaceInvariant(t *testing.T) {
	const N = 64
	w := newTestWork(t)

	jobs := make([]*Job, N)
	blockIDs := make([]string, N)
	for i := range jobs {
		blockIDs[i] = fmt.Sprintf("B%d", i)
		jobs[i] = compactionJob("t1", blockIDs[i])
	}

	registered := make([]bool, N)
	var free, skipped []string

	var wg sync.WaitGroup
	wg.Add(N + 1)
	go func() {
		defer wg.Done()
		free, skipped, _ = w.ClaimRedactionBatch(redactionBatch("b"), blockIDs, int64(1))
	}()
	for i := range jobs {
		go func() {
			defer wg.Done()
			registered[i] = w.TryRegisterJob(jobs[i])
		}()
	}
	wg.Wait()

	freeSet := toSet(free)
	skippedSet := toSet(skipped)
	for i, j := range jobs {
		if registered[i] {
			require.Contains(t, skippedSet, j.ID,
				"job %d (block %s) registered before the claim must be recorded for rescan", i, blockIDs[i])
			require.NotContains(t, freeSet, blockIDs[i],
				"block %s under compaction must not also be redacted directly", blockIDs[i])
		} else {
			require.Contains(t, freeSet, blockIDs[i],
				"block %s whose compaction was refused must be redacted directly", blockIDs[i])
		}
	}
}
