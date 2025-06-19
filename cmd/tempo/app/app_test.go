package app

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/require"
)

func TestApp_RunStop(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tempo-test-app-*")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(tempDir)
		require.NoError(t, err)
	}()

	config := NewDefaultConfig()
	config.StorageConfig.Trace.Backend = backend.Local
	config.StorageConfig.Trace.Local.Path = filepath.Join(tempDir, "tempo")
	config.StorageConfig.Trace.WAL.Filepath = filepath.Join(tempDir, "wal")
	config.UsageReport.Enabled = false // speeds up the shutdown process

	app, err := New(*config)
	require.NoError(t, err)

	// start Tempo
	go func() {
		require.NoError(t, app.Run())
	}()

	// check health endpoint is reachable
	healthCheckURL := "http://localhost:3200/ready"
	require.Eventually(t, func() bool {
		t.Log("Checking Tempo is up...")
		resp, httpErr := http.Get(healthCheckURL)
		return httpErr == nil && resp.StatusCode == http.StatusOK
	}, 30*time.Second, 1*time.Second)

	// stop Tempo
	app.Stop()

	// check health endpoint is not reachable anymore
	require.Eventually(t, func() bool {
		t.Log("Checking Tempo is down...")
		_, httpErr := http.Get(healthCheckURL)
		return httpErr != nil
	}, 30*time.Second, 1*time.Second)
}
