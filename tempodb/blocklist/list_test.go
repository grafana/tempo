package blocklist

import (
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backend_v1 "github.com/grafana/tempo/tempodb/backend/v1"
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
				"test": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
				"test2": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
			},
			expectedTenants: []string{"test", "test2"},
		},
		{
			name: "compacted meta only",
			compacted: PerTenantCompacted{
				"test": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
						},
					},
				},
				"test2": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
						},
					},
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
						},
					},
				},
			},
			expectedTenants: []string{},
		},
		{
			name: "all",
			metas: PerTenant{
				"test": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
				"blerg": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
			},
			compacted: PerTenantCompacted{
				"test": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
						},
					},
				},
				"test2": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
						},
					},
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
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
		existing []*backend_v1.BlockMeta
		add      []*backend_v1.BlockMeta
		remove   []*backend_v1.BlockMeta
		expected []*backend_v1.BlockMeta
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
			add: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
			},
			remove: nil,
			expected: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
			},
		},
		{
			name: "add to existing",
			existing: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
			},
			add: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
			remove: nil,
			expected: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
		},
		{
			name:     "remove from nil",
			existing: nil,
			add:      nil,
			remove: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
			expected: nil,
		},
		{
			name: "remove nil",
			existing: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
			add:    nil,
			remove: nil,
			expected: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
		},
		{
			name: "remove existing",
			existing: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
			add: nil,
			remove: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
			},
			expected: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
		},
		{
			name: "remove no match",
			existing: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
			},
			add: nil,
			remove: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
			expected: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
			},
		},
		{
			name: "add and remove",
			existing: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
			add: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
				},
			},
			remove: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
			expected: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
				},
			},
		},
		{
			name: "add already exists",
			existing: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
			},
			add: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
			remove: nil,
			expected: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				},
				{
					BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()

			l.metas[testTenantID] = tt.existing
			l.Update(testTenantID, tt.add, tt.remove, nil, nil)

			assert.Equal(t, len(tt.expected), len(l.metas[testTenantID]))

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].BlockId, l.metas[testTenantID][i].BlockId)
			}
		})
	}
}

func TestUpdateCompacted(t *testing.T) {
	tests := []struct {
		name     string
		existing []*backend_v1.CompactedBlockMeta
		add      []*backend_v1.CompactedBlockMeta
		remove   []*backend_v1.CompactedBlockMeta
		expected []*backend_v1.CompactedBlockMeta
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
			add: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
			},
			expected: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
			},
		},
		{
			name: "add to existing",
			existing: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
			},
			add: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					},
				},
			},
			expected: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					},
				},
			},
		},
		{
			name: "add already exists",
			existing: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
			},
			add: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					},
				},
			},
			expected: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					},
				},
			},
		},
		{
			name: "add and remove",
			existing: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					},
				},
			},
			add: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
					},
				},
			},
			remove: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					},
				},
			},
			expected: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					},
				},
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()

			l.compactedMetas[testTenantID] = tt.existing
			l.Update(testTenantID, nil, nil, tt.add, tt.remove)

			assert.Equal(t, len(tt.expected), len(l.compactedMetas[testTenantID]))

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].BlockId, l.compactedMetas[testTenantID][i].BlockId)
			}
		})
	}
}

func TestUpdatesSaved(t *testing.T) {
	// unlike most tests these are applied serially to the same list object and the expected
	// results are cumulative across all tests

	one := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	two := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	oneOhOne := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	oneOhTwo := uuid.MustParse("10000000-0000-0000-0000-000000000002")

	tests := []struct {
		applyMetas     PerTenant
		applyCompacted PerTenantCompacted
		updateTenant   string
		addMetas       []*backend_v1.BlockMeta
		removeMetas    []*backend_v1.BlockMeta
		addCompacted   []*backend_v1.CompactedBlockMeta

		expectedTenants   []string
		expectedMetas     PerTenant
		expectedCompacted PerTenantCompacted
	}{
		// STEP 1: apply a normal polling data and updates
		{
			applyMetas: PerTenant{
				"test": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(one),
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(oneOhOne),
						},
					},
				},
			},
			updateTenant: "test",
			addMetas: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(one),
				},
				{
					BlockId: uuidBytes(two),
				},
			},
			removeMetas: []*backend_v1.BlockMeta{
				{
					BlockId: uuidBytes(one),
				},
			},
			addCompacted: []*backend_v1.CompactedBlockMeta{
				{
					BlockMeta: backend_v1.BlockMeta{
						BlockId: uuidBytes(oneOhTwo),
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(two),
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(oneOhOne),
						},
					},
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(oneOhTwo),
						},
					},
				},
			},
		},
		// STEP 2: same polling apply, no update! but expect the same results
		{
			applyMetas: PerTenant{
				"test": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(one),
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(oneOhOne),
						},
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend_v1.BlockMeta{
					// Even though we have just appled one, it was removed in the previous step, and we we expect not to find it here.
					// {
					// 	BlockId: one,
					// },
					{
						BlockId: uuidBytes(two),
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(oneOhOne),
						},
					},
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(oneOhTwo),
						},
					},
				},
			},
		},
		// STEP 3: same polling apply, no update! but this time the update doesn't impact results
		{
			applyMetas: PerTenant{
				"test": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(one),
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(oneOhOne),
						},
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend_v1.BlockMeta{
					{
						BlockId: uuidBytes(one),
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend_v1.CompactedBlockMeta{
					{
						BlockMeta: backend_v1.BlockMeta{
							BlockId: uuidBytes(oneOhOne),
						},
					},
				},
			},
		},
	}

	l := New()
	for i, tc := range tests {
		t.Logf("step %d", i+1)

		l.ApplyPollResults(tc.applyMetas, tc.applyCompacted)
		if tc.updateTenant != "" {
			l.Update(tc.updateTenant, tc.addMetas, tc.removeMetas, tc.addCompacted, nil)
		}

		actualTenants := l.Tenants()
		actualMetas := l.metas
		actualCompacted := l.compactedMetas

		sort.Slice(actualTenants, func(i, j int) bool { return actualTenants[i] < actualTenants[j] })
		assert.Equal(t, tc.expectedTenants, actualTenants)
		assert.Equal(t, tc.expectedMetas, actualMetas)

		require.Equal(t, len(tc.expectedCompacted), len(actualCompacted), "expectedCompacted: %+v, actualCompacted: %+v", tc.expectedCompacted, actualCompacted)
		assert.Equal(t, tc.expectedCompacted, actualCompacted)
	}
}
