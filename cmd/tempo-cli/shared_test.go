package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "RFC3339 in UTC",
			input:    "2024-03-15T10:30:00Z",
			expected: time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "RFC3339 with timezone offset",
			input:    "2024-03-15T10:30:00+05:30",
			expected: time.Date(2024, 3, 15, 10, 30, 0, 0, time.FixedZone("", 5*3600+30*60)),
		},
		{
			name:     "now",
			input:    "now",
			expected: now,
		},
		{
			name:     "now-1h",
			input:    "now-1h",
			expected: now.Add(-1 * time.Hour),
		},
		{
			name:     "now-30m",
			input:    "now-30m",
			expected: now.Add(-30 * time.Minute),
		},
		{
			name:     "now-3h30m",
			input:    "now-3h30m",
			expected: now.Add(-3*time.Hour - 30*time.Minute),
		},
		{
			name:     "now-7d",
			input:    "now-7d",
			expected: now.Add(-7 * 24 * time.Hour),
		},
		{
			name:     "now-1y",
			input:    "now-1y",
			expected: now.Add(-365 * 24 * time.Hour),
		},
		{
			name:    "format without timezone is rejected",
			input:   "2024-03-15T10:30:00",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "yesterday",
			wantErr: true,
		},
		{
			name:    "invalid relative",
			input:   "now+1h",
			wantErr: true,
		},
		{
			name:    "invalid duration",
			input:   "now-abc",
			wantErr: true,
		},
		{
			name:     "leading spaces are trimmed",
			input:    "  now-1h",
			expected: now.Add(-1 * time.Hour),
		},
		{
			name:     "trailing spaces are trimmed",
			input:    "now-1h  ",
			expected: now.Add(-1 * time.Hour),
		},
		{
			name:     "now with leading space",
			input:    " now",
			expected: now,
		},
		{
			name:     "now with trailing spaces",
			input:    "now   ",
			expected: now,
		},
		{
			name:     "RFC3339 with surrounding spaces",
			input:    " 2024-03-15T10:30:00Z ",
			expected: time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTime(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.WithinDuration(t, tt.expected, result, 2*time.Second)
		})
	}
}
