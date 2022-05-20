package traceql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	tests := []struct {
		in string
	}{
		{in: "max(duration) > 3s | { status = error || .http.status = 500 }"},
		{in: "{ .http.status = 200 } | max(.field) - min(.field) > 3"},
		{in: "({ .http.status = 200 } | count()) + ({ name = `foo` } | avg(duration)) = 2"},
		{in: "{ (-(3 + 2) * .test - parent.blerg + duration)^3 }"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			pass1, err := Parse(tc.in)
			require.NoError(t, err)

			// now parse it a second time and confirm that it parses the same way twice
			pass2, err := Parse(pass1.String())
			ok := assert.NoError(t, err)
			if !ok {
				t.Logf("\n\t1: %s", pass1.String())
				return
			}

			ok = assert.Equal(t, pass1, pass2)
			t.Logf("\n\t1: %s\n\t2: %s", pass1.String(), pass2.String())
		})
	}
}

func stripWhitespace(s string) string {
	return strings.Join(strings.Fields(s), "")
}
