package v0

import (
	"bytes"
	"context"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
)

func TestDataReader(t *testing.T) {

	tests := []struct {
		readerBytes   []byte
		records       []*common.Record
		expectedBytes [][]byte
		expectedError bool
	}{
		{},
		{
			records: []*common.Record{
				{
					Start:  0,
					Length: 1,
				},
			},
			expectedError: true,
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []*common.Record{
				{
					Start:  0,
					Length: 1,
				},
			},
			expectedBytes: [][]byte{
				{0x01},
			},
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []*common.Record{
				{
					Start:  1,
					Length: 1,
				},
			},
			expectedBytes: [][]byte{
				{0x02},
			},
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []*common.Record{
				{
					Start:  0,
					Length: 1,
				},
				{
					Start:  1,
					Length: 1,
				},
			},
			expectedBytes: [][]byte{
				{0x01},
				{0x02},
			},
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []*common.Record{
				{
					Start:  0,
					Length: 5,
				},
			},
			expectedError: true,
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []*common.Record{
				{
					Start:  5,
					Length: 5,
				},
			},
			expectedError: true,
		},
		{
			readerBytes: []byte{0x01, 0x02, 0x03},
			records: []*common.Record{
				{
					Start:  1,
					Length: 1,
				},
				{
					Start:  2,
					Length: 1,
				},
			},
			expectedBytes: [][]byte{
				{0x02},
				{0x03},
			},
		},
		{
			readerBytes: []byte{0x01, 0x02, 0x03},
			records: []*common.Record{
				{
					Start:  0,
					Length: 1,
				},
				{
					Start:  2,
					Length: 1,
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		reader := NewDataReader(backend.NewContextReaderWithAllReader(bytes.NewReader(tc.readerBytes)))
		actual, err := reader.Read(context.Background(), tc.records)
		reader.Close()

		if tc.expectedError {
			assert.Error(t, err)
			continue
		}

		assert.NoError(t, err)
		assert.Equal(t, tc.expectedBytes, actual)
	}
}
