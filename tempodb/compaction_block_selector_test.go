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
		// {
		// 	name:      "nil - nil",
		// 	blocklist: nil,
		// 	expected:  nil,
		// },
		// {
		// 	name:      "empty - nil",
		// 	blocklist: []*backend.BlockMeta{},
		// 	expected:  nil,
		// },
		// {
		// 	name: "two blocks returned",
		// 	blocklist: []*backend.BlockMeta{
		// 		&backend.BlockMeta{
		// 			BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		// 		},
		// 	},
		// 	expected: []*backend.BlockMeta{
		// 		&backend.BlockMeta{
		// 			BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		// 		},
		// 	},
		// },
		// {
		// 	name: "three blocks choose smallest two",
		// 	blocklist: []*backend.BlockMeta{
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		// 			TotalObjects: 0,
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		// 			TotalObjects: 1,
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		// 			TotalObjects: 0,
		// 		},
		// 	},
		// 	expected: []*backend.BlockMeta{
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		// 			TotalObjects: 0,
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		// 			TotalObjects: 0,
		// 		},
		// 	},
		// },
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
		// {
		// 	name: "two iterations of four blocks across two windows",
		// 	blocklist: []*backend.BlockMeta{
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		// 			TotalObjects: 0,
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		// 			TotalObjects: 1,
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		// 			TotalObjects: 0,
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		// 			TotalObjects: 1,
		// 		},
		// 	},
		// 	expected: []*backend.BlockMeta{
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		// 			TotalObjects: 0,
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		// 			TotalObjects: 0,
		// 		},
		// 	},
		// 	expectedSecond: []*backend.BlockMeta{
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		// 			TotalObjects: 1,
		// 		},
		// 		&backend.BlockMeta{
		// 			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		// 			TotalObjects: 1,
		// 		},
		// 	},
		// },
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
