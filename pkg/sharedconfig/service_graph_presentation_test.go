package sharedconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Saved defaults and per-request rules share count, byte, text, and regex limits.
// Boundaries count UTF-8 bytes rather than characters and combine both lists.
func TestValidateServiceGraphPresentation(t *testing.T) {
	for _, tc := range []struct {
		name               string
		services, patterns []string
		invalid            bool
	}{
		{name: "empty"},
		{name: "exact rule limit", services: strings.Fields(strings.Repeat("gateway ", 128))},
		{name: "combined rule limit exceeded", services: strings.Fields(strings.Repeat("gateway ", 128)), patterns: []string{"proxy"}, invalid: true},
		{name: "exact byte limit", services: []string{strings.Repeat("é", 32768)}},
		{name: "combined byte limit exceeded", services: []string{strings.Repeat("é", 32768)}, patterns: []string{"x"}, invalid: true},
		{name: "empty string", services: []string{""}, invalid: true},
		{name: "invalid utf8", patterns: []string{string([]byte{255})}, invalid: true},
		{name: "invalid regex", patterns: []string{"["}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateServiceGraphPresentation(tc.services, tc.patterns)
			if tc.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
