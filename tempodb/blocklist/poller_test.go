package blocklist

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
)

var (
	testPollConcurrency       = uint(10)
	testTenantPollConcurrency = uint(2)
	testPollFallback          = true
	testBuilders              = 1
	testEmptyTenantIndexAge   = 1 * time.Minute
)

type mockJobSharder struct {
	owns bool
}

func (m *mockJobSharder) Owns(_ string) bool { return m.owns }

func TestTenantIndexBuilder(t *testing.T) {
	var (
		one   = backend.MustParse("00000000-0000-0000-0000-000000000001")
		two   = backend.MustParse("00000000-0000-0000-0000-000000000002")
		three = backend.MustParse("00000000-0000-0000-0000-000000000003")
	)

	tests := []struct {
		name                  string
		list                  PerTenant
		compactedList         PerTenantCompacted
		expectedList          PerTenant
		expectedCompactedList PerTenantCompacted
		expectsError          bool
	}{
		{
			name:                  "nothing!",
			expectedList:          PerTenant{},
			expectedCompactedList: PerTenantCompacted{},
		},
		{
			name: "err",
			list: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: one,
					},
				},
			},
			expectsError: true,
		},
		{
			name: "block meta",
			list: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: one,
					},
				},
			},
			expectedList: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: one,
					},
				},
			},
			expectedCompactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{},
			},
		},
		{
			name: "compacted block meta",
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: one,
						},
					},
				},
			},
			expectedList: PerTenant{
				"test": []*backend.BlockMeta{},
			},
			expectedCompactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: one,
						},
					},
				},
			},
		},
		{
			name: "all",
			list: PerTenant{
				"test2": []*backend.BlockMeta{
					{
						BlockID: three,
					},
				},
				"test": []*backend.BlockMeta{
					{
						BlockID: two,
					},
				},
			},
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: one,
						},
					},
				},
			},
			expectedList: PerTenant{
				"test2": []*backend.BlockMeta{
					{
						BlockID: three,
					},
				},
				"test": []*backend.BlockMeta{
					{
						BlockID: two,
					},
				},
			},
			expectedCompactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: one,
						},
					},
				},
				"test2": []*backend.CompactedBlockMeta{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newMockCompactor(tc.compactedList, tc.expectsError)
			r := newMockReader(tc.list, tc.compactedList, tc.expectsError)
			w := &backend.MockWriter{}
			b := newBlocklist(PerTenant{}, PerTenantCompacted{})

			poller := NewPoller(&PollerConfig{
				PollConcurrency:       testPollConcurrency,
				TenantPollConcurrency: testTenantPollConcurrency,
				PollFallback:          testPollFallback,
				TenantIndexBuilders:   testBuilders,
			}, &mockJobSharder{
				owns: true,
			}, r, c, w, log.NewNopLogger())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			actualList, actualCompactedList, _, err := poller.Do(ctx, b)

			// confirm return as expected
			assert.Equal(t, tc.expectedList, actualList)
			assert.Equal(t, tc.expectedCompactedList, actualCompactedList)
			if tc.expectsError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// confirm tenant index written as expected
			for tenant, list := range tc.expectedList {
				assert.Equal(t, list, w.IndexMeta[tenant])
			}
			for tenant, list := range tc.expectedCompactedList {
				assert.Equal(t, list, w.IndexCompactedMeta[tenant])
			}
		})
	}
}

func TestTenantIndexFallback(t *testing.T) {
	tests := []struct {
		name                      string
		isTenantIndexBuilder      bool
		errorOnCreateTenantIndex  bool
		pollFallback              bool
		expectsError              bool
		expectsTenantIndexWritten bool
		staleTenantIndex          time.Duration
	}{
		{
			name:                      "builder writes index",
			isTenantIndexBuilder:      true,
			expectsTenantIndexWritten: true,
		},
		{
			name:                      "reader does not write index",
			isTenantIndexBuilder:      false,
			expectsTenantIndexWritten: false,
		},
		{
			name:                      "reader does not write index on error if no fallback",
			isTenantIndexBuilder:      false,
			errorOnCreateTenantIndex:  true,
			pollFallback:              false,
			expectsError:              true,
			expectsTenantIndexWritten: false,
		},
		{
			name:                      "reader writes index on error if fallback",
			isTenantIndexBuilder:      false,
			errorOnCreateTenantIndex:  true,
			pollFallback:              true,
			expectsError:              false,
			expectsTenantIndexWritten: true,
		},
		{
			name:                      "reader does not write index on stale if no fallback",
			isTenantIndexBuilder:      false,
			pollFallback:              false,
			expectsError:              true,
			expectsTenantIndexWritten: false,
			staleTenantIndex:          time.Second,
		},
		{
			name:                      "reader writes index on stale if fallback",
			isTenantIndexBuilder:      false,
			pollFallback:              true,
			expectsError:              false,
			expectsTenantIndexWritten: true,
			staleTenantIndex:          time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &backend.MockCompactor{}
			r := newMockReader(PerTenant{
				"test": []*backend.BlockMeta{},
			}, nil, false)
			w := &backend.MockWriter{}
			b := newBlocklist(PerTenant{}, PerTenantCompacted{})

			r.(*backend.MockReader).TenantIndexFn = func(_ context.Context, _ string) (*backend.TenantIndex, error) {
				if tc.errorOnCreateTenantIndex {
					return nil, errors.New("err")
				}
				return &backend.TenantIndex{
					CreatedAt: time.Now().
						Add(-5 * time.Minute),
					// always make the tenant index 5 minutes old so the above tests can use that for fallback testing
				}, nil
			}

			poller := NewPoller(&PollerConfig{
				PollConcurrency:        testPollConcurrency,
				TenantPollConcurrency:  testTenantPollConcurrency,
				PollFallback:           tc.pollFallback,
				TenantIndexBuilders:    testBuilders,
				StaleTenantIndex:       tc.staleTenantIndex,
				EmptyTenantDeletionAge: testEmptyTenantIndexAge,
			}, &mockJobSharder{
				owns: tc.isTenantIndexBuilder,
			}, r, c, w, log.NewNopLogger())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, _, _, err := poller.Do(ctx, b)

			assert.Equal(t, tc.expectsError, err != nil)
			assert.Equal(t, tc.expectsTenantIndexWritten, w.IndexCompactedMeta != nil)
			assert.Equal(t, tc.expectsTenantIndexWritten, w.IndexMeta != nil)
		})
	}
}

