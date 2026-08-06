package backendscheduler

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// traceWithNamespace builds a trace whose resource carries a specific namespace attribute,
// so a TraceQL query can select it.
func traceWithNamespace(id common.ID, ns string) *tempopb.Trace {
	attrs := []*v1_common.KeyValue{{
		Key:   "namespace",
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: ns}},
	}}
	return &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{test.MakeBatchWithAttributes(2, id, attrs)}}
}

// writeRealTenantBlock writes a complete block (with trace data, not just a meta) to the store's
// backend so it is discoverable via BlockMetas and readable by RedactBlock.
func writeRealTenantBlock(ctx context.Context, t *testing.T, store storage.Store, tenant string, traces []*tempopb.Trace, ids []common.ID) {
	t.Helper()
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)
	meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: tenant}
	head, err := store.WAL().NewBlock(meta, model.CurrentEncoding)
	require.NoError(t, err)

	now := uint32(time.Now().Unix())
	for i, tr := range traces {
		b1, err := dec.PrepareForWrite(tr, 0, 0)
		require.NoError(t, err)
		b2, err := dec.ToObject([][]byte{b1})
		require.NoError(t, err)
		require.NoError(t, head.Append(ids[i], b2, now, now, true))
	}
	_, err = store.CompleteBlock(ctx, head)
	require.NoError(t, err)
}

// TestSubmitRedactionQueryEndToEnd exercises the query selector across the full submission →
// per-block execution path on real multi-block storage: a query submitted through the real API
// fans out one job per block, and executing those jobs redacts exactly the matching trace while
// leaving non-matching traces (and non-matching blocks) untouched.
//
// It drives the jobs the way Next() + the worker do — injecting the batch's selector into each
// job detail, then calling store.RedactBlock — rather than through the provider channel, which
// is non-deterministic in tests (the RedactionProvider goroutine races to drain pending jobs).
func TestSubmitRedactionQueryEndToEnd(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir + "/work"

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	tenant := "tenant-redact-e2e"
	idMatch := test.ValidTraceID(nil)
	idKeepA := test.ValidTraceID(nil)
	idKeepB := test.ValidTraceID(nil)

	// Block A holds the matching trace plus a keeper; block B holds only a keeper.
	writeRealTenantBlock(ctx, t, store, tenant,
		[]*tempopb.Trace{traceWithNamespace(idMatch, "secret"), traceWithNamespace(idKeepA, "keep")},
		[]common.ID{idMatch, idKeepA})
	writeRealTenantBlock(ctx, t, store, tenant,
		[]*tempopb.Trace{traceWithNamespace(idKeepB, "keep")},
		[]common.ID{idKeepB})

	require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == 2 },
		3*time.Second, 50*time.Millisecond, "blocklist poll should discover both blocks")

	query := `{resource.namespace = "secret"}`
	resp, err := s.SubmitRedaction(user.InjectOrgID(ctx, tenant), &tempopb.SubmitRedactionRequest{
		Selector: &tempopb.SubmitRedactionRequest_Query{Query: &tempopb.TraceQLSelector{Query: query}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, resp.JobsCreated, "one redaction job per block")

	batch := s.work.GetBatch(tenant)
	require.NotNil(t, batch)
	require.NotNil(t, batch.Query)
	require.Equal(t, query, batch.Query.Query)

	metaByID := make(map[string]*backend.BlockMeta)
	for _, m := range store.BlockMetas(tenant) {
		metaByID[m.BlockID.String()] = m
	}

	totalFound, rewroteBlocks := 0, 0
	for _, j := range s.work.ListAllPendingJobs() {
		require.Equal(t, tempopb.JobType_JOB_TYPE_REDACTION, j.GetType())
		rd := j.JobDetail.Redaction

		// Next() injects the batch selector; the worker forwards it to RedactBlock.
		rd.Query = batch.Query
		rd.Mode = batch.Mode

		meta := metaByID[rd.BlockId]
		require.NotNil(t, meta, "job references a discovered block")

		rewrote, found, _, err := store.RedactBlock(ctx, meta, tenant, nil, rd.Query.GetQuery(), rd.Mode)
		require.NoError(t, err)
		totalFound += found
		if rewrote {
			rewroteBlocks++
		}
	}

	require.Equal(t, 1, totalFound, "exactly the one matching trace is selected across all blocks")
	require.Equal(t, 1, rewroteBlocks, "only the block containing the match is rewritten")
}
