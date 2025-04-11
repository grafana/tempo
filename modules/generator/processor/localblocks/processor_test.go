package localblocks

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

type mockOverrides struct{}

var _ ProcessorOverrides = (*mockOverrides)(nil)

func (m *mockOverrides) DedicatedColumns(string) backend.DedicatedColumns {
	return nil
}

func (m *mockOverrides) MaxLocalTracesPerUser(string) int {
	return 0
}

func (m *mockOverrides) MaxBytesPerTrace(string) int {
	return 0
}

func (m *mockOverrides) UnsafeQueryHints(string) bool {
	return false
}

var _ tempodb.Writer = (*mockWriter)(nil)

type mockWriter struct {
	mtx    sync.Mutex
	blocks []*backend.BlockMeta
}

func (m *mockWriter) metas() []*backend.BlockMeta {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.blocks
}

func (m *mockWriter) WriteBlock(_ context.Context, b tempodb.WriteableBlock) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.blocks = append(m.blocks, b.BlockMeta())
	return nil
}

func (m *mockWriter) CompleteBlock(context.Context, common.WALBlock) (common.BackendBlock, error) {
	return nil, nil
}

func (m *mockWriter) CompleteBlockWithBackend(context.Context, common.WALBlock, backend.Reader, backend.Writer) (common.BackendBlock, error) {
	return nil, nil
}

func (m *mockWriter) WAL() *wal.WAL { return nil }

