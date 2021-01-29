package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBlockPage(t *testing.T) {
	tt := []struct {
		inBuff           []byte
		expectedBuff     []byte
		expectedEncoding Encoding
		expectedError    bool
	}{
		{
			expectedError: true,
		},
		{
			inBuff:        []byte{},
			expectedError: true,
		},
		{
			inBuff:        []byte{byte(maxEncoding + 1)},
			expectedError: true,
		},
		{
			inBuff:           []byte{byte(EncGZIP), 0x01, 0x02},
			expectedEncoding: EncGZIP,
			expectedBuff:     []byte{0x01, 0x02},
		},
	}

	for _, tc := range tt {
		page, err := NewBlockPage(tc.inBuff)

		if tc.expectedError {
			assert.Error(t, err)
			continue
		}

		assert.NotNil(t, page)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedBuff, page.encodedBytes)
		assert.Equal(t, tc.expectedEncoding, page.encoding)
	}
}
