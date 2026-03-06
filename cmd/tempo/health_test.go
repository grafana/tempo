package main

import (
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
		{"empty args", nil, false},
		{"health as substring", []string{"-health.url=http://localhost:3200/ready"}, false},
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
		{"default when no url flag", []string{"-health"}, defaultHealthURL},
		{"equals syntax", []string{"-health.url=http://custom:9090/ready"}, "http://custom:9090/ready"},
		{"space syntax", []string{"-health.url", "http://custom:9090/ready"}, "http://custom:9090/ready"},
		{"double dash equals", []string{"--health.url=http://custom:9090/ready"}, "http://custom:9090/ready"},
		{"colon syntax", []string{"-health.url:http://custom:9090/ready"}, "http://custom:9090/ready"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getHealthURL(tt.args))
		})
	}
}

func TestRunHealthCheck(t *testing.T) {
	t.Run("healthy endpoint returns 0", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		code := RunHealthCheck([]string{"-health", "-health.url=" + srv.URL + "/ready"})
		assert.Equal(t, 0, code)
	})

	t.Run("unhealthy endpoint returns 1", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		code := RunHealthCheck([]string{"-health", "-health.url=" + srv.URL + "/ready"})
		assert.Equal(t, 1, code)
	})

	t.Run("unreachable endpoint returns 1", func(t *testing.T) {
		code := RunHealthCheck([]string{"-health", "-health.url=http://127.0.0.1:1/ready"})
		assert.Equal(t, 1, code)
	})
}
