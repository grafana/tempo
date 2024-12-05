package blockbuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshallCommitMeta(t *testing.T) {
	tests := []struct {
		name         string
		commitRecTs  int64
		expectedMeta string
	}{
		{"ValidTimestamp", 1627846261, "1,1627846261"},
		{"ZeroTimestamp", 0, "1,0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := marshallCommitMeta(tc.commitRecTs)
			assert.Equal(t, tc.expectedMeta, meta, "expected: %s, got: %s", tc.expectedMeta, meta)
		})
	}
}

func TestUnmarshallCommitMeta(t *testing.T) {
	tests := []struct {
		name          string
		meta          string
		expectedTs    int64
		expectedError bool
	}{
		{"ValidMeta", "1,1627846261", 1627846261, false},
		{"InvalidMetaFormat", "1,invalid", 0, true},
		{"UnsupportedVersion", "2,1627846261", 0, true},
		{"EmptyMeta", "", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, err := unmarshallCommitMeta(tc.meta)
			assert.Equal(t, tc.expectedError, err != nil, "expected error: %v, got: %v", tc.expectedError, err)
			assert.Equal(t, tc.expectedTs, ts, "expected: %d, got: %d", tc.expectedTs, ts)
		})
	}
}
