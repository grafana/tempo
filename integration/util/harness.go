package util

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/exporter"
	"gopkg.in/yaml.v2"
)

const (
	ServiceDistributor      = "distributor"
	ServiceQueryFrontend    = "query-frontend"
	ServiceQuerier          = "querier"
	ServiceMetricsGenerator = "metrics-generator"
	ServiceLiveStoreZoneA   = "live-store-zone-a-0"
	ServiceLiveStoreZoneB   = "live-store-zone-b-0"
	ServiceBlockBuilder     = "block-builder-0"
)

// TempoHarness contains all the services and clients needed to run integration tests
// with the new Tempo architecture (Kafka + LiveStore).
type TempoHarness struct {
	// Infrastructure services
	Kafka      e2e.Service      // Kafka service
	Backend    e2e.Service      // Object storage backend (Minio/GCS/Azure/Local)
	Prometheus *e2e.HTTPService // Optional: prometheus for metrics generator

	// Tempo services - use constants above to access services by name
	Services map[string]*e2e.HTTPService

	// Clients
	HTTPClient     *httpclient.Client    // HTTP client for Tempo API
	JaegerExporter *JaegerToOTLPExporter // Client for sending traces via OTLP // jpe - do we need both jaeger and otlp exporter? do we just need a function to send traces?
	OTLPExporter   exporter.Traces       // Direct OTLP exporter

	// Endpoints
	DistributorOTLPEndpoint   string // OTLP gRPC endpoint (port 4317) // jpe - do we need these if we have the HTTP clients above?
	QueryFrontendHTTPEndpoint string // HTTP endpoint (port 3200)
	QueryFrontendGRPCEndpoint string // gRPC endpoint (port 3200)

	// Overrides file path for dynamic updates
	overridesPath string
}

// TestHarnessConfig provides configuration options for the test harness
type TestHarnessConfig struct {
	// ConfigOverlay is a config file that will be merged on top of config-base.yaml
	// This allows tests to only specify the differences from the default config
	// If empty, only config-base.yaml will be used
	ConfigOverlay string

	// jpe -
	// bitmask? worker/scheduler? maybe provide different "layers" as an enum: RecentDataQuerying, BackendQuerying, BackendWork, MetricsGenerator?

	// LiveStoreFastFlushDuration configures the flush check period and max block duration to the provided duration which will force traces to be flushed more aggressively to complete blocks
	LiveStoreFastFlush time.Duration // jpe - better to just put in the overlay?

	// EnableMetricsGenerator starts a metrics generator and Prometheus instance
	EnableMetricsGenerator bool

	// ExtraLiveStoreArgs are additional arguments to pass to live store instances - jpe - do we need the extra args here and below?
	ExtraLiveStoreArgs []string

	// ExtraBlockBuilderArgs are additional arguments to pass to block builder instances
	ExtraBlockBuilderArgs []string
}