func TestProcessorDoesNotRace(t *testing.T) {
	wal, err := wal.New(&wal.Config{
		Filepath: t.TempDir(),
		Version:  encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)

	var (
		ctx    = context.Background()
		tenant = "fake"
		cfg    = Config{
			Concurrency:          1,
			FlushCheckPeriod:     10 * time.Millisecond,
			TraceIdlePeriod:      time.Second,
			CompleteBlockTimeout: time.Minute,
			Block: &common.BlockConfig{
				BloomShardSizeBytes: 100_000,
				BloomFP:             0.05,
				Version:             encoding.DefaultEncoding().Version(),
			},
			Metrics: MetricsConfig{
				ConcurrentBlocks:  10,
				TimeOverlapCutoff: 0.2,
			},
		}
		overrides = &mockOverrides{}
		e         = traceql.NewEngine()
	)

	p, err := New(cfg, tenant, wal, &mockWriter{}, overrides)
	require.NoError(t, err)
	defer p.Shutdown(t.Context())

	qr := &tempopb.QueryRangeRequest{
		Query: "{} | rate() by (resource.service.name)",
		Start: uint64(time.Now().Add(-5 * time.Minute).UnixNano()),
		End:   uint64(time.Now().UnixNano()),
		Step:  uint64(30 * time.Second),
	}
	me, err := e.CompileMetricsQueryRange(qr, 0, 0, false)
	require.NoError(t, err)

	je, err := e.CompileMetricsQueryRangeNonRaw(qr, traceql.AggregateModeSum)
	require.NoError(t, err)

	var (
		end = make(chan struct{})
		wg  = sync.WaitGroup{}
	)

	concurrent := func(f func()) {
		wg.Add(1)
		defer wg.Done()

		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	go concurrent(func() {
		tr := test.MakeTrace(10, nil)
		for _, b := range tr.ResourceSpans {
			for _, ss := range b.ScopeSpans {
				for _, s := range ss.Spans {
					s.Kind = v1.Span_SPAN_KIND_SERVER
				}
			}
		}

		req := &tempopb.PushSpansRequest{
			Batches: tr.ResourceSpans,
		}
		p.PushSpans(ctx, req)
	})

	go concurrent(func() {
		err := p.cutIdleTraces(true)
		require.NoError(t, err, "cutting idle traces")
	})

	go concurrent(func() {
		err := p.cutBlocks(true)
		require.NoError(t, err, "cutting blocks")
	})

	go concurrent(func() {
		err := p.flushBlock(uuid.New())
		require.NoError(t, err, "flushing blocks")
	})

	go concurrent(func() {
		err := p.deleteOldBlocks()
		require.NoError(t, err, "deleting old blocks")
	})

	// Run multiple queries
	go concurrent(func() {
		_, err := p.GetMetrics(ctx, &tempopb.SpanMetricsRequest{
			Query:   "{}",
			GroupBy: "status",
		})
		require.NoError(t, err)
	})

	go concurrent(func() {
		_, err := p.GetMetrics(ctx, &tempopb.SpanMetricsRequest{
			Query:   "{}",
			GroupBy: "status",
		})
		require.NoError(t, err)
	})

	go concurrent(func() {
		err := p.QueryRange(ctx, qr, me, je)
		require.NoError(t, err)
	})

	// Run for a bit
	time.Sleep(2000 * time.Millisecond)

	// Cleanup
	close(end)
	wg.Wait()
}

func TestReplicationFactor(t *testing.T) {
	wal, err := wal.New(&wal.Config{
		Filepath: t.TempDir(),
		Version:  encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)

	cfg := Config{
		Concurrency:          1,
		FlushCheckPeriod:     time.Minute,
		TraceIdlePeriod:      time.Minute,
		CompleteBlockTimeout: time.Minute,
		Block: &common.BlockConfig{
			BloomShardSizeBytes: 100_000,
			BloomFP:             0.05,
			Version:             encoding.DefaultEncoding().Version(),
		},
		Metrics: MetricsConfig{
			ConcurrentBlocks:  10,
			TimeOverlapCutoff: 0.2,
		},
		FilterServerSpans: false,
		FlushToStorage:    true,
	}

	mockWriter := &mockWriter{}

	p, err := New(cfg, "fake", wal, mockWriter, &mockOverrides{})
	require.NoError(t, err)
	defer p.Shutdown(t.Context())

	tr := test.MakeTrace(10, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	p.PushSpans(context.TODO(), &tempopb.PushSpansRequest{
		Batches: tr.ResourceSpans,
	})

	require.NoError(t, p.cutIdleTraces(true))

	p.blocksMtx.Lock()
	verifyReplicationFactor(t, p.headBlock)
	p.blocksMtx.Unlock()

	require.NoError(t, p.cutBlocks(true))

	p.blocksMtx.Lock()
	for _, b := range p.walBlocks {
		verifyReplicationFactor(t, b)
	}
	p.blocksMtx.Unlock()

	require.Eventually(t, func() bool {
		p.blocksMtx.Lock()
		defer p.blocksMtx.Unlock()
		return len(p.completeBlocks) > 0
	}, 10*time.Second, 100*time.Millisecond)

	p.blocksMtx.Lock()
	for _, b := range p.completeBlocks {
		verifyReplicationFactor(t, b)
	}
	p.blocksMtx.Unlock()

	require.Eventually(t, func() bool {
		return len(mockWriter.metas()) == 1
	}, 10*time.Second, 100*time.Millisecond)
	p.blocksMtx.Lock()
	verifyReplicationFactor(t, &mockBlock{meta: mockWriter.metas()[0]})
	p.blocksMtx.Unlock()
}

func verifyReplicationFactor(t *testing.T, b common.BackendBlock) {
	require.Equal(t, 1, int(b.BlockMeta().ReplicationFactor))
}

func TestBadBlocks(t *testing.T) {
	wal, err := wal.New(&wal.Config{
		Filepath: t.TempDir(),
		Version:  encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)

	u1 := uuid.New().String()

	// Write some bad wal data
	writeBadJSON(t,
		filepath.Join(
			wal.GetFilepath(),
			u1+"+test-tenant+"+vparquet3.VersionString,
			backend.MetaName,
		),
	)

	writeBadJSON(t,
		filepath.Join(
			wal.GetFilepath(),
			u1+"+test-tenant+"+vparquet3.VersionString,
			"0000000001",
		),
	)

	writeBadJSON(t,
		filepath.Join(
			wal.GetFilepath(),
			uuid.New().String()+"+test-tenant+"+vparquet3.VersionString,
			"0000000001",
		),
	)

	// write a bad block meta for a completed block
	writeBadJSON(t,
		filepath.Join(
			wal.GetFilepath(),
			"blocks",
			"test-tenant",
			uuid.New().String(),
			backend.MetaName,
		),
	)

	cfg := Config{
		FlushCheckPeriod:     time.Minute,
		TraceIdlePeriod:      time.Minute,
		CompleteBlockTimeout: time.Minute,
		Block: &common.BlockConfig{
			Version: encoding.DefaultEncoding().Version(),
		},
	}

	p, err := New(cfg, "test-tenant", wal, nil, &mockOverrides{})
	require.NoError(t, err)
	p.Shutdown(t.Context())
}

func TestProcessorWithNonEmptyWAL(t *testing.T) {
	wal, err := wal.New(&wal.Config{
		Filepath: t.TempDir(),
		Version:  encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)

	// write to the wal
	tenantID := "test-tenant"
	traceID := test.ValidTraceID(nil)

	meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: tenantID}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
	require.NoError(t, err)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	obj, err := dec.PrepareForWrite(test.MakeTrace(1, traceID), 0, 0)
	require.NoError(t, err)

	obj2, err := dec.ToObject([][]byte{obj})
	require.NoError(t, err)

	err = head.Append(traceID, obj2, 0, 0, true)
	require.NoError(t, err)

	err = head.Flush()
	require.NoError(t, err)

	// create a new processor
	cfg := Config{
		FlushCheckPeriod:     time.Minute,
		TraceIdlePeriod:      time.Minute,
		CompleteBlockTimeout: time.Second,
		Block: &common.BlockConfig{
			Version: encoding.DefaultEncoding().Version(),
		},
	}

	p, err := New(cfg, tenantID, wal, nil, &mockOverrides{})
	require.NoError(t, err)
	p.Shutdown(t.Context())
}

func writeBadJSON(t *testing.T, path string) {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0o700)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("{")
	require.NoError(t, err)
}

var _ common.BackendBlock = (*mockBlock)(nil)

type mockBlock struct {
	meta *backend.BlockMeta
}

func (m *mockBlock) FindTraceByID(context.Context, common.ID, common.SearchOptions) (*tempopb.TraceByIDResponse, error) {
	return nil, nil
}

func (m *mockBlock) Search(context.Context, *tempopb.SearchRequest, common.SearchOptions) (*tempopb.SearchResponse, error) {
	return nil, nil
}

func (m *mockBlock) SearchTags(context.Context, traceql.AttributeScope, common.TagsCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (m *mockBlock) SearchTagValues(context.Context, string, common.TagValuesCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (m *mockBlock) SearchTagValuesV2(context.Context, traceql.Attribute, common.TagValuesCallbackV2, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (m *mockBlock) Fetch(context.Context, traceql.FetchSpansRequest, common.SearchOptions) (traceql.FetchSpansResponse, error) {
	return traceql.FetchSpansResponse{}, nil
}

func (m *mockBlock) FetchTagValues(context.Context, traceql.FetchTagValuesRequest, traceql.FetchTagValuesCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (m *mockBlock) FetchTagNames(context.Context, traceql.FetchTagsRequest, traceql.FetchTagsCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (m *mockBlock) BlockMeta() *backend.BlockMeta { return m.meta }

func (m *mockBlock) Validate(context.Context) error { return nil }
