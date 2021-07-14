package blocklist

import (
	"context"
	"errors"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

var (
	testPollConcurrency = uint(10)
	testPollFallback    = true
)

type mockPollerSharder struct{}

func (m *mockPollerSharder) BuildTenantIndex() bool { return true }

// jpe extend
func TestDo(t *testing.T) {
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

			poller := NewPoller(testPollConcurrency, testPollFallback, &mockPollerSharder{}, r, c, w, log.NewNopLogger())
			actualList, actualCompactedList, err := poller.Do()

			assert.Equal(t, tc.expectedList, actualList)
			assert.Equal(t, tc.expectedCompactedList, actualCompactedList)
			if tc.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
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

			poller := NewPoller(testPollConcurrency, testPollFallback, &mockPollerSharder{}, r, c, w, log.NewNopLogger())
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
