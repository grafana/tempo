package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"single dash", []string{"-health"}, true},
		{"double dash", []string{"--health"}, true},
		{"with other flags", []string{"-config.file=tempo.yaml", "-health"}, true},
		{"no health flag", []string{"-config.file=tempo.yaml"}, false},
		{"empty args", []string{}, false},
		{"health as value", []string{"-config.file=health"}, false},
		{"health prefix", []string{"-health.url=http://localhost:3200/ready"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CheckHealth(tt.args))
		})
	}
}

func TestGetHealthURL(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"default URL", []string{"-health"}, defaultHealthURL},
		{"custom URL with equals", []string{"-health", "-health.url=http://localhost:8080/ready"}, "http://localhost:8080/ready"},
		{"custom URL with space", []string{"-health", "-health.url", "http://localhost:8080/ready"}, "http://localhost:8080/ready"},
		{"double dash", []string{"--health", "--health.url=http://localhost:8080/ready"}, "http://localhost:8080/ready"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getHealthURL(tt.args))
		})
	}
}

func TestRunHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantExit   int
	}{
		{"healthy", http.StatusOK, 0},
		{"unhealthy", http.StatusServiceUnavailable, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			args := []string{"-health", fmt.Sprintf("-health.url=%s/ready", server.URL)}
			assert.Equal(t, tt.wantExit, RunHealthCheck(args))
		})
	}
}
