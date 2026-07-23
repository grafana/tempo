// vp5 search ground truth (test-only): builds a real in-memory vparquet5 block from a proto trace and
// runs the standard search engine over it. This is the ground truth the in-memory Filter is checked
// against. It lives in test code so none of the block-building or backend machinery ships in the binary
// - the Filter itself never builds a block.

package tracefilter

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
)

// high enough that ExecuteSearch never truncates: trace-by-id must return every matched span.
const blockFilterMaxResults = 1_000_000

// buildVp5Block builds and opens an in-memory vparquet5 block from trace. you can query it with searchVp5Block.
func buildVp5Block(tb testing.TB, trace *tempopb.Trace) common.BackendBlock {
	tb.Helper()
	ctx := context.Background()

	traceID := firstTraceID(trace)
	require.NotNil(tb, traceID)

	be := newInMemBackend()
	r := backend.NewReader(be)
	w := backend.NewWriter(be)

	start, end := traceTimeRange(trace)
	meta := backend.NewBlockMeta("single-tenant", uuid.New(), vparquet5.VersionString)
	meta.TotalObjects = 1
	meta.StartTime = start
	meta.EndTime = end

	cfg := &common.BlockConfig{BloomFP: 0.01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 100_000_000}
	enc, err := encoding.FromVersionForWrites(vparquet5.VersionString)
	require.NoError(tb, err)

	newMeta, err := enc.CreateBlock(ctx, cfg, meta, newSingleTraceIterator(traceID, trace), r, w)
	require.NoError(tb, err)

	block, err := encoding.OpenBlock(newMeta, r)
	require.NoError(tb, err)

	return block
}

// searchVp5Block runs traceql.Engine.ExecuteSearch (the standard search path) against a prebuilt block
// and returns the sorted matched hex span ids.
func searchVp5Block(tb testing.TB, block common.BackendBlock, query string) []string {
	tb.Helper()
	ctx := context.Background()

	fetcher := traceql.NewSpansetFetcherWrapperBoth(
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return block.Fetch(ctx, req, common.DefaultSearchOptions())
		},
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			return block.FetchSpans(ctx, req, common.DefaultSearchOptions())
		},
	)
	resp, err := traceql.NewEngine().ExecuteSearch(ctx, &tempopb.SearchRequest{
		Query:           query,
		Limit:           blockFilterMaxResults,
		SpansPerSpanSet: blockFilterMaxResults,
	}, fetcher)
	require.NoError(tb, err)

	var got []string
	for _, tr := range resp.Traces {
		for _, ss := range tr.SpanSets {
			for _, sp := range ss.Spans {
				got = append(got, sp.SpanID)
			}
		}
	}
	sort.Strings(got)
	return got
}

// newSingleTraceIterator wraps one proto trace in a one-shot common.Iterator for CreateBlock.
func newSingleTraceIterator(traceID []byte, trace *tempopb.Trace) common.Iterator {
	return &singleTraceIterator{id: traceID, trace: trace}
}

type singleTraceIterator struct {
	id    []byte
	trace *tempopb.Trace
	done  bool
}

func (i *singleTraceIterator) Next(context.Context) (common.ID, *tempopb.Trace, error) {
	if i.done {
		return nil, nil, nil
	}
	i.done = true
	return i.id, i.trace, nil
}

func (i *singleTraceIterator) Close() {}

// firstTraceID returns the trace id from the first span, or nil if the trace has no spans.
func firstTraceID(trace *tempopb.Trace) []byte {
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				return span.TraceId
			}
		}
	}
	return nil
}

// traceTimeRange returns the [min start, max end] of the trace as block meta bounds. Time filtering is
// disabled at query time, so these only need to be a valid, non-degenerate range.
func traceTimeRange(trace *tempopb.Trace) (time.Time, time.Time) {
	var start, end uint64
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if start == 0 || span.StartTimeUnixNano < start {
					start = span.StartTimeUnixNano
				}
				if span.EndTimeUnixNano > end {
					end = span.EndTimeUnixNano
				}
			}
		}
	}
	if start == 0 {
		start = uint64(time.Now().UnixNano())
	}
	if end < start {
		end = start
	}
	return time.Unix(0, int64(start)), time.Unix(0, int64(end))
}

