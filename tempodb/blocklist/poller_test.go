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

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
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
			actualList, actualCompactedList, err := poller.Do(ctx, b)

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
			_, _, err := poller.Do(ctx, b)

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
			actualMeta, actualCompactedMeta, err := poller.pollBlock(context.Background(), tc.pollTenantID, (uuid.UUID)(tc.pollBlockID), false)

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

func TestPollBlockWithNoCompactFlag(t *testing.T) {
	blockID := backend.MustParse("00000000-0000-0000-0000-000000000001")
	blockUUID := uuid.MustParse(blockID.String())
	tenantID := "test"

	tests := []struct {
		name                   string
		hasNoCompactFlag       bool
		noCompactFlagError     error
		skipNoCompactBlocks    bool
		expectedMeta           *backend.BlockMeta
		expectedNoCompactCalls int
		expectedBlockMetaCalls int
		wantErr                bool
	}{
		{
			name:                   "block without nocompact flag is included",
			hasNoCompactFlag:       false,
			skipNoCompactBlocks:    true,
			expectedMeta:           &backend.BlockMeta{BlockID: blockID, TenantID: tenantID},
			expectedNoCompactCalls: 1,
			expectedBlockMetaCalls: 1,
			wantErr:                false,
		},
		{
			name:                   "block with nocompact flag is excluded",
			hasNoCompactFlag:       true,
			skipNoCompactBlocks:    true,
			expectedMeta:           nil,
			expectedNoCompactCalls: 1,
			expectedBlockMetaCalls: 0, // no calls for excluded block
			wantErr:                false,
		},
		{
			name:                   "block with nocompact flag is included if skipNoCompactBlocks is false",
			hasNoCompactFlag:       true,
			skipNoCompactBlocks:    false,
			expectedMeta:           &backend.BlockMeta{BlockID: blockID, TenantID: tenantID},
			expectedNoCompactCalls: 0, // no compact check calls
			expectedBlockMetaCalls: 1,
			wantErr:                false,
		},
		{
			name:                   "block with nocompact flag check error is excluded",
			hasNoCompactFlag:       false,
			skipNoCompactBlocks:    true,
			noCompactFlagError:     errors.New("flag check error"),
			expectedMeta:           nil,
			expectedNoCompactCalls: 1,
			expectedBlockMetaCalls: 0, // no calls for errored block
			wantErr:                true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &backend.MockReader{
				T: []string{tenantID},
				BlockMetaFn: func(_ context.Context, blockID uuid.UUID, tID string) (*backend.BlockMeta, error) {
					if blockID == blockUUID && tID == tenantID {
						return &backend.BlockMeta{BlockID: backend.UUID(blockID), TenantID: tID}, nil
					}
					return nil, backend.ErrDoesNotExist
				},
				HasNoCompactFlagFn: func(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
					if tc.noCompactFlagError != nil {
						return false, tc.noCompactFlagError
					}
					return tc.hasNoCompactFlag, nil
				},
			}

			c := &backend.MockCompactor{
				BlockMetaFn: func(_ uuid.UUID, _ string) (*backend.CompactedBlockMeta, error) {
					return nil, backend.ErrDoesNotExist
				},
			}

			w := &backend.MockWriter{}

			poller := NewPoller(&PollerConfig{
				PollConcurrency:       testPollConcurrency,
				TenantPollConcurrency: testTenantPollConcurrency,
				PollFallback:          testPollFallback,
				TenantIndexBuilders:   testBuilders,
				SkipNoCompactBlocks:   tc.skipNoCompactBlocks,
			}, &mockJobSharder{}, r, c, w, log.NewNopLogger())

			actualMeta, actualCompactedMeta, err := poller.pollBlock(context.Background(), tenantID, blockUUID, false)
			if tc.wantErr {
				assert.Error(t, err, "expected error for block with nocompact flag or error checking the flag")
			} else {
				assert.NoError(t, err)
			}

			assert.Nil(t, actualCompactedMeta)

			assert.Equal(t, tc.expectedMeta, actualMeta, "block without nocompact flag should be included")

			// Verify the methods were called the expected number of times
			assert.Equal(t, tc.expectedBlockMetaCalls, r.BlockMetaCalls[tenantID][blockUUID], "BlockMeta should be called expected number of times")
			assert.Equal(t, tc.expectedNoCompactCalls, r.HasNoCompactFlagCalls[tenantID][blockUUID], "HasNoCompactFlag should be called expected number of times")
		})
	}
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
				BlocksFn: func(_ context.Context, tenantID string) ([]uuid.UUID, []uuid.UUID, error) {
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
							return nil, nil, errs[count]
						}
					}

					// i, _ := strconv.Atoi(tenantID)
					// return nil, nil, tc.tenantErrors[i]
					return nil, nil, nil
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
			_, _, err := poller.Do(ctx, b)

			if tc.expectedError != nil {
				assert.ErrorContains(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
					(uuid.UUID)(zero): 1,
				},
			},
			expectedCompactedBlockMetaCalls: map[string]map[uuid.UUID]int{
				"test": {
					(uuid.UUID)(eff): 1,
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
					(uuid.UUID)(eff): 1,
				},
			},
			expectedCompactedBlockMetaCalls: map[string]map[uuid.UUID]int{
				"test": {
					(uuid.UUID)(aaa):  1,
					(uuid.UUID)(zero): 1,
				},
			},
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

			metas, compactedMetas, err := poller.Do(ctx, previous)
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
				require.Equal(t, expectedCompactedMetas, l)
			}

			require.Equal(t, tc.expectedBlockMetaCalls, r.(*backend.MockReader).BlockMetaCalls)
			require.Equal(t, tc.expectedCompactedBlockMetaCalls, c.(*backend.MockCompactor).CompactedBlockMetaCalls)
		})
	}
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
					ml, cl, _ = poller.Do(ctx, list)
				}
				b.StopTimer()

				list.ApplyPollResults(ml, cl)
			})

			// No change to the list
			b.Run("second", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ml, cl, _ = poller.Do(ctx, list)
				}
				b.StopTimer()

				list.ApplyPollResults(ml, cl)
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
						ml, cl, _ = poller.Do(ctx, list)
					}
					b.StopTimer()

					list.ApplyPollResults(ml, cl)
				})
			}
		})
	}
}

func benchmarkPollTenant(b *testing.B, poller *Poller, tenant string, previous *List) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, err := poller.pollTenantBlocks(context.Background(), tenant, previous)
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
				if (uuid.UUID)(m.BlockID) == blockID {
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
		BlocksFn: func(_ context.Context, tenantID string) ([]uuid.UUID, []uuid.UUID, error) {
			if expectsError {
				return nil, nil, errors.New("err")
			}
			blocks := list[tenantID]
			uuids := []uuid.UUID{}
			compactedUUIDs := []uuid.UUID{}
			for _, b := range blocks {
				uuids = append(uuids, (uuid.UUID)(b.BlockID))
			}
			compactedBlocks := compactedList[tenantID]
			for _, b := range compactedBlocks {
				compactedUUIDs = append(compactedUUIDs, (uuid.UUID)(b.BlockID))
			}

			return uuids, compactedUUIDs, nil
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
				if (uuid.UUID)(m.BlockID) == blockID {
					return m, nil
				}
			}

			return nil, backend.ErrDoesNotExist
		},
	}
}

func newBlocklist(metas PerTenant, compactedMetas PerTenantCompacted) *List {
	l := New()

	l.ApplyPollResults(metas, compactedMetas)

	return l
}
