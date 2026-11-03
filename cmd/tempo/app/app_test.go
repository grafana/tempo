package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/tempo/v3/pkg/usagestats"
	"github.com/grafana/tempo/v3/pkg/util"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestAppNewSetsStorageBlockFormatUsageStat(t *testing.T) {
	config := NewDefaultConfig()
	config.StorageConfig.Trace.Block.Version = "vParquet4"

	_, err := New(*config)
	require.NoError(t, err)

	report := usagestats.BuildStats()
	require.Equal(t, "vParquet4", report.Metrics["storage_block_format"])
}

func useFreshRegistry(t *testing.T) {
	t.Helper()
	registry := prometheus.NewRegistry()
	previousRegisterer := prometheus.DefaultRegisterer
	previousGatherer := prometheus.DefaultGatherer
	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = previousRegisterer
		prometheus.DefaultGatherer = previousGatherer
	})
}

func TestApp_RunStop(t *testing.T) {
	tests := []struct {
		name         string
		workIsFile   bool
		wantFlushErr bool
	}{
		{name: "clean stop"},
		{name: "flush fails during shutdown", workIsFile: true, wantFlushErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useFreshRegistry(t)
			tempDir := t.TempDir()
			workPath := tempDir
			if tt.workIsFile {
				workPath = filepath.Join(tempDir, "work-cache")
				require.NoError(t, os.WriteFile(workPath, []byte("not a directory"), 0o600))
			}

			config := NewDefaultConfig()
			config.Target = BackendScheduler
			config.Server.HTTPListenPort = util.MustGetFreePort()
			config.Server.GRPCListenPort = util.MustGetFreePort()
			config.StorageConfig.Trace.Backend = backend.Local
			config.StorageConfig.Trace.Local.Path = filepath.Join(tempDir, "tempo")
			config.StorageConfig.Trace.WAL.Filepath = filepath.Join(tempDir, "wal")
			config.BackendScheduler.LocalWorkPath = workPath
			config.UsageReport.Enabled = false

			app, err := New(*config)
			require.NoError(t, err)

			errCh := make(chan error, 1)
			go func() {
				errCh <- app.Run()
			}()

			healthCheckURL := fmt.Sprintf("http://localhost:%d/ready", config.Server.HTTPListenPort)
			require.Eventually(t, func() bool {
				t.Log("Checking Tempo is up...")
				// #nosec G107 -- nosemgrep: tainted-url-host
				resp, httpErr := http.Get(healthCheckURL)
				return httpErr == nil && resp.StatusCode == http.StatusOK
			}, 30*time.Second, 1*time.Second)

			app.Stop()

			require.Eventually(t, func() bool {
				t.Log("Checking Tempo is down...")
				// #nosec G107 -- nosemgrep: tainted-url-host
				_, httpErr := http.Get(healthCheckURL)
				return httpErr != nil
			}, 60*time.Second, 1*time.Second)

			select {
			case runErr := <-errCh:
				require.NoError(t, runErr)
			case <-time.After(30 * time.Second):
				t.Fatal("tempo did not finish stopping")
			}
			if tt.wantFlushErr {
				service, ok := app.serviceMap[BackendScheduler]
				require.True(t, ok)
				require.Error(t, service.FailureCase())
			}
		})
	}
}

func TestApp_RunExitsWhenOverridesFailToLoad(t *testing.T) {
	useFreshRegistry(t)

	tempDir := t.TempDir()
	overridesFile := filepath.Join(tempDir, "overrides.yaml")
	require.NoError(t, os.WriteFile(overridesFile, []byte("not: valid: {{{"), 0o600))

	config := NewDefaultConfig()
	config.Target = BackendScheduler
	config.Server.HTTPListenPort = util.MustGetFreePort()
	config.Server.GRPCListenPort = util.MustGetFreePort()
	config.StorageConfig.Trace.Backend = backend.Local
	config.StorageConfig.Trace.Local.Path = filepath.Join(tempDir, "tempo")
	config.StorageConfig.Trace.WAL.Filepath = filepath.Join(tempDir, "wal")
	config.BackendScheduler.LocalWorkPath = tempDir
	config.UsageReport.Enabled = false
	config.Overrides.PerTenantOverrideConfig = overridesFile

	app, err := New(*config)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	select {
	case err := <-errCh:
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to load runtime config")
	case <-time.After(30 * time.Second):
		app.Stop()
		t.Fatal("process kept running after overrides failed to load")
	}
}