func TestPollBlock(t *testing.T) {
	one := backend.MustParse("00000000-0000-0000-0000-000000000001")

	tests := []struct {
		name                  string
		list                  PerTenant
		compactedList         PerTenantCompacted
		pollTenantID          string
		pollBlockID           backend.UUID
		expectedMeta          *backend.BlockMeta
		expectedCompactedMeta *backend.CompactedBlockMeta
		expectsError          bool
	}{
		{
			name:         "block and tenant don't exist",
			pollTenantID: "test",
			pollBlockID:  one,
		},
		{
			name:         "block exists",
			pollTenantID: "test",
			pollBlockID:  one,
			list: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: one,
					},
				},
			},
			expectedMeta: &backend.BlockMeta{
				BlockID: one,
			},
		},
		{
			name:         "compactedblock exists",
			pollTenantID: "test",
			pollBlockID:  one,
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: one,
						},
					},
				},
			},
			expectedCompactedMeta: &backend.CompactedBlockMeta{
				BlockMeta: backend.BlockMeta{
					BlockID: one,
				},
			},
		},
		{
			name:         "errors",
			pollTenantID: "test",
			pollBlockID:  one,
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: one,
						},
					},
				},
			},
			expectsError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newMockCompactor(tc.compactedList, tc.expectsError)
			r := newMockReader(tc.list, nil, tc.expectsError)
			w := &backend.MockWriter{}

			poller := NewPoller(&PollerConfig{
				PollConcurrency:       testPollConcurrency,
				TenantPollConcurrency: testTenantPollConcurrency,
				PollFallback:          testPollFallback,
				TenantIndexBuilders:   testBuilders,
			}, &mockJobSharder{}, r, c, w, log.NewNopLogger())
			actualMeta, actualCompactedMeta, err := poller.pollBlock(context.Background(), tc.pollTenantID, uuid.UUID(tc.pollBlockID), false)

			assert.Equal(t, tc.expectedMeta, actualMeta)
			assert.Equal(t, tc.expectedCompactedMeta, actualCompactedMeta)
			if tc.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPollNoCompactFlag(t *testing.T) {
	var (
		tenantID  = "test"
		knownID   = backend.MustParse("00000000-0000-0000-0000-000000000001")
		unknownID = backend.MustParse("00000000-0000-0000-0000-000000000002")
		unflagged = backend.MustParse("00000000-0000-0000-0000-000000000003")
		notLiveID = backend.MustParse("00000000-0000-0000-0000-000000000004")
		knownMeta = &backend.BlockMeta{BlockID: knownID, TenantID: tenantID}
		otherMeta = &backend.BlockMeta{BlockID: unflagged, TenantID: tenantID}
		current   = PerTenant{tenantID: {knownMeta, {BlockID: unknownID, TenantID: tenantID}, otherMeta}}
		noCompact = []uuid.UUID{uuid.UUID(knownID), uuid.UUID(unknownID), uuid.UUID(notLiveID)}
		r         = newMockReader(current, nil, false).(*backend.MockReader)
		w         = &backend.MockWriter{}
		previous  = newBlocklist(PerTenant{tenantID: {knownMeta, otherMeta}}, PerTenantCompacted{})
		blocksFn  = r.BlocksFn
	)

	r.BlocksFn = func(ctx context.Context, tenantID string) ([]uuid.UUID, []uuid.UUID, []uuid.UUID, error) {
		ids, compactedIDs, _, err := blocksFn(ctx, tenantID)
		return ids, compactedIDs, noCompact, err
	}

	poller := NewPoller(&PollerConfig{
		PollConcurrency:       testPollConcurrency,
		TenantPollConcurrency: testTenantPollConcurrency,
		PollFallback:          testPollFallback,
		TenantIndexBuilders:   testBuilders,
	}, &mockJobSharder{owns: true}, r, newMockCompactor(nil, false), w, log.NewNopLogger())

	// Flags of known and unknown live blocks are returned and written to the tenant
	// index. A flag without a live block is dropped.
	metas, _, noCompactList, err := poller.Do(context.Background(), previous)
	require.NoError(t, err)
	require.Len(t, metas[tenantID], 3)
	require.ElementsMatch(t, []backend.UUID{knownID, unknownID}, noCompactList[tenantID])
	require.ElementsMatch(t, []backend.UUID{knownID, unknownID}, w.IndexNoCompact[tenantID])

	// Metas are passed through unchanged.
	require.Same(t, knownMeta, metas[tenantID][0])
	require.Same(t, otherMeta, metas[tenantID][1])

	// Deleting the flags clears them on the next poll.
	noCompact = nil
	_, _, noCompactList, err = poller.Do(context.Background(), newBlocklist(metas, PerTenantCompacted{}))
	require.NoError(t, err)
	require.Empty(t, noCompactList[tenantID])
	require.Empty(t, w.IndexNoCompact[tenantID])
}

