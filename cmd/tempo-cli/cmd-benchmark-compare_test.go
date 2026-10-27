package main

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServeCompare(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "compared")
	})
	ctx, cancel := context.WithCancel(context.Background())
	out, lines := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- serveCompare(ctx, ":0", h, lines)
		lines.Close()
	}()

	// A port of 0 is picked by the system, and the address says which. With
	// no host it is served on localhost only.
	line, err := bufio.NewReader(out).ReadString('\n')
	require.NoError(t, err)
	url := strings.TrimSpace(strings.TrimPrefix(line, "Serving the comparison on "))
	require.True(t, strings.HasPrefix(url, "http://localhost:"), url)

	resp, err := http.Get(url) //nolint:noctx // a test against a local server
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, "compared", string(body))

	// Cancelling stops the server cleanly.
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the server did not stop")
	}
}

func TestServeCompareBadAddress(t *testing.T) {
	err := serveCompare(context.Background(), "8080", http.NotFoundHandler(), io.Discard)
	require.ErrorContains(t, err, `invalid --http address "8080"`)
}
