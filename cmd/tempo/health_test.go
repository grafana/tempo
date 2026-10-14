package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantExit   int
	}{
		{"healthy", http.StatusOK, "", 0},
		{"unhealthy", http.StatusServiceUnavailable, "not ready", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			assert.Equal(t, tt.wantExit, RunHealthCheck(server.URL+"/ready"))
		})
	}
}

func TestRunHealthCheckDefaultURL(t *testing.T) {
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		http.DefaultTransport = healthCheckRoundTripper(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, defaultHealthURL, r.URL.String())
			return &http.Response{StatusCode: status, Body: http.NoBody}, nil
		})
		wantExit := 1
		if status == http.StatusOK {
			wantExit = 0
		}
		assert.Equal(t, wantExit, RunHealthCheck(""))
	}
}

type healthCheckRoundTripper func(*http.Request) (*http.Response, error)

func (f healthCheckRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRunHealthCheckConnectionError(t *testing.T) {
	assert.Equal(t, 1, RunHealthCheck("http://localhost:1/ready"))
}