func TestTenantIndexPollError(t *testing.T) {
	p := NewPoller(&PollerConfig{
		StaleTenantIndex: time.Minute,
	}, nil, nil, nil, nil, log.NewNopLogger())

	// tenant index doesn't matter if there's an error
	assert.Error(t, p.tenantIndexPollError(nil, errors.New("blerg")))

	// tenant index older than 1 minute is stale, error!
	assert.Error(t, p.tenantIndexPollError(&backend.TenantIndex{
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}, nil))

	// no error, tenant index is within 1 minute
	assert.NoError(t, p.tenantIndexPollError(&backend.TenantIndex{
		CreatedAt: time.Now().Add(-time.Second),
	}, nil))

	p = NewPoller(&PollerConfig{}, nil, nil, nil, nil, log.NewNopLogger())

	// no error, index is super old but stale tenant index is 0
	assert.NoError(t, p.tenantIndexPollError(&backend.TenantIndex{
		CreatedAt: time.Now().Add(30 * time.Hour),
	}, nil))
}

func TestBlockListBackendMetrics(t *testing.T) {
	tests := []struct {
		name                                 string
		list                                 PerTenant
		compactedList                        PerTenantCompacted
		testType                             string
		expectedBackendObjectsTotal          int
		expectedBackendBytesTotal            uint64
		expectedCompactedBackendObjectsTotal int
		expectedCompacteddBackendBytesTotal  uint64
	}{
		{
			name: "total backend objects calculation is correct",
			list: PerTenant{
				"test": []*backend.BlockMeta{
					{
						TotalObjects: 10,
					},
					{
						TotalObjects: 7,
					},
					{
						TotalObjects: 8,
					},
				},
			},
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							TotalObjects: 7,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							TotalObjects: 8,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							TotalObjects: 5,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							TotalObjects: 15,
						},
					},
				},
			},
			expectedBackendObjectsTotal:          25,
			expectedBackendBytesTotal:            0,
			expectedCompactedBackendObjectsTotal: 35,
			expectedCompacteddBackendBytesTotal:  0,
			testType:                             "backend objects",
		},
		{
			name: "total backend bytes calculation is correct",
			list: PerTenant{
				"test": []*backend.BlockMeta{
					{
						Size_: 250,
					},
					{
						Size_: 500,
					},
					{
						Size_: 250,
					},
				},
			},
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							Size_: 300,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							Size_: 200,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							Size_: 250,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							Size_: 500,
						},
					},
				},
			},
			expectedBackendObjectsTotal:          0,
			expectedBackendBytesTotal:            1000,
			expectedCompactedBackendObjectsTotal: 0,
			expectedCompacteddBackendBytesTotal:  1250,
			testType:                             "backend bytes",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newBlockList := tc.list["test"]
			newCompactedBlockList := tc.compactedList["test"]
			backendMetaMetrics := sumTotalBackendMetaMetrics(newBlockList, newCompactedBlockList)
			assert.Equal(t, tc.expectedBackendObjectsTotal, backendMetaMetrics.blockMetaTotalObjects)
			assert.Equal(t, tc.expectedCompactedBackendObjectsTotal, backendMetaMetrics.compactedBlockMetaTotalObjects)
			assert.Equal(t, tc.expectedBackendBytesTotal, backendMetaMetrics.blockMetaTotalBytes)
			assert.Equal(t, tc.expectedCompacteddBackendBytesTotal, backendMetaMetrics.compactedBlockMetaTotalBytes)
		})
	}
}

