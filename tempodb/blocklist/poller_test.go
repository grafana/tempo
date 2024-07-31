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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
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
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			expectedList: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					},
				},
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					},
				},
			},
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
				},
			},
			expectedList: PerTenant{
				"test2": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					},
				},
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					},
				},
			},
			expectedCompactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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
			actualList, actualCompactedList, err := poller.Do(b)

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

			r.(*backend.MockReader).TenantIndexFn = func(_ context.Context, tenantID string) (*backend.TenantIndex, error) {
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
			_, _, err := poller.Do(b)

			assert.Equal(t, tc.expectsError, err != nil)
			assert.Equal(t, tc.expectsTenantIndexWritten, w.IndexCompactedMeta != nil)
			assert.Equal(t, tc.expectsTenantIndexWritten, w.IndexMeta != nil)
		})
	}
}

func TestPollBlock(t *testing.T) {
	tests := []struct {
		name                  string
		list                  PerTenant
		compactedList         PerTenantCompacted
		pollTenantID          string
		pollBlockID           uuid.UUID
		expectedMeta          *backend.BlockMeta
		expectedCompactedMeta *backend.CompactedBlockMeta
		expectsError          bool
	}{
		{
			name:         "block and tenant don't exist",
			pollTenantID: "test",
			pollBlockID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		},
		{
			name:         "block exists",
			pollTenantID: "test",
			pollBlockID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			list: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			expectedMeta: &backend.BlockMeta{
				BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		},
		{
			name:         "compactedblock exists",
			pollTenantID: "test",
			pollBlockID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
				},
			},
			expectedCompactedMeta: &backend.CompactedBlockMeta{
				BlockMeta: backend.BlockMeta{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name:         "errors",
			pollTenantID: "test",
			pollBlockID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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
			actualMeta, actualCompactedMeta, err := poller.pollBlock(context.Background(), tc.pollTenantID, tc.pollBlockID, false)

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
						Size: 250,
					},
					{
						Size: 500,
					},
					{
						Size: 250,
					},
				},
			},
			compactedList: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							Size: 300,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							Size: 200,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							Size: 250,
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							Size: 500,
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
			expectedError:           errors.New("tenant one err"),
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
			expectedError: errors.New("tenant four err"),
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

			_, _, err := poller.Do(b)

			if tc.expectedError != nil {
				assert.ErrorContains(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPollComparePreviousResults(t *testing.T) {
	zero := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	aaa := uuid.MustParse("00000000-0000-0000-0000-00000000000A")
	eff := uuid.MustParse("00000000-0000-0000-0000-00000000000F")

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
					zero: 1,
				},
			},
			expectedCompactedBlockMetaCalls: map[string]map[uuid.UUID]int{
				"test": {
					eff: 1,
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
					eff: 1,
				},
			},
			expectedCompactedBlockMetaCalls: map[string]map[uuid.UUID]int{
				"test": {
					aaa:  1,
					zero: 1,
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
				c        = newMockCompactor(tc.currentCompactedPerTenant, false)
				w        = &backend.MockWriter{}
				s        = &mockJobSharder{owns: true}
				r        = newMockReader(tc.currentPerTenant, tc.currentCompactedPerTenant, tc.readerErr)
				previous = newBlocklist(tc.previousPerTenant, tc.previousCompactedPerTenant)
			)

			// This mock reader returns error or nil based on the tenant ID
			poller := NewPoller(&PollerConfig{
				PollConcurrency:           testPollConcurrency,
				PollFallback:              testPollFallback,
				TenantIndexBuilders:       testBuilders,
				TenantPollConcurrency:     testTenantPollConcurrency,
				TolerateConsecutiveErrors: tc.tollerateErrors,
				TolerateTenantFailures:    tc.tollerateTenantFailures,
			}, s, r, c, w, log.NewNopLogger())

			metas, compactedMetas, err := poller.Do(previous)
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

func benchmarkPollTenant(b *testing.B, poller *Poller, tenant string, previous *List) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, err := poller.pollTenantBlocks(context.Background(), tenant, previous)
		require.NoError(b, err)
	}
}

func newBlockMetas(count int) []*backend.BlockMeta {
	metas := make([]*backend.BlockMeta, count)
	for i := 0; i < count; i++ {
		metas[i] = &backend.BlockMeta{
			BlockID: uuid.New(),
		}
	}

	return metas
}

func newCompactedMetas(count int) []*backend.CompactedBlockMeta {
	metas := make([]*backend.CompactedBlockMeta, count)
	for i := 0; i < count; i++ {
		metas[i] = &backend.CompactedBlockMeta{
			BlockMeta: backend.BlockMeta{
				BlockID: uuid.New(),
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
	perTenant := make(PerTenant, tenantCount)
	var metas []*backend.BlockMeta
	var id string
	for i := 0; i < tenantCount; i++ {
		metas = newBlockMetas(blockCount)
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
				if m.BlockID == blockID {
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
				uuids = append(uuids, b.BlockID)
			}
			compactedBlocks := compactedList[tenantID]
			for _, b := range compactedBlocks {
				compactedUUIDs = append(compactedUUIDs, b.BlockID)
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
				if m.BlockID == blockID {
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
