package traceql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jpe extend
func TestStringer(t *testing.T) {
	tests := []struct {
		in string
	}{
		{in: "max(duration) > 3s | { status = error || .http.status = 500 }"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			actual, err := Parse(tc.in)
			require.NoError(t, err)
			assert.Equal(t,
				stripWhitespace(tc.in),
				stripWhitespace(actual.String()),
			)
			t.Log(stripWhitespace(tc.in))
		})
	}
}

func stripWhitespace(s string) string {
	return strings.Join(strings.Fields(s), "")
}