func TestPollTolerateConsecutiveErrors(t *testing.T) {
	var (
		c = newMockCompactor(PerTenantCompacted{}, false)
		w = &backend.MockWriter{}
		s = &mockJobSharder{owns: true}
		b = newBlocklist(PerTenant{}, PerTenantCompacted{})
	)

	testCases := []struct {
		name                    string
		tolerate                int
		tollerateTenantFailures int
		tenantErrors            map[string][]error
		expectedError           error
	}{
		{
			name:          "no errors",
			tolerate:      0,
			tenantErrors:  map[string][]error{"one": {}},
			expectedError: nil,
		},
		{
			name:                    "untolerated single error",
			tolerate:                0,
			tollerateTenantFailures: 0,
			tenantErrors:            map[string][]error{"one": {errors.New("tenant one error")}},
			expectedError:           errors.New("too many tenant failures; abandoning polling cycle"),
		},
		{
			name:                    "tolerated errors",
			tolerate:                2,
			tollerateTenantFailures: 1,
			tenantErrors: map[string][]error{
				"one": {
					errors.New("tenant one error"),
					errors.New("tenant one error"),
					nil,
				},
				"two": {
					errors.New("tenant two error"),
					errors.New("tenant two error"),
					nil,
				},
			},
			expectedError: nil,
		},
		{
			name:                    "too many errors",
			tolerate:                2,
			tollerateTenantFailures: 1,
			tenantErrors: map[string][]error{
				"one": {
					errors.New("tenant one error"),
					errors.New("tenant one error"),
					nil,
				},
				"two": {
					errors.New("tenant two error"),
					errors.New("tenant two error"),
					nil,
				},
				"three": {
					errors.New("tenant three error"),
					errors.New("tenant three error"),
					errors.New("tenant three error"),
				},
				"four": {
					errors.New("tenant four error"),
					errors.New("tenant four error"),
					errors.New("tenant four error"),
				},
			},
			expectedError: errors.New("too many tenant failures; abandoning polling cycle"), // test for tenant x err to avoid needing to care which of the last two tenants were caught
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callCounter := make(map[string]int)
			mtx := sync.Mutex{}

			// This mock reader returns error or nil based on the tenant ID
			r := &backend.MockReader{
				BlocksFn: func(_ context.Context, tenantID string) ([]uuid.UUID, []uuid.UUID, []uuid.UUID, error) {
					mtx.Lock()
					defer func() {
						callCounter[tenantID]++
						mtx.Unlock()
					}()

					// init the callCoutner
					if _, ok := callCounter[tenantID]; !ok {
						callCounter[tenantID] = 0
					}

					count := callCounter[tenantID]

					if errs, ok := tc.tenantErrors[tenantID]; ok {
						if len(errs) > count {
							return nil, nil, nil, errs[count]
						}
					}

					// i, _ := strconv.Atoi(tenantID)
					// return nil, nil, tc.tenantErrors[i]
					return nil, nil, nil, nil
				},
			}
			// Tenant ID for each index in the slice
			for t := range tc.tenantErrors {
				r.T = append(r.T, t)
			}

			poller := NewPoller(&PollerConfig{
				PollConcurrency:           testPollConcurrency,
				TenantPollConcurrency:     testTenantPollConcurrency,
				PollFallback:              testPollFallback,
				TenantIndexBuilders:       testBuilders,
				TolerateConsecutiveErrors: tc.tolerate,
				TolerateTenantFailures:    tc.tollerateTenantFailures,
				EmptyTenantDeletionAge:    testEmptyTenantIndexAge,
			}, s, r, c, w, log.NewLogfmtLogger(os.Stdout))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, _, _, err := poller.Do(ctx, b)

			if tc.expectedError != nil {
				assert.ErrorContains(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPollerDoSpanParenting verifies that the per-tenant spans started inside
// Poller.Do's goroutines are children of the Poller.Do span, so a trace
// viewer can navigate from the poll cycle down into individual tenant work.
// testTraceExporter and testTraceOnce back testSpans: OTel's global
// TracerProvider only delegates reliably to the first concrete provider ever
// installed via otel.SetTracerProvider in a process — a *Tracer obtained
// beforehand (as this package's package-level `tracer` var is) resolves once
// and keeps sending spans to that first provider even if a later test calls
// otel.SetTracerProvider again with a fresh one. So install exactly one
// provider for the whole test binary and reset its exporter per test instead
// of swapping providers.
var (
	testTraceExporter = tracetest.NewInMemoryExporter()
	testTraceOnce     sync.Once
)

// testSpans installs the shared tracer provider (once) and returns a fresh
// view of the spans recorded during this test.
func testSpans(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	testTraceOnce.Do(func() {
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(testTraceExporter))
		otel.SetTracerProvider(tp)
	})
	testTraceExporter.Reset()
	return testTraceExporter
}

func TestPollerDoSpanParenting(t *testing.T) {
	exporter := testSpans(t)

	var (
		c = newMockCompactor(PerTenantCompacted{}, false)
		w = &backend.MockWriter{}
		s = &mockJobSharder{owns: true}
		b = newBlocklist(PerTenant{}, PerTenantCompacted{})
		r = &backend.MockReader{T: []string{"test"}}
	)

	poller := NewPoller(&PollerConfig{
		PollConcurrency:        testPollConcurrency,
		TenantPollConcurrency:  testTenantPollConcurrency,
		PollFallback:           testPollFallback,
		TenantIndexBuilders:    testBuilders,
		EmptyTenantDeletionAge: testEmptyTenantIndexAge,
	}, s, r, c, w, log.NewNopLogger())

	_, _, _, err := poller.Do(context.Background(), b)
	require.NoError(t, err)

	spans := exporter.GetSpans()

	var rootSpan, tenantSpan *tracetest.SpanStub
	for i := range spans {
		switch spans[i].Name {
		case "Poller.Do":
			rootSpan = &spans[i]
		case "Poller.Do.func":
			tenantSpan = &spans[i]
		}
	}

	require.NotNil(t, rootSpan, "expected a Poller.Do span")
	require.NotNil(t, tenantSpan, "expected a Poller.Do.func span")

	assert.True(t, tenantSpan.Parent.IsValid(), "tenant span should have a valid parent span context")
	assert.Equal(t, rootSpan.SpanContext.TraceID(), tenantSpan.Parent.TraceID(), "tenant span should belong to the same trace as Poller.Do")
	assert.Equal(t, rootSpan.SpanContext.SpanID(), tenantSpan.Parent.SpanID(), "tenant span should be a direct child of Poller.Do")
}

// tenantsErrorReader wraps a MockReader to force Tenants() to fail, since
// MockReader itself has no override hook for that specific call.
type tenantsErrorReader struct {
	*backend.MockReader
}

func (r *tenantsErrorReader) Tenants(context.Context) ([]string, error) {
	return nil, errors.New("boom")
}

// TestPollerSpansRecordErrorStatus verifies that the spans created in the
// Poller.Do call chain are marked with the OTel span status matching whether
// that specific operation actually failed, rather than staying Unset
// regardless of outcome.
func TestPollerSpansRecordErrorStatus(t *testing.T) {
	spanStatus := func(t *testing.T, spans []tracetest.SpanStub, name string) codes.Code {
		t.Helper()
		for _, s := range spans {
			if s.Name == name {
				return s.Status.Code
			}
		}
		t.Fatalf("no span named %q found", name)
		return codes.Unset
	}

	t.Run("failing tenant marks its span chain errored without failing the whole poll", func(t *testing.T) {
		exporter := testSpans(t)

		var (
			c = newMockCompactor(PerTenantCompacted{}, false)
			w = &backend.MockWriter{}
			s = &mockJobSharder{owns: true}
			b = newBlocklist(PerTenant{}, PerTenantCompacted{})
			r = &backend.MockReader{
				T: []string{"test"},
				BlocksFn: func(context.Context, string) ([]uuid.UUID, []uuid.UUID, []uuid.UUID, error) {
					return nil, nil, nil, errors.New("boom")
				},
			}
		)

		poller := NewPoller(&PollerConfig{
			PollConcurrency:           testPollConcurrency,
			TenantPollConcurrency:     testTenantPollConcurrency,
			PollFallback:              testPollFallback,
			TenantIndexBuilders:       testBuilders,
			TolerateConsecutiveErrors: 0,
			TolerateTenantFailures:    1,
			EmptyTenantDeletionAge:    testEmptyTenantIndexAge,
		}, s, r, c, w, log.NewNopLogger())

		_, _, _, err := poller.Do(context.Background(), b)
		require.NoError(t, err, "one tolerated tenant failure should not fail the whole poll")

		spans := exporter.GetSpans()

		assert.Equal(t, codes.Ok, spanStatus(t, spans, "Poller.Do"), "Poller.Do itself succeeded overall")
		assert.Equal(t, codes.Error, spanStatus(t, spans, "Poller.Do.func"), "the failing tenant's span should be marked errored")
		assert.Equal(t, codes.Error, spanStatus(t, spans, "Poller.pollTenantAndCreateIndex"))
		assert.Equal(t, codes.Error, spanStatus(t, spans, "Poller.pollTenantBlocks"))
	})

	t.Run("fully successful poll marks every span Ok", func(t *testing.T) {
		exporter := testSpans(t)

		var (
			c = newMockCompactor(PerTenantCompacted{}, false)
			w = &backend.MockWriter{}
			s = &mockJobSharder{owns: true}
			b = newBlocklist(PerTenant{}, PerTenantCompacted{})
			r = &backend.MockReader{T: []string{"test"}}
		)

		poller := NewPoller(&PollerConfig{
			PollConcurrency:        testPollConcurrency,
			TenantPollConcurrency:  testTenantPollConcurrency,
			PollFallback:           testPollFallback,
			TenantIndexBuilders:    testBuilders,
			EmptyTenantDeletionAge: testEmptyTenantIndexAge,
		}, s, r, c, w, log.NewNopLogger())

		_, _, _, err := poller.Do(context.Background(), b)
		require.NoError(t, err)

		spans := exporter.GetSpans()
		for _, name := range []string{"Poller.Do", "Poller.Do.func", "Poller.pollTenantAndCreateIndex", "Poller.pollTenantBlocks", "pollUnknown"} {
			assert.Equal(t, codes.Ok, spanStatus(t, spans, name), "span %q should be explicitly marked Ok on success, not left Unset", name)
		}
	})

	t.Run("Tenants listing failure marks Poller.Do's own span errored", func(t *testing.T) {
		exporter := testSpans(t)

		var (
			c = newMockCompactor(PerTenantCompacted{}, false)
			w = &backend.MockWriter{}
			s = &mockJobSharder{owns: true}
			b = newBlocklist(PerTenant{}, PerTenantCompacted{})
			r = &tenantsErrorReader{MockReader: &backend.MockReader{}}
		)

		poller := NewPoller(&PollerConfig{
			PollConcurrency:        testPollConcurrency,
			TenantPollConcurrency:  testTenantPollConcurrency,
			PollFallback:           testPollFallback,
			TenantIndexBuilders:    testBuilders,
			EmptyTenantDeletionAge: testEmptyTenantIndexAge,
		}, s, r, c, w, log.NewNopLogger())

		_, _, _, err := poller.Do(context.Background(), b)
		require.Error(t, err)

		spans := exporter.GetSpans()
		assert.Equal(t, codes.Error, spanStatus(t, spans, "Poller.Do"))
	})
}

func TestPollComparePreviousResults(t *testing.T) {
	zero := backend.MustParse("00000000-0000-0000-0000-000000000000")
	aaa := backend.MustParse("00000000-0000-0000-0000-00000000000A")
	eff := backend.MustParse("00000000-0000-0000-0000-00000000000F")

	testCases := []struct {
		name string

		previousPerTenant          PerTenant
		previousCompactedPerTenant PerTenantCompacted

		currentPerTenant          PerTenant
		currentCompactedPerTenant PerTenantCompacted

		expectedPerTenant          PerTenant
		expectedCompactedPerTenant PerTenantCompacted

		expectedBlockMetaCalls          map[string]map[uuid.UUID]int
		expectedCompactedBlockMetaCalls map[string]map[uuid.UUID]int

		tollerateErrors         int
		tollerateTenantFailures int

		readerErr bool
		err       error
	}{
		{
			name:                       "with no previous results, the blocklist is polled",
			previousPerTenant:          PerTenant{},
			previousCompactedPerTenant: PerTenantCompacted{},
			currentPerTenant: PerTenant{
				"test": []*backend.BlockMeta{
					{BlockID: zero},
				},
			},
			currentCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			expectedPerTenant: PerTenant{
				"test": []*backend.BlockMeta{
					{BlockID: zero},
				},
			},
			expectedCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			expectedBlockMetaCalls: map[string]map[uuid.UUID]int{
				"test": {
					uuid.UUID(zero): 1,
				},
			},
			expectedCompactedBlockMetaCalls: map[string]map[uuid.UUID]int{
				"test": {
					uuid.UUID(eff): 1,
				},
			},
		},
		{
			name: "with previous results, meta should be read from only new blocks",
			previousPerTenant: PerTenant{
				"test": []*backend.BlockMeta{
					{BlockID: zero},
					{BlockID: eff},
				},
			},
			previousCompactedPerTenant: PerTenantCompacted{},
			currentPerTenant: PerTenant{
				"test": []*backend.BlockMeta{
					{BlockID: zero},
					{BlockID: eff},
				},
			},
			currentCompactedPerTenant: PerTenantCompacted{},
			expectedPerTenant: PerTenant{
				"test": []*backend.BlockMeta{
					{BlockID: zero},
					{BlockID: eff},
				},
			},
			expectedCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{},
			},
			expectedBlockMetaCalls: map[string]map[uuid.UUID]int{},
		},
		{
			name: "with previous results, blocks that have been compacted since the last poll should be known as compacted",
			previousPerTenant: PerTenant{
				"test": []*backend.BlockMeta{
					{BlockID: zero},
					{BlockID: aaa},
				},
			},
			previousCompactedPerTenant: PerTenantCompacted{},
			currentPerTenant: PerTenant{
				"test": []*backend.BlockMeta{
					{BlockID: eff},
				},
			},
			currentCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
				},
			},
			expectedPerTenant: PerTenant{
				"test": []*backend.BlockMeta{
					{BlockID: eff},
				},
			},
			expectedCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
				},
			},
			expectedBlockMetaCalls: map[string]map[uuid.UUID]int{
				"test": {
					uuid.UUID(eff): 1,
				},
			},
			// zero and aaa were previously known as live blocks, so their CompactedBlockMeta
			// is synthesized from cached data — no backend fetch required.
			expectedCompactedBlockMetaCalls: nil,
		},
		{
			name:              "with previous compactions should be known",
			previousPerTenant: PerTenant{},
			previousCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			currentPerTenant: PerTenant{},
			currentCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			expectedPerTenant: PerTenant{
				"test": []*backend.BlockMeta{},
			},
			expectedCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			expectedBlockMetaCalls: map[string]map[uuid.UUID]int{},
		},
		{
			name:              "with previous compactions removed, should be forgotten",
			previousPerTenant: PerTenant{},
			previousCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
				},
			},
			currentPerTenant:           PerTenant{},
			currentCompactedPerTenant:  PerTenantCompacted{},
			expectedPerTenant:          PerTenant{},
			expectedCompactedPerTenant: PerTenantCompacted{},
			expectedBlockMetaCalls:     map[string]map[uuid.UUID]int{},
		},
		{
			name:                    "previous results with read error should maintain previous results",
			tollerateErrors:         1, // Fail at the single tenant level
			tollerateTenantFailures: 2,
			readerErr:               true,
			previousPerTenant:       PerTenant{},
			previousCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			currentPerTenant: PerTenant{},
			currentCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			expectedPerTenant: PerTenant{
				"test": []*backend.BlockMeta{},
			},
			expectedCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			expectedBlockMetaCalls: map[string]map[uuid.UUID]int{},
		},
		{
			name:                    "previous results with read error should maintain previous results when tolerations are low and multiple tenants",
			tollerateErrors:         0, // Fail at the single tenant level
			tollerateTenantFailures: 2,
			readerErr:               true,
			previousPerTenant: PerTenant{
				"test2": []*backend.BlockMeta{
					{BlockID: zero},
					{BlockID: eff},
				},
			},
			previousCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			currentPerTenant: PerTenant{
				"test2": []*backend.BlockMeta{
					{BlockID: zero},
					{BlockID: eff},
				},
			},
			currentCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
			},
			expectedPerTenant: PerTenant{
				"test": []*backend.BlockMeta{},
				"test2": []*backend.BlockMeta{
					{BlockID: zero},
					{BlockID: eff},
				},
			},
			expectedCompactedPerTenant: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{BlockMeta: backend.BlockMeta{BlockID: zero}},
					{BlockMeta: backend.BlockMeta{BlockID: aaa}},
					{BlockMeta: backend.BlockMeta{BlockID: eff}},
				},
				"test2": []*backend.CompactedBlockMeta{},
			},
			expectedBlockMetaCalls: map[string]map[uuid.UUID]int{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				c           = newMockCompactor(tc.currentCompactedPerTenant, false)
				w           = &backend.MockWriter{}
				s           = &mockJobSharder{owns: true}
				r           = newMockReader(tc.currentPerTenant, tc.currentCompactedPerTenant, tc.readerErr)
				previous    = newBlocklist(tc.previousPerTenant, tc.previousCompactedPerTenant)
				ctx, cancel = context.WithCancel(context.Background())
			)
			defer cancel()

			// This mock reader returns error or nil based on the tenant ID
			poller := NewPoller(&PollerConfig{
				PollConcurrency:           testPollConcurrency,
				PollFallback:              testPollFallback,
				TenantIndexBuilders:       testBuilders,
				TenantPollConcurrency:     testTenantPollConcurrency,
				TolerateConsecutiveErrors: tc.tollerateErrors,
				TolerateTenantFailures:    tc.tollerateTenantFailures,
			}, s, r, c, w, log.NewNopLogger())

			metas, compactedMetas, _, err := poller.Do(ctx, previous)
			require.Equal(t, tc.err, err)

			require.Equal(t, len(tc.expectedPerTenant), len(metas))
			for tenantID, expectedMetas := range tc.expectedPerTenant {
				l := metas[tenantID]
				sort.Slice(l, func(i, j int) bool {
					x := bytes.Compare(l[i].BlockID[:], l[j].BlockID[:])
					return x > 0
				})

				sort.Slice(expectedMetas, func(i, j int) bool {
					x := bytes.Compare(expectedMetas[i].BlockID[:], expectedMetas[j].BlockID[:])
					return x > 0
				})

				require.Equal(t, expectedMetas, l)
			}

			require.Equal(t, len(tc.expectedCompactedPerTenant), len(compactedMetas))
			for tenantID, expectedCompactedMetas := range tc.expectedCompactedPerTenant {
				l := compactedMetas[tenantID]
				sort.Slice(l, func(i, j int) bool {
					x := bytes.Compare(l[i].BlockID[:], l[j].BlockID[:])
					return x > 0
				})

				sort.Slice(expectedCompactedMetas, func(i, j int) bool {
					x := bytes.Compare(expectedCompactedMetas[i].BlockID[:], expectedCompactedMetas[j].BlockID[:])
					return x > 0
				})

				// Strip CompactedTime before comparing: blocks synthesized from previously-known
				// live blocks use time.Now(), which we cannot predict in expected values.
				stripped := make([]*backend.CompactedBlockMeta, len(l))
				for i, m := range l {
					stripped[i] = &backend.CompactedBlockMeta{BlockMeta: m.BlockMeta}
				}
				require.Equal(t, expectedCompactedMetas, stripped)
			}

			require.Equal(t, tc.expectedBlockMetaCalls, r.(*backend.MockReader).BlockMetaCalls)
			require.Equal(t, tc.expectedCompactedBlockMetaCalls, c.(*backend.MockCompactor).CompactedBlockMetaCalls)
		})
	}
}

