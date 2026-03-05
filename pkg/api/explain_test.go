package api

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseExplainRequest(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantQ     string
		wantStart int64
		wantEnd   int64
		wantErr   bool
	}{
		{
			name:    "missing q param",
			query:   "",
			wantErr: true,
		},
		{
			name:    "only q param",
			query:   "?q={status=error}",
			wantQ:   "{status=error}",
			wantErr: false,
		},
		{
			name:      "q with start and end",
			query:     "?q={status=error}&start=1000&end=2000",
			wantQ:     "{status=error}",
			wantStart: 1000,
			wantEnd:   2000,
			wantErr:   false,
		},
		{
			name:    "invalid start",
			query:   "?q={status=error}&start=notanumber",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{URL: mustParseURL("http://localhost" + tc.query)}
			q, start, end, err := ParseExplainRequest(r)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantQ, q)
			require.Equal(t, tc.wantStart, start)
			require.Equal(t, tc.wantEnd, end)
		})
	}
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
