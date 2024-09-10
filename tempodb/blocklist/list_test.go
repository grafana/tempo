package blocklist

import (
	"sort"
	"testing"

	google_uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/uuid"
	"github.com/grafana/tempo/tempodb/backend"
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
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
				"test2": []*backend.BlockMeta{
					{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
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
							BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
						},
					},
				},
				"test2": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
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
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
				"blerg": []*backend.BlockMeta{
					{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
			},
			compacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
						},
					},
				},
				"test2": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
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
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
			},
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
			},
		},
		{
			name: "add to existing",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
			},
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
		},
		{
			name:     "remove from nil",
			existing: nil,
			add:      nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
			expected: nil,
		},
		{
			name: "remove nil",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
			add:    nil,
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
		},
		{
			name: "remove existing",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
			add: nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
		},
		{
			name: "remove no match",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
			},
			add: nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
			},
		},
		{
			name: "add and remove",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000003")},
				},
			},
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000003")},
				},
			},
		},
		{
			name: "add already exists",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
			},
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
				},
			},
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				},
				{
					BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
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
		remove   []*backend.CompactedBlockMeta
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
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
			},
			expected: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
			},
		},
		{
			name: "add to existing",
			existing: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
			},
			add: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
					},
				},
			},
			expected: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
					},
				},
			},
		},
		{
			name: "add already exists",
			existing: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
			},
			add: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
					},
				},
			},
			expected: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
					},
				},
			},
		},
		{
			name: "add and remove",
			existing: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
					},
				},
			},
			add: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000003")},
					},
				},
			},
			remove: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000002")},
					},
				},
			},
			expected: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000001")},
					},
				},
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: google_uuid.MustParse("00000000-0000-0000-0000-000000000003")},
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
				assert.Equal(t, tt.expected[i].BlockID, l.compactedMetas[testTenantID][i].BlockID)
			}
		})
	}
}

func TestUpdatesSaved(t *testing.T) {
	// unlike most tests these are applied serially to the same list object and the expected
	// results are cumulative across all tests

	one := google_uuid.MustParse("00000000-0000-0000-0000-000000000001")
	two := google_uuid.MustParse("00000000-0000-0000-0000-000000000002")
	oneOhOne := google_uuid.MustParse("10000000-0000-0000-0000-000000000001")
	oneOhTwo := google_uuid.MustParse("10000000-0000-0000-0000-000000000002")

	tests := []struct {
		applyMetas     PerTenant
		applyCompacted PerTenantCompacted
		updateTenant   string
		addMetas       []*backend.BlockMeta
		removeMetas    []*backend.BlockMeta
		addCompacted   []*backend.CompactedBlockMeta

		expectedTenants   []string
		expectedMetas     PerTenant
		expectedCompacted PerTenantCompacted
	}{
		// STEP 1: apply a normal polling data and updates
		{
			applyMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.UUID{UUID: one},
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: oneOhOne},
						},
					},
				},
			},
			updateTenant: "test",
			addMetas: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: one},
				},
				{
					BlockID: uuid.UUID{UUID: two},
				},
			},
			removeMetas: []*backend.BlockMeta{
				{
					BlockID: uuid.UUID{UUID: one},
				},
			},
			addCompacted: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.UUID{UUID: oneOhTwo},
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.UUID{UUID: two},
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: oneOhOne},
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: oneOhTwo},
						},
					},
				},
			},
		},
		// STEP 2: same polling apply, no update! but expect the same results
		{
			applyMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.UUID{UUID: one},
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: oneOhOne},
						},
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend.BlockMeta{
					// Even though we have just appled one, it was removed in the previous step, and we we expect not to find it here.
					// {
					// 	BlockID: one,
					// },
					{
						BlockID: uuid.UUID{UUID: two},
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: oneOhOne},
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: oneOhTwo},
						},
					},
				},
			},
		},
		// STEP 3: same polling apply, no update! but this time the update doesn't impact results
		{
			applyMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.UUID{UUID: one},
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: oneOhOne},
						},
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: uuid.UUID{UUID: one},
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: uuid.UUID{UUID: oneOhOne},
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
