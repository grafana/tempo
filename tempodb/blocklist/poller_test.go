package blocklist

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

var (
	testPollConcurrency = uint(10)
	testPollFallback    = true
	testBuilders        = 1
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

			poller := NewPoller(&PollerConfig{
				PollConcurrency:     testPollConcurrency,
				PollFallback:        testPollFallback,
				TenantIndexBuilders: testBuilders,
			}, &mockJobSharder{
				owns: true,
			}, r, c, w, log.NewNopLogger())
			actualList, actualCompactedList, err := poller.Do()

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

			r.(*backend.MockReader).TenantIndexFn = func(ctx context.Context, tenantID string) (*backend.TenantIndex, error) {
				if tc.errorOnCreateTenantIndex {
					return nil, errors.New("err")
				}
				return &backend.TenantIndex{
					CreatedAt: time.Now().Add(-5 * time.Minute), // always make the tenant index 5 minutes old so the above tests can use that for fallback testing
				}, nil
			}

			poller := NewPoller(&PollerConfig{
				PollConcurrency:     testPollConcurrency,
				PollFallback:        tc.pollFallback,
				TenantIndexBuilders: testBuilders,
				StaleTenantIndex:    tc.staleTenantIndex,
			}, &mockJobSharder{
				owns: tc.isTenantIndexBuilder,
			}, r, c, w, log.NewNopLogger())
			_, _, err := poller.Do()

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
				PollConcurrency:     testPollConcurrency,
				PollFallback:        testPollFallback,
				TenantIndexBuilders: testBuilders,
			}, &mockJobSharder{}, r, c, w, log.NewNopLogger())
			actualMeta, actualCompactedMeta, err := poller.pollBlock(context.Background(), tc.pollTenantID, tc.pollBlockID)

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
	for t := range list {
		tenants = append(tenants, t)
	}
	for t := range compactedList {
		tenants = append(tenants, t)
	}

	return &backend.MockReader{
		T: tenants,
		BlockFn: func(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
			if expectsError {
				return nil, errors.New("err")
			}
			blocks := list[tenantID]
			uuids := []uuid.UUID{}
			for _, b := range blocks {
				uuids = append(uuids, b.BlockID)
			}
			compactedBlocks := compactedList[tenantID]
			for _, b := range compactedBlocks {
				uuids = append(uuids, b.BlockID)
			}

			return uuids, nil
		},
		BlockMetaFn: func(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
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