// TestPollLiveToCompactedSynthesized verifies that blocks previously known as live are
// synthesized into CompactedBlockMeta without a backend fetch, and that CompactedTime is set.
func TestPollLiveToCompactedSynthesized(t *testing.T) {
	blockID := backend.MustParse("00000000-0000-0000-0000-000000000001")
	tenantID := "test"

	previousMeta := &backend.BlockMeta{
		BlockID:  blockID,
		TenantID: tenantID,
	}

	// Previous poll saw the block as live; current poll reports it as compacted.
	previous := newBlocklist(
		PerTenant{tenantID: []*backend.BlockMeta{previousMeta}},
		PerTenantCompacted{},
	)
	currentCompacted := PerTenantCompacted{
		tenantID: []*backend.CompactedBlockMeta{
			{BlockMeta: backend.BlockMeta{BlockID: blockID}},
		},
	}

	c := newMockCompactor(currentCompacted, false)
	r := newMockReader(PerTenant{}, currentCompacted, false)
	w := &backend.MockWriter{}
	s := &mockJobSharder{owns: true}

	poller := NewPoller(&PollerConfig{
		PollConcurrency:       testPollConcurrency,
		TenantPollConcurrency: testTenantPollConcurrency,
		PollFallback:          testPollFallback,
		TenantIndexBuilders:   testBuilders,
	}, s, r, c, w, log.NewNopLogger())

	before := time.Now()
	_, compactedMetas, _, err := poller.Do(context.Background(), previous)
	require.NoError(t, err)

	// Block should appear in the compacted list.
	require.Len(t, compactedMetas[tenantID], 1)
	got := compactedMetas[tenantID][0]
	require.Equal(t, blockID, got.BlockID)
	require.Equal(t, *previousMeta, got.BlockMeta)

	// CompactedTime must be set (non-zero and after our start time).
	require.False(t, got.CompactedTime.IsZero(), "CompactedTime should be set for synthesized compacted block")
	require.True(t, got.CompactedTime.After(before) || got.CompactedTime.Equal(before))

	// No backend fetch should have occurred for this block.
	require.Empty(t, c.(*backend.MockCompactor).CompactedBlockMetaCalls)
}