var (
	_ backend.RawReader = (*inMemBackend)(nil)
	_ backend.RawWriter = (*inMemBackend)(nil)
)

// inMemBackend is a RawReader+RawWriter over a map[string][]byte that round-trips written bytes.
// backend.MockRawReader does not read back what MockRawWriter stored, so it is unusable for a real
// block build+open cycle - this one is.
type inMemBackend struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func newInMemBackend() *inMemBackend {
	return &inMemBackend{objects: make(map[string][]byte)}
}

// objectKey mirrors the on-disk layout: keypath segments then the object name, so a block's objects
// share a common prefix that List/Find/ListBlocks can walk.
func objectKey(keypath backend.KeyPath, name string) string {
	parts := append(append([]string(nil), keypath...), name)
	return strings.Join(parts, "/")
}

// inMemAppendTracker carries the target key across Append calls for one object.
type inMemAppendTracker struct {
	key string
}

// Write stores the full object, replacing any existing bytes at the same key.
func (b *inMemBackend) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, _ int64, _ *backend.CacheInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	buf, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.objects[objectKey(keypath, name)] = buf
	b.mu.Unlock()
	return nil
}

// Append starts (tracker nil) or continues an object, accumulating bytes in place.
func (b *inMemBackend) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if tracker == nil {
		// nil tracker starts a fresh object, truncating any prior bytes like local's os.Create.
		t := &inMemAppendTracker{key: objectKey(keypath, name)}
		b.objects[t.key] = append([]byte(nil), buffer...)
		return t, nil
	}

	t := tracker.(*inMemAppendTracker)
	b.objects[t.key] = append(b.objects[t.key], buffer...)
	return t, nil
}

// CloseAppend is a no-op: Append already committed the bytes.
func (b *inMemBackend) CloseAppend(ctx context.Context, _ backend.AppendTracker) error {
	return ctx.Err()
}

func (b *inMemBackend) Delete(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	delete(b.objects, objectKey(keypath, name))
	b.mu.Unlock()
	return nil
}

// Read returns the whole object. Parquet page reads go through ReadRange instead.
func (b *inMemBackend) Read(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) (io.ReadCloser, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, -1, err
	}
	b.mu.RLock()
	data, ok := b.objects[objectKey(keypath, name)]
	b.mu.RUnlock()
	if !ok {
		return nil, -1, backend.ErrDoesNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

// ReadRange copies buffer-many bytes at offset - the load-bearing path for parquet page reads.
func (b *inMemBackend) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, _ *backend.CacheInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.RLock()
	data, ok := b.objects[objectKey(keypath, name)]
	b.mu.RUnlock()
	if !ok {
		return backend.ErrDoesNotExist
	}
	if offset+uint64(len(buffer)) > uint64(len(data)) {
		return io.ErrUnexpectedEOF
	}
	copy(buffer, data[offset:])
	return nil
}

// List returns the distinct object names one level beneath keypath.
func (b *inMemBackend) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	prefix := strings.Join(keypath, "/")
	if prefix != "" {
		prefix += "/"
	}
	seen := make(map[string]struct{})
	var out []string
	b.mu.RLock()
	for k := range b.objects {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		rest := strings.TrimPrefix(k, prefix)
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			rest = rest[:i]
		}
		if _, ok := seen[rest]; ok {
			continue
		}
		seen[rest] = struct{}{}
		out = append(out, rest)
	}
	b.mu.RUnlock()
	return out, nil
}

// ListBlocks parses tenant/<blockID>/meta keys into meta and compacted-meta block ids.
func (b *inMemBackend) ListBlocks(ctx context.Context, tenant string) ([]uuid.UUID, []uuid.UUID, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	prefix := tenant + "/"
	var metas, compacted []uuid.UUID
	b.mu.RLock()
	for k := range b.objects {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		parts := strings.Split(k, "/")
		if len(parts) != 3 {
			continue
		}
		id, err := uuid.Parse(parts[1])
		if err != nil {
			continue
		}
		switch parts[2] {
		case backend.MetaName:
			metas = append(metas, id)
		case backend.CompactedMetaName:
			compacted = append(compacted, id)
		}
	}
	b.mu.RUnlock()
	return metas, compacted, nil
}

