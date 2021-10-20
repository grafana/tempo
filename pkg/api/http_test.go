package api

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"
	"github.com/stretchr/testify/assert"
)

// For licensing reasons these strings exist in two packages. This test exists to make sure they don't
// drift.
func TestEquality(t *testing.T) {
	assert.Equal(t, HeaderAccept, tempo.AcceptHeaderKey)
	assert.Equal(t, HeaderAcceptProtobuf, tempo.ProtobufTypeHeaderValue)
}

func TestParseBackendSearch(t *testing.T) {
	tests := []struct {
		start         int64
		end           int64
		limit         int
		expectedLimit int
		expectedError error
	}{
		{
			expectedError: errors.New("please provide non-zero values for http parameters start and end"),
		},
		{
			start:         10,
			expectedError: errors.New("please provide non-zero values for http parameters start and end"),
		},
		{
			end:           10,
			expectedError: errors.New("please provide non-zero values for http parameters start and end"),
		},
		{
			start:         15,
			end:           10,
			expectedError: errors.New("http parameter start must be before end. received start=15 end=10"),
		},
		{
			start:         10,
			end:           100000,
			expectedError: errors.New("range specified by start and end exceeds 1800 seconds. received start=10 end=100000"),
		},
		{
			start:         10,
			end:           20,
			expectedLimit: 20,
		},
		{
			start:         10,
			end:           20,
			limit:         30,
			expectedLimit: 30,
		},
	}

	for _, tc := range tests {
		url := "/blerg?"
		if tc.start != 0 {
			url += fmt.Sprintf("&start=%d", tc.start)
		}
		if tc.end != 0 {
			url += fmt.Sprintf("&end=%d", tc.end)
		}
		if tc.limit != 0 {
			url += fmt.Sprintf("&limit=%d", tc.limit)
		}
		r := httptest.NewRequest("GET", url, nil)

		actualStart, actualEnd, actualLimit, actualError := ParseBackendSearch(r)

		if tc.expectedError != nil {
			assert.Equal(t, tc.expectedError, actualError)
			continue
		}
		assert.NoError(t, actualError)
		assert.Equal(t, tc.start, actualStart)
		assert.Equal(t, tc.end, actualEnd)
		assert.Equal(t, tc.expectedLimit, actualLimit)
	}
}