func BenchmarkPoller10k(b *testing.B) {
	tests := []struct {
		tenantCount     int
		blocksPerTenant int
	}{
		{
			tenantCount:     1,
			blocksPerTenant: 100,
		},
		{
			tenantCount:     1,
			blocksPerTenant: 1000,
		},
		{
			tenantCount:     1,
			blocksPerTenant: 10000,
		},
		{
			tenantCount:     1,
			blocksPerTenant: 100000,
		},
	}

	for _, tc := range tests {
		previousPerTenant := newPerTenant(tc.tenantCount, tc.blocksPerTenant)
		previousPerTenantCompacted := newPerTenantCompacted(tc.tenantCount, tc.blocksPerTenant)

		// currentPerTenant := newPerTenant(uuids, tc.tenantCount, tc.blocksPerTenant)
		// currentPerTenantCompacted := newPerTenantCompacted(uuids, tc.tenantCount, tc.blocksPerTenant)
		currentPerTenant := maps.Clone(previousPerTenant)
		currentPerTenantCompacted := maps.Clone(previousPerTenantCompacted)

		var (
			c        = newMockCompactor(currentPerTenantCompacted, false)
			w        = &backend.MockWriter{}
			s        = &mockJobSharder{owns: true}
			r        = newMockReader(currentPerTenant, currentPerTenantCompacted, false)
			previous = newBlocklist(previousPerTenant, previousPerTenantCompacted)
		)

		// This mock reader returns error or nil based on the tenant ID
		poller := NewPoller(&PollerConfig{
			PollConcurrency:       testPollConcurrency,
			TenantPollConcurrency: testTenantPollConcurrency,
			PollFallback:          testPollFallback,
			TenantIndexBuilders:   testBuilders,
		}, s, r, c, w, log.NewNopLogger())

		runName := fmt.Sprintf("%d-%d", tc.tenantCount, tc.blocksPerTenant)
		b.Run(runName, func(b *testing.B) {
			for tenant := range previousPerTenant {
				benchmarkPollTenant(b, poller, tenant, previous)
			}
		})
	}
}

