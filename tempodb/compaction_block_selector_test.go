package tempodb

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

func TestTimeWindowBlockSelector(t *testing.T) {
	tests := []struct {
		name           string
		blocklist      []*backend.BlockMeta
		expected       []*backend.BlockMeta
		expectedSecond []*backend.BlockMeta
	}{
		{
			name:      "nil - nil",
			blocklist: nil,
			expected:  nil,
		},
		{
			name:      "empty - nil",
			blocklist: []*backend.BlockMeta{},
			expected:  nil,
		},
		{
			name: "two blocks returned",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				&backend.BlockMeta{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				&backend.BlockMeta{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name: "three blocks choose smallest two",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
				},
			},
		},
		{
			name: "three blocks across two windows",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
					StartTime:    time.Unix(1, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					StartTime:    time.Unix(1, 0),
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					StartTime:    time.Unix(1, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
					StartTime:    time.Unix(1, 0),
				},
			},
		},
		{
			name: "two iterations of four blocks across one window",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects: 1,
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
				},
			},
			expectedSecond: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects: 1,
				},
			},
		},
		{
			name: "two iterations of four blocks across two windows",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					StartTime:    time.Unix(1, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects: 1,
					StartTime:    time.Unix(1, 0),
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
				},
			},
			expectedSecond: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					StartTime:    time.Unix(1, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects: 1,
					StartTime:    time.Unix(1, 0),
				},
			},
		},
		{
			name: "two iterations of six blocks across two windows",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					TotalObjects: 1,
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
					StartTime:    time.Unix(1, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
					StartTime:    time.Unix(1, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					TotalObjects: 1,
					StartTime:    time.Unix(2, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					StartTime:    time.Unix(3, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects: 1,
					StartTime:    time.Unix(3, 0),
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					TotalObjects: 0,
					StartTime:    time.Unix(1, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					TotalObjects: 1,
					StartTime:    time.Unix(1, 0),
				},
			},
			expectedSecond: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					TotalObjects: 0,
					StartTime:    time.Unix(3, 0),
				},
				&backend.BlockMeta{
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					TotalObjects: 1,
					StartTime:    time.Unix(3, 0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := newTimeWindowBlockSelector(tt.blocklist, time.Second, 100)

			actual, _ := selector.BlocksToCompact()
			assert.Equal(t, tt.expected, actual)

			actual, _ = selector.BlocksToCompact()
			assert.Equal(t, tt.expectedSecond, actual)
		})
	}
}

func TestTimeWindowBlockSelectorActiveWindow(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		blocklist      []*backend.BlockMeta
		expected       []*backend.BlockMeta
		expectedSecond []*backend.BlockMeta
	}{
		{
			name:      "nil - nil",
			blocklist: nil,
			expected:  nil,
		},
		{
			name:      "empty - nil",
			blocklist: []*backend.BlockMeta{},
			expected:  nil,
		},
		{
			name: "two blocks returned",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					StartTime: now,
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					StartTime: now,
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					StartTime: now,
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					StartTime: now,
				},
			},
		},
		{
			name: "three blocks choose smallest two",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 0,
					StartTime:       now,
				},
				&backend.BlockMeta{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					CompactionLevel: 1,
					StartTime:       now,
				},
				&backend.BlockMeta{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CompactionLevel: 0,
					StartTime:       now,
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 0,
					StartTime:       now,
				},
				&backend.BlockMeta{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CompactionLevel: 0,
					StartTime:       now,
				},
			},
		},
		{
			name: "three blocks choose none",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CompactionLevel: 0,
					StartTime:       now,
				},
				&backend.BlockMeta{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					CompactionLevel: 1,
					StartTime:       now,
				},
				&backend.BlockMeta{
					BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CompactionLevel: 2,
					StartTime:       now,
				},
			},
			expected: nil,
		},
		{
			name: "four blocks across two time windows",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					StartTime: now,
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					StartTime: now,
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					StartTime: now.Add(-24 * time.Hour),
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					StartTime: now.Add(-24 * time.Hour),
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					StartTime: now,
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					StartTime: now,
				},
			},
		},
		{
			name: "four blocks across two time windows.  skip buffer",
			blocklist: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					StartTime: now,
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					StartTime: now,
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					StartTime: now.Add(-24 * time.Hour),
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					StartTime: now.Add(-24 * time.Hour),
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					StartTime: now.Add(-48 * time.Hour),
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000005"),
					StartTime: now.Add(-48 * time.Hour),
				},
			},
			expected: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					StartTime: now,
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					StartTime: now,
				},
			},
			expectedSecond: []*backend.BlockMeta{
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000004"),
					StartTime: now.Add(-48 * time.Hour),
				},
				&backend.BlockMeta{
					BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000005"),
					StartTime: now.Add(-48 * time.Hour),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := newTimeWindowBlockSelector(tt.blocklist, 24*time.Hour, 100)

			actual, _ := selector.BlocksToCompact()
			assert.Equal(t, tt.expected, actual)

			actual, _ = selector.BlocksToCompact()
			assert.Equal(t, tt.expectedSecond, actual)
		})
	}
}
