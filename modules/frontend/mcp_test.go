package frontend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimDocs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "content with cutoff marker",
			input:    "header content\n<!-- mcp-cutoff -->actual content\nmore content",
			expected: "actual content\nmore content",
		},
		{
			name:     "content without cutoff marker",
			input:    "no cutoff marker here\njust regular content",
			expected: "no cutoff marker here\njust regular content",
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "only cutoff marker",
			input:    "<!-- mcp-cutoff -->",
			expected: "",
		},
		{
			name:     "cutoff marker at beginning",
			input:    "<!-- mcp-cutoff -->remaining content",
			expected: "remaining content",
		},
		{
			name:     "cutoff marker at end",
			input:    "some content<!-- mcp-cutoff -->",
			expected: "",
		},
		{
			name:     "multiple cutoff markers",
			input:    "first<!-- mcp-cutoff -->second<!-- mcp-cutoff -->third",
			expected: "second<!-- mcp-cutoff -->third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimDocs(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