func BenchmarkFullPoller(b *testing.B) {
	cases := []struct {
		name            string
		tenants         int
		blocksPerTenant int
		iterations      int
		blocksPer       int
		compactionsPer  int
	}{
		{
			name: "no tenants",
		},
		{
			name:            "single tenant",
			tenants:         1,
			blocksPerTenant: 1000,
		},
		{
			name:            "multi tenant",
			tenants:         10,
			blocksPerTenant: 1000,
		},
		{
			name:            "multi tenant growth",
			tenants:         10,
			blocksPerTenant: 1000,
			iterations:      10,
			blocksPer:       100,
		},
		{
			name:            "multi tenant growth and compactions",
			tenants:         10,
			blocksPerTenant: 1000,
			iterations:      10,
			blocksPer:       100,
			compactionsPer:  10,
		},
	}

	for _, bc := range cases {
		b.Run(fmt.Sprintf("%sTenants%dBlocks%dGrow%dCompactions%d", bc.name, bc.tenants, bc.blocksPerTenant, bc.blocksPer, bc.compactionsPer), func(b *testing.B) {
			s := &mockJobSharder{owns: true}

			d := b.TempDir()
			defer os.RemoveAll(d)

			rr, ww, cc, err := local.New(&local.Config{
				Path: d,
			})
			require.NoError(b, err)

			var (
				ctx = context.Background()
				r   = backend.NewReader(rr)
				w   = backend.NewWriter(ww)
				// c   = backend.NewCompactor(cc)
			)

			poller := NewPoller(&PollerConfig{
				PollConcurrency:       testPollConcurrency,
				TenantPollConcurrency: testTenantPollConcurrency,
				PollFallback:          testPollFallback,
				TenantIndexBuilders:   testBuilders,
			}, s, r, cc, w, log.NewNopLogger())

			// Create the tenants and push the initial blocks to them.
			tenants := make([]string, bc.tenants)
			for i := 0; i < bc.tenants; i++ {
				tenant := fmt.Sprintf("tenant-%d", i)
				tenants = append(tenants, tenant)
				writeNewBlocksForTenant(ctx, b, w, tenant, bc.blocksPerTenant)
			}

			var (
				ml   = PerTenant{}
				cl   = PerTenantCompacted{}
				list = New()
			)

			b.Run("initial", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ml, cl, _, _ = poller.Do(ctx, list)
				}
				b.StopTimer()

				list.ApplyPollResults(ml, cl, nil)
			})

			// No change to the list
			b.Run("second", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ml, cl, _, _ = poller.Do(ctx, list)
				}
				b.StopTimer()

				list.ApplyPollResults(ml, cl, nil)
			})

			for i := 0; i < bc.iterations; i++ {
				// push more blocks to the tenants
				for _, tenant := range tenants {
					writeNewBlocksForTenant(ctx, b, w, tenant, bc.blocksPer)
				}

				// Compact some blocks
				for tenant, blocks := range ml {
					blocksToCompact := blocks[:bc.compactionsPer]
					for _, block := range blocksToCompact {
						err := cc.MarkBlockCompacted(uuid.UUID(block.BlockID), tenant)
						require.NoError(b, err)
					}
				}

				b.Run(fmt.Sprintf("grow%d", i), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						ml, cl, _, _ = poller.Do(ctx, list)
					}
					b.StopTimer()

					list.ApplyPollResults(ml, cl, nil)
				})
			}
		})
	}
}

