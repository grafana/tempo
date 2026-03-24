package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestApplyHeadersHTTP(t *testing.T) {
	tests := []struct {
		name            string
		headers         []string
		expectedHeaders map[string]string
	}{
		{
			name:    "single header",
			headers: []string{"X-TOKEN=my-secret"},
			expectedHeaders: map[string]string{
				"X-Token": "my-secret",
			},
		},
		{
			name:    "multiple headers",
			headers: []string{"X-TOKEN=my-secret", "X-Custom=value"},
			expectedHeaders: map[string]string{
				"X-Token":  "my-secret",
				"X-Custom": "value",
			},
		},
		{
			name:            "malformed header without equals is ignored",
			headers:         []string{"no-equals-sign"},
			expectedHeaders: map[string]string{},
		},
		{
			name:    "value containing equals",
			headers: []string{"Authorization=Bearer token=abc"},
			expectedHeaders: map[string]string{
				"Authorization": "Bearer token=abc",
			},
		},
		{
			name:            "empty input",
			headers:         []string{},
			expectedHeaders: map[string]string{},
		},
		{
			name:            "empty key is ignored",
			headers:         []string{"=value"},
			expectedHeaders: map[string]string{},
		},
		{
			name:    "whitespace around key is trimmed",
			headers: []string{" X-TOKEN =my-secret"},
			expectedHeaders: map[string]string{
				"X-Token": "my-secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://localhost:3200/api/search", nil)
			applyHeadersHTTP(req, tt.headers)

			require.Len(t, req.Header, len(tt.expectedHeaders))
			for k, v := range tt.expectedHeaders {
				require.Equal(t, v, req.Header.Get(k))
			}
		})
	}
}

func TestApplyHeadersGRPC(t *testing.T) {
	ctx := applyHeadersGRPC(context.Background(), []string{"X-TOKEN=my-secret", "X-Custom=value"})

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)
	require.Equal(t, []string{"my-secret"}, md.Get("X-TOKEN"))
	require.Equal(t, []string{"value"}, md.Get("X-Custom"))
}

func TestApplyHeadersGRPC_lastWins(t *testing.T) {
	ctx := applyHeadersGRPC(context.Background(), []string{"X-TOKEN=first", "X-TOKEN=second"})

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)
	require.Equal(t, []string{"second"}, md.Get("X-TOKEN"))
}

func TestApplyHeadersGRPC_empty(t *testing.T) {
	ctx := context.Background()
	result := applyHeadersGRPC(ctx, []string{})
	_, ok := metadata.FromOutgoingContext(result)
	require.False(t, ok)
}

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
