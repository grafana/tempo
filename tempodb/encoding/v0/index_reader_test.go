package v0

import (
	"testing"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexReader(t *testing.T) {
	record1 := &common.Record{
		ID:     []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Start:  0,
		Length: 1,
	}
	record2 := &common.Record{
		ID:     []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Start:  1,
		Length: 2,
	}
	record3 := &common.Record{
		ID:     []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Start:  2,
		Length: 3,
	}

	recordBytes, err := marshalRecords([]*common.Record{record1, record2, record3})
	require.NoError(t, err)

	tests := []struct {
		recordBytes       []byte
		expectedError     bool
		at                int
		expectedAt        *common.Record
		find              common.ID
		expectedFind      *common.Record
		expectedFindIndex int
	}{
		{
			recordBytes:       []byte{},
			expectedFindIndex: -1,
		},
		{
			recordBytes:   []byte{0x01},
			expectedError: true,
		},
		{
			recordBytes:       []byte{},
			at:                12,
			expectedFindIndex: -1,
		},
		{
			recordBytes:  recordBytes,
			at:           0,
			expectedAt:   record1,
			find:         []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedFind: record1,
		},
		{
			recordBytes:       recordBytes,
			at:                1,
			expectedAt:        record2,
			find:              []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedFind:      record2,
			expectedFindIndex: 1,
		},
		{
			recordBytes:       recordBytes,
			at:                2,
			expectedAt:        record3,
			find:              []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedFind:      record3,
			expectedFindIndex: 2,
		},
	}

	for _, tc := range tests {
		reader, err := NewIndexReader(tc.recordBytes)
		if tc.expectedError {
			assert.Error(t, err)
			continue
		}

		assert.Equal(t, tc.expectedAt, reader.At(tc.at))
		actualFind, actualIndex := reader.Find(tc.find)
		assert.Equal(t, tc.expectedFind, actualFind)
		assert.Equal(t, tc.expectedFindIndex, actualIndex)
	}
}