func (b *inMemBackend) Find(ctx context.Context, keypath backend.KeyPath, f backend.FindFunc) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	prefix := strings.Join(keypath, "/")
	if prefix != "" {
		prefix += "/"
	}
	b.mu.RLock()
	keys := make([]string, 0, len(b.objects))
	for k := range b.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	b.mu.RUnlock()
	for _, k := range keys {
		f(backend.FindMatch{Key: k})
	}
	return nil
}

func (b *inMemBackend) Shutdown() {}

// TestInMemBackendRoundTrips locks the one property the block build depends on and that
// backend.MockRawReader lacks: bytes written come back byte-identical via Read and ReadRange.
func TestInMemBackendRoundTrips(t *testing.T) {
	ctx := context.Background()
	be := newInMemBackend()
	kp := backend.KeyPathForBlock(uuid.New(), "single-tenant")

	t.Run("Write then Read", func(t *testing.T) {
		payload := []byte("hello parquet block")
		require.NoError(t, be.Write(ctx, "meta.json", kp, bytes.NewReader(payload), int64(len(payload)), nil))

		rc, size, err := be.Read(ctx, "meta.json", kp, nil)
		require.NoError(t, err)
		require.Equal(t, int64(len(payload)), size)
		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		require.Equal(t, payload, got)
	})

	t.Run("Append across calls then ReadRange", func(t *testing.T) {
		data := "data.parquet"
		tracker, err := be.Append(ctx, data, kp, nil, []byte("PAR1"))
		require.NoError(t, err)
		tracker, err = be.Append(ctx, data, kp, tracker, []byte("payload-bytes"))
		require.NoError(t, err)
		require.NoError(t, be.CloseAppend(ctx, tracker))

		full := []byte("PAR1payload-bytes")

		rc, size, err := be.Read(ctx, data, kp, nil)
		require.NoError(t, err)
		require.Equal(t, int64(len(full)), size)
		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		require.Equal(t, full, got)

		// ReadRange is the load-bearing parquet page read: exact bytes at an arbitrary offset.
		buf := make([]byte, 4)
		require.NoError(t, be.ReadRange(ctx, data, kp, uint64(len(full)-4), buf, nil))
		require.Equal(t, full[len(full)-4:], buf)
	})

	t.Run("nil tracker truncates a prior object", func(t *testing.T) {
		name := "truncate-me"
		tracker, err := be.Append(ctx, name, kp, nil, []byte("first-write-longer"))
		require.NoError(t, err)
		require.NoError(t, be.CloseAppend(ctx, tracker))

		tracker, err = be.Append(ctx, name, kp, nil, []byte("second"))
		require.NoError(t, err)
		require.NoError(t, be.CloseAppend(ctx, tracker))

		_, size, err := be.Read(ctx, name, kp, nil)
		require.NoError(t, err)
		require.Equal(t, int64(len("second")), size)
	})

	t.Run("missing object and out-of-range read error", func(t *testing.T) {
		_, _, err := be.Read(ctx, "nope", kp, nil)
		require.ErrorIs(t, err, backend.ErrDoesNotExist)

		require.ErrorIs(t, be.ReadRange(ctx, "nope", kp, 0, make([]byte, 1), nil), backend.ErrDoesNotExist)

		require.NoError(t, be.Write(ctx, "small", kp, bytes.NewReader([]byte("ab")), 2, nil))
		require.ErrorIs(t, be.ReadRange(ctx, "small", kp, 0, make([]byte, 8), nil), io.ErrUnexpectedEOF)
	})

	t.Run("ListBlocks finds the block after WriteBlockMeta", func(t *testing.T) {
		blockID := uuid.New()
		metaKP := backend.KeyPathForBlock(blockID, "single-tenant")
		require.NoError(t, be.Write(ctx, backend.MetaName, metaKP, bytes.NewReader([]byte("{}")), 2, nil))

		metas, compacted, err := be.ListBlocks(ctx, "single-tenant")
		require.NoError(t, err)
		require.Contains(t, metas, blockID)
		require.Empty(t, compacted)
	})
}