func benchmarkPollTenant(b *testing.B, poller *Poller, tenant string, previous *List) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, _, err := poller.pollTenantBlocks(context.Background(), tenant, previous)
		require.NoError(b, err)
	}
}

func writeNewBlocksForTenant(ctx context.Context, b *testing.B, w backend.Writer, tenant string, count int) {
	var (
		err   error
		metas = newBlockMetas(count, tenant)
	)

	for _, m := range metas {
		err = w.WriteBlockMeta(ctx, m)
		require.NoError(b, err)
	}
}

func newBlockMetas(count int, tenantID string) []*backend.BlockMeta {
	metas := make([]*backend.BlockMeta, count)
	for i := 0; i < count; i++ {
		metas[i] = &backend.BlockMeta{
			BlockID:  backend.NewUUID(),
			TenantID: tenantID,
		}
	}

	return metas
}

func newCompactedMetas(count int) []*backend.CompactedBlockMeta {
	metas := make([]*backend.CompactedBlockMeta, count)
	for i := 0; i < count; i++ {
		metas[i] = &backend.CompactedBlockMeta{
			BlockMeta: backend.BlockMeta{
				BlockID: backend.NewUUID(),
			},
		}
	}

	return metas
}

var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func newPerTenant(tenantCount, blockCount int) PerTenant {
	var (
		perTenant = make(PerTenant, tenantCount)
		metas     []*backend.BlockMeta
		id        string
		tenant    string
	)
	for i := 0; i < tenantCount; i++ {
		tenant = fmt.Sprintf("tenant-%d", i)
		metas = newBlockMetas(blockCount, tenant)
		id = randString(5)
		perTenant[id] = metas
	}

	return perTenant
}

func newPerTenantCompacted(tenantCount, blockCount int) PerTenantCompacted {
	perTenantCompacted := make(PerTenantCompacted)
	var metas []*backend.CompactedBlockMeta
	var id string
	for i := 0; i < tenantCount; i++ {
		metas = newCompactedMetas(blockCount)
		id = randString(5)
		perTenantCompacted[id] = metas
	}

	return perTenantCompacted
}

func newMockCompactor(list PerTenantCompacted, expectsError bool) backend.Compactor {
	return &backend.MockCompactor{
		BlockMetaFn: func(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
			if expectsError {
				return nil, errors.New("err")
			}

			l, ok := list[tenantID]
			if !ok {
				return nil, backend.ErrDoesNotExist
			}

			for _, m := range l {
				if uuid.UUID(m.BlockID) == blockID {
					return m, nil
				}
			}

			return nil, backend.ErrDoesNotExist
		},
	}
}

func newMockReader(list PerTenant, compactedList PerTenantCompacted, expectsError bool) backend.Reader {
	tenants := []string{}
	ttt := make(map[string]bool)

	for t := range list {
		ttt[t] = true
	}
	for t := range compactedList {
		ttt[t] = true
	}

	for k := range ttt {
		tenants = append(tenants, k)
	}

	return &backend.MockReader{
		T: tenants,
		BlocksFn: func(_ context.Context, tenantID string) ([]uuid.UUID, []uuid.UUID, []uuid.UUID, error) {
			if expectsError {
				return nil, nil, nil, errors.New("err")
			}
			blocks := list[tenantID]
			uuids := []uuid.UUID{}
			compactedUUIDs := []uuid.UUID{}
			for _, b := range blocks {
				uuids = append(uuids, uuid.UUID(b.BlockID))
			}
			compactedBlocks := compactedList[tenantID]
			for _, b := range compactedBlocks {
				compactedUUIDs = append(compactedUUIDs, uuid.UUID(b.BlockID))
			}

			return uuids, compactedUUIDs, nil, nil
		},
		BlockMetaCalls: make(map[string]map[uuid.UUID]int),
		BlockMetaFn: func(_ context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
			if expectsError {
				return nil, errors.New("err")
			}

			l, ok := list[tenantID]
			if !ok {
				return nil, backend.ErrDoesNotExist
			}

			for _, m := range l {
				if uuid.UUID(m.BlockID) == blockID {
					return m, nil
				}
			}

			return nil, backend.ErrDoesNotExist
		},
	}
}

func newBlocklist(metas PerTenant, compactedMetas PerTenantCompacted) *List {
	l := New()

	l.ApplyPollResults(metas, compactedMetas, nil)

	return l
}