// WithTempoHarness sets up the new Tempo architecture and waits for everything to be ready.
//
// Components started:
// - Object storage backend (S3/Azure/GCS - auto-detected from config, local backend is skipped)
// - Kafka
// - LiveStore instances (in zone-a/zone-b pairs, default 1 pair = 2 instances)
// - Block Builder instances (optional, configurable count)
// - Distributor
// - Metrics Generator + Prometheus (optional)
// - Query Frontend + Querier (always started)
//
// Example usage:
//
//	func TestMyFeature(t *testing.T) {
//		s, err := e2e.NewScenario("tempo_e2e")
//		require.NoError(t, err)
//		defer s.Close()
//
//		util.WithTempoHarness(t, s, util.TestHarnessConfig{
//			ConfigOverlay: "config-s3.yaml", // Optional: overlay config on top of config-base.yaml
//		}, func(h *util.TempoHarness) {
//			// Send traces
//			info := tempoUtil.NewTraceInfo(time.Now(), "")
//			require.NoError(t, info.EmitAllBatches(h.JaegerExporter))
//
//			// Query traces
//			trace, err := h.HTTPClient.QueryTrace(info.HexID())
//			require.NoError(t, err)
//		})
//	}
func WithTempoHarness(t *testing.T, s *e2e.Scenario, config TestHarnessConfig, testFunc func(*TempoHarness)) {
	t.Helper()

	harness := &TempoHarness{
		Services: map[string]*e2e.HTTPService{},
	}

	// Load base config into map
	baseConfigPath := "../util/config-base.yaml" // jpe - path?
	baseBuff, err := os.ReadFile(baseConfigPath)
	require.NoError(t, err, "failed to read base config file")

	var baseMap map[any]any
	err = yaml.Unmarshal(baseBuff, &baseMap)
	require.NoError(t, err, "failed to parse base config file")

	// Apply config overlay if provided
	if config.ConfigOverlay != "" {
		overlayBuff, err := os.ReadFile(config.ConfigOverlay)
		require.NoError(t, err, "failed to read config overlay file")

		var overlayMap map[any]any
		err = yaml.Unmarshal(overlayBuff, &overlayMap)
		require.NoError(t, err, "failed to parse config overlay file")

		// Merge overlay onto base
		baseMap = mergeMaps(baseMap, overlayMap)
	}

	// Create empty overrides file
	overridesPath := s.SharedDir() + "/overrides.yaml"
	err = os.WriteFile(overridesPath, []byte("overrides: {}\n"), 0644)
	require.NoError(t, err, "failed to write initial overrides file")
	harness.overridesPath = overridesPath

	// make modifications if necessary
	if config.LiveStoreFastFlush > 0 {
		baseMap["live_store"].(map[any]any)["flush_check_period"] = config.LiveStoreFastFlush.String()
		baseMap["live_store"].(map[any]any)["max_block_duration"] = config.LiveStoreFastFlush.String()
	}

	// Write the merged config to the shared directory
	mergedConfigBytes, err := yaml.Marshal(baseMap)
	require.NoError(t, err, "failed to marshal merged config")
	configPath := s.SharedDir() + "/config.yaml"
	err = os.WriteFile(configPath, mergedConfigBytes, 0644)
	require.NoError(t, err, "failed to write merged config file")

	// Unmarshal to app.Config to validate it works
	var cfg app.Config
	err = yaml.UnmarshalStrict(mergedConfigBytes, &cfg)
	require.NoError(t, err, "failed to unmarshal merged config into app.Config")

	// Start object storage backend if not using local filesystem
	// Local backend doesn't require an external service
	if cfg.StorageConfig.Trace.Backend != backend.Local {
		var backendErr error
		harness.Backend, backendErr = startBackend(t, s, cfg)
		require.NoError(t, backendErr, "failed to start backend")
	}

	// Start Kafka
	harness.Kafka = e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(harness.Kafka), "failed to start Kafka")

	// Livestores
	liveStoreZoneA := NewNamedTempoLiveStore(
		"live-store-zone-a",
		0,
		config.ExtraLiveStoreArgs...,
	)
	harness.Services[ServiceLiveStoreZoneA] = liveStoreZoneA
	require.NoError(t, s.StartAndWaitReady(liveStoreZoneA), "failed to start live store zone a")

	liveStoreZoneB := NewNamedTempoLiveStore(
		"live-store-zone-b",
		0,
		config.ExtraLiveStoreArgs...,
	)
	harness.Services[ServiceLiveStoreZoneB] = liveStoreZoneB
	require.NoError(t, s.StartAndWaitReady(liveStoreZoneB), "failed to start live store zone b")

	// Wait for live stores to join the partition ring
	matchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: "Active"},
		{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
	}
	require.NoError(t, liveStoreZoneA.WaitSumMetricsWithOptions(
		e2e.Equals(float64(1)),
		[]string{"tempo_partition_ring_partitions"},
		e2e.WithLabelMatchers(matchers...),
	), "live stores failed to join partition ring")

	// Start Distributor
	harness.Services[ServiceDistributor] = NewTempoDistributor()
	require.NoError(t, s.StartAndWaitReady(harness.Services[ServiceDistributor]), "failed to start distributor")

	// Start Block Builder
	blockBuilder := NewTempoBlockBuilder(0, config.ExtraBlockBuilderArgs...)
	harness.Services[ServiceBlockBuilder] = blockBuilder

	require.NoError(t, s.StartAndWaitReady(blockBuilder), "failed to start block builders")

	// Start Metrics Generator and Prometheus (if requested)
	if config.EnableMetricsGenerator {
		harness.Services[ServiceMetricsGenerator] = NewTempoMetricsGenerator()
		harness.Prometheus = NewPrometheus()
		require.NoError(t, s.StartAndWaitReady(harness.Services[ServiceMetricsGenerator], harness.Prometheus), "failed to start metrics generator and prometheus")
	}

	// Start Query Frontend and Querier
	harness.Services[ServiceQueryFrontend] = NewTempoQueryFrontend()
	harness.Services[ServiceQuerier] = NewTempoQuerier()
	require.NoError(t, s.StartAndWaitReady(harness.Services[ServiceQueryFrontend], harness.Services[ServiceQuerier]), "failed to start query frontend and querier")

	// Set endpoints jpe - do we need both http and grpc endpoints here?
	harness.QueryFrontendHTTPEndpoint = harness.Services[ServiceQueryFrontend].Endpoint(3200)
	harness.QueryFrontendGRPCEndpoint = harness.Services[ServiceQueryFrontend].Endpoint(3200)

	// Create HTTP client
	harness.HTTPClient = httpclient.New("http://"+harness.QueryFrontendHTTPEndpoint, "")

	// Set distributor endpoints
	harness.DistributorOTLPEndpoint = harness.Services[ServiceDistributor].Endpoint(4317)

	// Create Jaeger to OTLP exporter - jpe - do we need both of these?
	harness.JaegerExporter, err = NewJaegerToOTLPExporter(harness.DistributorOTLPEndpoint)
	require.NoError(t, err, "failed to create Jaeger to OTLP exporter")
	require.NotNil(t, harness.JaegerExporter)

	// Create OTLP exporter (jpe - do we need both of these?)
	harness.OTLPExporter, err = NewOtelGRPCExporter(harness.DistributorOTLPEndpoint)
	require.NoError(t, err, "failed to create OTLP exporter")
	require.NotNil(t, harness.OTLPExporter)

	// Run the test function
	testFunc(harness)
}

