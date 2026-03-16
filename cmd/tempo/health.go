package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultHealthURL = "http://localhost:3200/ready"
	healthTimeout    = 5 * time.Second
)

// RunHealthCheck performs a health check against the /ready endpoint.
// Returns exit code 0 if healthy, 1 if unhealthy.
func RunHealthCheck(url string) int {
	if url == "" {
		url = defaultHealthURL
	}

	client := &http.Client{
		Timeout: healthTimeout,
	}

	resp, err := client.Get(url) //nolint:gosec
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Tempo is healthy")
		return 0
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Fprintf(os.Stderr, "Tempo is unhealthy: status %d: %s\n", resp.StatusCode, string(body))
	return 1
}
