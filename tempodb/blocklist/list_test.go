package blocklist

import (
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

const testTenantID = "test"

func TestApplyPollResults(t *testing.T) {
	tests := []struct {
		name            string
		metas           PerTenant
		compacted       PerTenantCompacted
		expectedTenants []string
	}{
		{
			name:            "all nil",
			expectedTenants: []string{},
		},
		{
			name: "meta only",
			metas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
				"test2": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			expectedTenants: []string{"test", "test2"},
		},
		{
			name: "compacted meta only",
			compacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
				},
				"test2": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						},
					},
				},
			},
			expectedTenants: []string{},
		},
		{
			name: "all",
			metas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
				"blerg": []*backend.BlockMeta{
					{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			compacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
				},
				"test2": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						},
					},
				},
			},
			expectedTenants: []string{"blerg", "test"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := New()
			l.ApplyPollResults(tc.metas, tc.compacted)

			actualTenants := l.Tenants()
			sort.Slice(actualTenants, func(i, j int) bool { return actualTenants[i] < actualTenants[j] })
			assert.Equal(t, tc.expectedTenants, actualTenants)
			for tenant, expected := range tc.metas {
				actual := l.Metas(tenant)
				assert.Equal(t, expected, actual)
			}
			for tenant, expected := range tc.compacted {
				actual := l.CompactedMetas(tenant)
				assert.Equal(t, expected, actual)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name     string
		existing []*backend.BlockMeta
		add      []*backend.BlockMeta
		remove   []*backend.BlockMeta
		expected []*backend.BlockMeta
	}{
		{
			name:     "all nil",
			existing: nil,
			add:      nil,
			remove:   nil,
			expected: nil,
		},
		{
			name:     "add to nil",
			existing: nil,
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name: "add to existing",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name:     "remove from nil",
			existing: nil,
			add:      nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: nil,
		},
		{
			name: "remove nil",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add:    nil,
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name: "remove existing",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add: nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name: "remove no match",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			add: nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name: "add and remove",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
			},
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()

			l.metas[testTenantID] = tt.existing
			l.Update(testTenantID, tt.add, tt.remove, nil)

			assert.Equal(t, len(tt.expected), len(l.metas[testTenantID]))

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].BlockID, l.metas[testTenantID][i].BlockID)
			}
		})
	}
}

func TestUpdateCompacted(t *testing.T) {
	tests := []struct {
		name     string
		existing []*backend.CompactedBlockMeta
		add      []*backend.CompactedBlockMeta
		expected []*backend.CompactedBlockMeta
	}{
		{
			name:     "all nil",
			existing: nil,
			add:      nil,
			expected: nil,
		},
		{
			name:     "add to nil",
			existing: nil,
			add: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			expected: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
		},
		{
			name: "add to existing",
			existing: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			add: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					},
				},
			},
			expected: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()

			l.compactedMetas[testTenantID] = tt.existing
			l.Update(testTenantID, nil, nil, tt.add)

			assert.Equal(t, len(tt.expected), len(l.compactedMetas[testTenantID]))

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].BlockID, l.compactedMetas[testTenantID][i].BlockID)
			}
		})
	}
}
