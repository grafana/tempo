package pipeline

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdjustStartEndWare(t *testing.T) {
	now := time.Now()

	makeRequest := func(params map[string]string) *http.Request {
		req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/api/v2/traces/123345", nil)
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
		return req
	}

	// Helper: assert start/end fields
	assertStartEnd := func(t *testing.T, r Request, nanos, expectSinceCleared bool) {
		vals := r.HTTPRequest().URL.Query()
		start, end := vals.Get("start"), vals.Get("end")

		assert.NotEmpty(t, start, "start parameter should not be empty")
		assert.NotEmpty(t, end, "end parameter should not be empty")

		parse := func(s string) time.Time {
			if nanos {
				assert.Len(t, s, 19)
				n, err := strconv.ParseInt(s, 10, 64)
				require.NoError(t, err)
				return time.Unix(0, n)
			}
			assert.Len(t, s, 10)
			sec, err := strconv.ParseInt(s, 10, 64)
			require.NoError(t, err)
			return time.Unix(sec, 0)
		}

		startTime, endTime := parse(start), parse(end)
		assert.True(t, startTime.Before(endTime), "start should be before end")

		if expectSinceCleared {
			assert.Empty(t, vals.Get("since"), "since parameter should be cleared")
		}
	}

	defaultStart := 1 * time.Minute
	defaultBuffer := 1 * time.Second

	tests := []struct {
		name        string
		sendNanos   bool
		params      map[string]string
		expectError bool
	}{
		{"no params - nanos", true, nil, false},
		{"no params - seconds", false, nil, false},
		{"with since param", true, map[string]string{"since": "15m"}, false},
		{"valid start/end - nanos", true, map[string]string{
			"start": strconv.FormatInt(now.Add(-10*time.Minute).UnixNano(), 10),
			"end":   strconv.FormatInt(now.Add(-2*time.Minute).UnixNano(), 10),
		}, false},
		{"valid start/end - seconds", false, map[string]string{
			"start": strconv.FormatInt(now.Add(-10*time.Minute).Unix(), 10),
			"end":   strconv.FormatInt(now.Add(-2*time.Minute).Unix(), 10),
		}, false},
		{"clears since even with start/end", false, map[string]string{
			"since": "10m",
			"start": strconv.FormatInt(now.Add(-5*time.Minute).Unix(), 10),
			"end":   strconv.FormatInt(now.Add(-1*time.Minute).Unix(), 10),
		}, false},
		{"only start - error", true, map[string]string{"start": "1234567890"}, true},
		{"only end - error", true, map[string]string{"end": "1234567890"}, true},
		{"invalid since - error", true, map[string]string{"since": "invalid-duration"}, true},
		// {"mismatched start/end - error", true, map[string]string{
		// 	"start": "1234567890",          // 10 digits
		// 	"end":   "1234567890123456789", // 19 digits
		// }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rt AsyncRoundTripper[combiner.PipelineResponse]

			if tt.expectError {
				rt = NewAdjustStartEndWare(defaultStart, defaultBuffer, tt.sendNanos).
					Wrap(GetRoundTripperFunc())
			} else {
				rt = NewAdjustStartEndWare(defaultStart, defaultBuffer, tt.sendNanos).
					Wrap(GetRoundTripperFuncWithAsserts(t, func(t *testing.T, r Request) {
						assertStartEnd(t, r, tt.sendNanos, true)
					}))
			}

			resp, err := rt.RoundTrip(NewHTTPRequest(makeRequest(tt.params)))
			require.NoError(t, err)
			httpResponse, _, err := resp.Next(context.Background())
			require.NoError(t, err)

			if tt.expectError {
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusBadRequest, httpResponse.HTTPResponse().StatusCode)
			}
		})
	}
}