// startBackend starts the appropriate object storage backend based on the config
func startBackend(t *testing.T, s *e2e.Scenario, cfg app.Config) (e2e.Service, error) {
	t.Helper()

	var backendService e2e.Service
	switch cfg.StorageConfig.Trace.Backend {
	case backend.S3:
		port, err := parsePort(cfg.StorageConfig.Trace.S3.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = e2edb.NewMinio(port, "tempo")
		if backendService == nil {
			return nil, fmt.Errorf("error creating minio backend")
		}
		err = s.StartAndWaitReady(backendService)
		if err != nil {
			return nil, err
		}
	case backend.Azure:
		port, err := parsePort(cfg.StorageConfig.Trace.Azure.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = newAzurite(port)
		err = s.StartAndWaitReady(backendService)
		if err != nil {
			return nil, err
		}
		// Get the actual endpoint after the service is started
		httpService, ok := backendService.(*e2e.HTTPService)
		if ok {
			cfg.StorageConfig.Trace.Azure.Endpoint = httpService.Endpoint(port)
		}
		_, err = azure.CreateContainer(context.TODO(), cfg.StorageConfig.Trace.Azure)
		if err != nil {
			return nil, err
		}
	case backend.GCS:
		port, err := parsePort(cfg.StorageConfig.Trace.GCS.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = newGCS(port)
		if backendService == nil {
			return nil, fmt.Errorf("error creating gcs backend")
		}
		err = s.StartAndWaitReady(backendService)
		if err != nil {
			return nil, err
		}
	}

	return backendService, nil
}

// parsePort extracts the port number from an endpoint string
func parsePort(endpoint string) (int, error) {
	substrings := strings.Split(endpoint, ":")
	portStrings := strings.Split(substrings[len(substrings)-1], "/")
	port, err := strconv.Atoi(portStrings[0])
	if err != nil {
		return 0, err
	}
	return port, nil
}

// mergeMaps recursively merges overlay map onto base map
// Values in overlay take precedence over base values
func mergeMaps(base, overlay map[any]any) map[any]any {
	result := make(map[any]any)

	// Copy all base values
	for k, v := range base {
		result[k] = v
	}

	// Overlay values, recursively merging nested maps
	for k, v := range overlay {
		if v == nil {
			result[k] = v
			continue
		}

		// If both base and overlay have a map at this key, merge recursively
		if baseVal, exists := result[k]; exists {
			baseMap, baseIsMap := toMapAnyAny(baseVal)
			overlayMap, overlayIsMap := toMapAnyAny(v)

			if baseIsMap && overlayIsMap {
				result[k] = mergeMaps(baseMap, overlayMap)
				continue
			}
		}

		// Otherwise, overlay value replaces base value
		result[k] = v
	}

	return result
}

// toMapAnyAny converts various map types to map[any]any
func toMapAnyAny(v any) (map[any]any, bool) {
	switch m := v.(type) {
	case map[any]any:
		return m, true
	case map[string]any:
		result := make(map[any]any)
		for k, v := range m {
			result[k] = v
		}
		return result, true
	default:
		return nil, false
	}
}

// newAzurite creates a new Azurite service for Azure blob storage emulation
func newAzurite(port int) *e2e.HTTPService {
	s := e2e.NewHTTPService(
		"azurite",
		azuriteImage,
		e2e.NewCommandWithoutEntrypoint("sh", "-c", "azurite -l /data --blobHost 0.0.0.0"),
		e2e.NewHTTPReadinessProbe(port, "/devstoreaccount1?comp=list", 403, 403), // If we get 403 the Azurite is ready
		port, // blob storage port
	)

	s.SetBackoff(TempoBackoff())

	return s
}

// newGCS creates a new fake GCS service for Google Cloud Storage emulation
func newGCS(port int) *e2e.HTTPService {
	commands := []string{
		"mkdir -p /data/tempo",
		"/bin/fake-gcs-server -data /data -public-host=tempo_e2e-gcs -port=4443",
	}
	s := e2e.NewHTTPService(
		"gcs",
		gcsImage,
		e2e.NewCommandWithoutEntrypoint("sh", "-c", strings.Join(commands, " && ")),
		e2e.NewHTTPReadinessProbe(port, "/", 400, 400), // for lack of a better way, readiness probe does not support https at the moment
		port,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

// UpdateOverrides updates the tenant overrides file with the provided configuration.
// The overrides parameter should be a map where keys are tenant IDs and values are
// override configurations for that tenant.
//
// Example usage:
//
//	h.UpdateOverrides(map[string]*overrides.Overrides{
//		"tenant-1": {
//			Ingestion: overrides.IngestionOverrides{
//				RateLimitBytes: 1000000,
//			},
//		},
//	})
//
// Tempo will automatically reload the overrides based on the per_tenant_override_period
// configured in config-base.yaml.
func (h *TempoHarness) UpdateOverrides(tenantOverrides map[string]*overrides.Overrides) error {
	overridesConfig := struct {
		Overrides map[string]*overrides.Overrides `yaml:"overrides"`
	}{
		Overrides: tenantOverrides,
	}

	data, err := yaml.Marshal(overridesConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal overrides: %w", err)
	}

	err = os.WriteFile(h.overridesPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write overrides file: %w", err)
	}

	// overrides reload every 1 second. wait 5 to make sure it gets loaded - jpe - metric for determinism?
	time.Sleep(5 * time.Second)

	return nil
}
