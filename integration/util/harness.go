package util

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

const (
	// tempo
	ServiceDistributor      = "distributor"
	ServiceQueryFrontend    = "query-frontend"
	ServiceQuerier          = "querier"
	ServiceMetricsGenerator = "metrics-generator"
	ServiceLiveStoreZoneA   = "live-store-zone-a-0"
	ServiceLiveStoreZoneB   = "live-store-zone-b-0"
	ServiceBlockBuilder     = "block-builder-0"
	ServiceBackendScheduler = "backend-scheduler"
	ServiceBackendWorker    = "backend-worker"

	// not tempo!
	ServiceObjectStorage = "object-storage"
	ServicePrometheus    = "prometheus"
)

var AllTempoServices = []string{
	ServiceDistributor,
	ServiceQueryFrontend,
	ServiceQuerier,
	ServiceMetricsGenerator,
	ServiceLiveStoreZoneA,
	ServiceLiveStoreZoneB,
	ServiceBlockBuilder,
}

const (
	azuriteImage = "mcr.microsoft.com/azure-storage/azurite:3.35.0"
	gcsImage     = "fsouza/fake-gcs-server:1.52.2"
)

// DeploymentMode specifies whether to run Tempo as a single binary, microservices or none.
type DeploymentMode int

const (
	// DeploymentModeMicroservices runs Tempo as separate microservices (default)
	DeploymentModeMicroservices DeploymentMode = iota
	// DeploymentModeSingleBinary runs Tempo as a single all-in-one binary
	DeploymentModeSingleBinary
	// DeploymentModeNone does not start any Tempo services. This seems odd but it's used by poller_test.go to just start backends
	DeploymentModeNone
)

// ComponentsMask is a bitmask for controlling which optional Tempo components are started. These flags are meant to be used
// with the | operator to request multiple component sets from the test harness. For instance if you wanted to test the full
// querying capabilities of Tempo you would use: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying
type ComponentsMask uint

const (
	componentsKafka ComponentsMask = 1 << iota
	componentsPrometheus
	componentsLiveStore
	componentsDistributor
	componentsQueryFrontendQuerier // both query-frontend and querier. bundled b/c the query-frontend will never raise its readiness probe w/o querier
	componentsBlockBuilder
	componentsMetricsGenerator
	componentsBackendSchedulerWorker // both backend-scheduler and backend-worker. too tightly coupled to separate

	// the config-base.yaml uses live-stores as the memberlist gossip seeds and so always have to start live stores to stabilize the memberlist gossip cluster. even if they are not needed by the test
	// todo: improve this by templating the memberlist join members and dynamically choosing which components are the seeds

	// ComponentsRecentDataQuerying starts the distributor, kafka, query-frontend, querier, and livestores for recent data querying. (default)
	ComponentsRecentDataQuerying = componentsLiveStore | componentsDistributor | componentsQueryFrontendQuerier | componentsKafka
	// ComponentsBackendQuerying starts the distributor, kafka, query-frontend, querier, and block-builder for backend querying.
	ComponentsBackendQuerying = componentsDistributor | componentsQueryFrontendQuerier | componentsBlockBuilder | componentsKafka | componentsLiveStore
	// ComponentsMetricsGeneration starts the distributor, kafka, metrics generator and prometheus for metrics generation.
	ComponentsMetricsGeneration = componentsDistributor | componentsMetricsGenerator | componentsPrometheus | componentsKafka | componentsLiveStore
	// ComponentsBackendWork starts the backend scheduler and worker for testing backend work.
	ComponentsBackendWork = componentsBackendSchedulerWorker
)

// BackendsMask is a bitmask for controlling which object storage backends are started. These flags are meant to be used
// with the | operator to request multiple backends from the test harness.
// The test function will be called once for each backend and so specifying multiple backends will result in longer tests!
// Use this only when you really need to test the behavior of Tempo against multiple backends.
type BackendsMask uint

const (
	// BackendLocal starts the local backend. (default)
	BackendLocal BackendsMask = 1 << iota
	// BackendObjectStorageS3 starts the S3 backend.
	BackendObjectStorageS3
	// BackendObjectStorageAzure starts the Azure backend.
	BackendObjectStorageAzure
	// BackendObjectStorageGCS starts the GCS backend.
	BackendObjectStorageGCS

	// BackendObjectStorageAll starts all three object storage backends. A convenience flag to avoid having to specify all three backends manually.
	BackendObjectStorageAll = BackendObjectStorageS3 | BackendObjectStorageAzure | BackendObjectStorageGCS
)

type TempoHarness struct {
	Services     map[string]*e2e.HTTPService
	TestScenario *e2e.Scenario

	overridesPath  string
	readinessProbe e2e.ReadinessProbe
}

// TestHarnessConfig provides configuration options for the test harness
type TestHarnessConfig struct {
	// ConfigOverlay is a config file that will be merged on top of config-base.yaml
	// This allows tests to only specify the differences from the default config
	// If empty, only config-base.yaml will be used
	ConfigOverlay string

	// ConfigTemplateData provides template variables to expand in the ConfigOverlay
	// Template variables should use Go template syntax: {{ .VariableName }}
	ConfigTemplateData map[string]any

	// ReadinessProbe is a function to use a custom readiness probe for the Tempo services.
	// It's used by the https tests to swap the readiness port to 3201.
	ReadinessProbe e2e.ReadinessProbe

	// PreStartHook is called before starting Tempo services to perform any setup or adjustments.
	// It's used by a few tests to copy in required files, start prerequisite services, or adjust the template data for overlays.
	PreStartHook func(*e2e.Scenario, map[string]any) error

	// DeploymentMode specifies whether to run as single binary, microservices, or none
	// Defaults to DeploymentModeMicroservices
	DeploymentMode DeploymentMode

	// Backends is a bitmask controlling which object storage backends are started.
	// Defaults to BackendLocal
	Backends BackendsMask

	// Components is a bitmask controlling which optional Tempo components are started.
	// Recent data components are always started. Use this to add optional components.
	// Defaults to ComponentsRecentDataQuerying
	Components ComponentsMask
}

// RunIntegrationTests sets up Tempo for integration tests as requested through the config and then calls the provided testFunc
func RunIntegrationTests(t *testing.T, config TestHarnessConfig, testFunc func(*TempoHarness)) {
	t.Helper()
	t.Parallel()

	// defaults
	if config.Backends == 0 {
		config.Backends = BackendLocal
	}

	if config.DeploymentMode == 0 {
		config.DeploymentMode = DeploymentModeMicroservices
	}

	if config.Components == 0 {
		config.Components = ComponentsRecentDataQuerying
	}

	backendTCs := backendTestCases(config.Backends)

	// Run tests for each deployment mode and backend combination
	for _, be := range backendTCs {
		// t.Run() with t.Parallel() here will cause some tests to run faster, but others like TestKVStores will fail for unknown reasons.
		runTempoHarness(t, config, be.name, testFunc)
	}
}

type backendTestCase struct {
	mask BackendsMask
	name string
}

// backendTestCases returns the list of backends to test based on the config
func backendTestCases(backends BackendsMask) []backendTestCase {
	var result []backendTestCase

	if backends&BackendObjectStorageS3 != 0 {
		result = append(result, backendTestCase{BackendObjectStorageS3, backend.S3})
	}
	if backends&BackendObjectStorageAzure != 0 {
		result = append(result, backendTestCase{BackendObjectStorageAzure, backend.Azure})
	}
	if backends&BackendObjectStorageGCS != 0 {
		result = append(result, backendTestCase{BackendObjectStorageGCS, backend.GCS})
	}
	if backends&BackendLocal != 0 {
		result = append(result, backendTestCase{BackendLocal, backend.Local})
	}

	return result
}

// runTempoHarness is the internal implementation that sets up and runs a single harness instance
func runTempoHarness(t *testing.T, harnessCfg TestHarnessConfig, requestedBackend string, testFunc func(*TempoHarness)) {
	t.Helper()

	// Create scenario with normalized test name
	name := normalizeTestName(t.Name())
	s, err := e2e.NewScenario("e2e_" + name)
	require.NoError(t, err)
	defer s.Close()

	harness := &TempoHarness{
		Services:       map[string]*e2e.HTTPService{},
		TestScenario:   s,
		readinessProbe: harnessCfg.ReadinessProbe,
	}

	// Setup config and infrastructure
	tempoCfg := setupConfig(t, s, &harnessCfg, requestedBackend, harness)

	// Start object storage backend if not using local filesystem
	if tempoCfg.StorageConfig.Trace.Backend != backend.Local {
		backend, err := startBackend(t, s, tempoCfg)
		require.NoError(t, err, "failed to start backend")
		harness.Services[ServiceObjectStorage] = backend
	}

	// bail out here if we don't need any tempo components
	if harnessCfg.DeploymentMode == DeploymentModeNone {
		return
	}

	// Start Kafka
	//   todo: should we add a field to reference kafka on the harness? not needed atm. maybe to test failure states by stopping it?
	if harnessCfg.Components&componentsKafka != 0 {
		kafka := e2edb.NewKafka()
		require.NoError(t, s.StartAndWaitReady(kafka), "failed to start Kafka")
	}

	if harnessCfg.Components&componentsPrometheus != 0 {
		prometheus := newPrometheus()
		require.NoError(t, s.StartAndWaitReady(prometheus), "failed to start prometheus")
		harness.Services[ServicePrometheus] = prometheus
	}

	// Start Tempo services based on deployment mode
	switch harnessCfg.DeploymentMode {
	case DeploymentModeSingleBinary:
		require.NoError(t, harness.startSingleBinary(t), "failed to start single binary")
	case DeploymentModeMicroservices:
		require.NoError(t, harness.startMicroservices(t, harnessCfg), "failed to start microservices")
	default:
		panic(fmt.Sprintf("unknown deployment mode: %d", harnessCfg.DeploymentMode))
	}

	// Run the test function
	testFunc(harness)
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

	err = os.WriteFile(h.overridesPath, data, 0o644) // nolint:gosec // G306: Expect WriteFile permissions to be 0600 or less
	if err != nil {
		return fmt.Errorf("failed to write overrides file: %w", err)
	}

	// overrides reload every 1 second. wait 5 to make sure it gets loaded
	time.Sleep(5 * time.Second)

	return nil
}

// GetConfig reads and parses the Tempo configuration file that was set up by the harness.
// Returns the parsed app.Config or an error if reading/parsing fails.
func (h *TempoHarness) GetConfig() (app.Config, error) {
	configPath := sharedContainerPath(h.TestScenario, tempoConfigFile)

	buff, err := os.ReadFile(configPath)
	if err != nil {
		return app.Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg app.Config
	err = yaml.UnmarshalStrict(buff, &cfg)
	if err != nil {
		return app.Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// restartServiceWithConfigOverlay stops a service, applies a config overlay, and restarts the service.
// The overlay file is merged onto the existing config, with overlay values taking precedence.
func (h *TempoHarness) restartServiceWithConfigOverlay(t *testing.T, service *e2e.HTTPService, overlayPath string) error {
	// Stop the service
	err := service.Stop()
	if strings.Contains(err.Error(), "exit status 137") { // 137 is returned by linux when it is force killed b/c it doesn't stop in time.
		t.Logf("service %s was force killed during stop: %v", service.Name(), err)
	} else if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Apply overlay to current config
	err = applyConfigOverlay(h.TestScenario, overlayPath, nil)
	if err != nil {
		return fmt.Errorf("failed to apply config overlay: %w", err)
	}

	// Restart the service
	if err := service.Start(h.TestScenario.NetworkName(), h.TestScenario.SharedDir()); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	if err := service.WaitReady(); err != nil {
		return fmt.Errorf("service did not become ready after restart: %w", err)
	}

	return nil
}

func (h *TempoHarness) WaitTracesWritable(t *testing.T) {
	t.Helper()

	matchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: "Active"},
		{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
	}

	// distributors have to be non-nil if we're writing anything
	distributor := h.Services[ServiceDistributor]
	require.NoError(t, distributor.WaitSumMetricsWithOptions(
		e2e.Equals(float64(1)),
		[]string{"tempo_partition_ring_partitions"},
		e2e.WithLabelMatchers(matchers...)), "distributor failed to see the partition ring")
}

func (h *TempoHarness) WaitTracesQueryable(t *testing.T, traces int) {
	t.Helper()

	liveStoreZoneA := h.Services[ServiceLiveStoreZoneA]
	require.NoError(t, liveStoreZoneA.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(float64(traces)), []string{"tempo_live_store_traces_created_total"}, e2e.WaitMissingMetrics))

	liveStoreZoneB := h.Services[ServiceLiveStoreZoneB]
	require.NoError(t, liveStoreZoneB.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(float64(traces)), []string{"tempo_live_store_traces_created_total"}, e2e.WaitMissingMetrics))
}

func (h *TempoHarness) WaitTracesWrittenToBackend(t *testing.T, traces int) {
	t.Helper()

	queryFrontend := h.Services[ServiceQueryFrontend]
	require.NoError(t, queryFrontend.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(float64(traces)), []string{"tempodb_backend_objects_total"}, e2e.WaitMissingMetrics))
}

// ForceBackendQuerying restarts the query frontend with the query-backend config overlay applied.
// Using this function requires re-creating any clients pointing at the frontend!
func (h *TempoHarness) ForceBackendQuerying(t *testing.T) {
	frontend := h.Services[ServiceQueryFrontend]
	require.NoError(t, h.restartServiceWithConfigOverlay(t, frontend, queryBackendConfigFile))
}

/*
  local object storage
*/
// startBackend starts the appropriate object storage backend based on the config
func startBackend(t *testing.T, s *e2e.Scenario, cfg app.Config) (*e2e.HTTPService, error) {
	t.Helper()

	var backendService *e2e.HTTPService
	switch cfg.StorageConfig.Trace.Backend {
	case backend.S3:
		port, err := parsePort(cfg.StorageConfig.Trace.S3.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = e2edb.NewMinio(port, "tempo")
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
		cfg.StorageConfig.Trace.Azure.Endpoint = backendService.Endpoint(port)
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

// startMicroservices starts all Tempo microservices and waits for them to be ready
func (h *TempoHarness) startMicroservices(t *testing.T, config TestHarnessConfig) error {
	t.Helper()

	s := h.TestScenario
	readinessProbe := e2e.ReadinessProbe(e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299))
	if h.readinessProbe != nil {
		readinessProbe = h.readinessProbe
	}

	if config.Components&componentsLiveStore != 0 {
		liveStoreZoneA := NewTempoService("live-store-zone-a-0", "live-store", readinessProbe, nil, "-live-store.instance-availability-zone=zone-a")
		h.Services[ServiceLiveStoreZoneA] = liveStoreZoneA
		if err := s.StartAndWaitReady(liveStoreZoneA); err != nil {
			return fmt.Errorf("failed to start live store zone a: %w", err)
		}

		liveStoreZoneB := NewTempoService("live-store-zone-b-0", "live-store", readinessProbe, nil, "-live-store.instance-availability-zone=zone-b")
		h.Services[ServiceLiveStoreZoneB] = liveStoreZoneB
		if err := s.StartAndWaitReady(liveStoreZoneB); err != nil {
			return fmt.Errorf("failed to start live store zone b: %w", err)
		}
	}

	if config.Components&componentsDistributor != 0 {
		h.Services[ServiceDistributor] = NewTempoService("distributor", "distributor",
			readinessProbe,
			[]int{14250, 4317, 4318, 9411}, // jaeger grpc ingest, otlp grpc, otlp http, zipkin ingest
		)
		if err := s.StartAndWaitReady(h.Services[ServiceDistributor]); err != nil {
			return fmt.Errorf("failed to start distributor: %w", err)
		}
	}

	if config.Components&componentsQueryFrontendQuerier != 0 {
		h.Services[ServiceQueryFrontend] = NewTempoService("query-frontend", "query-frontend", readinessProbe, nil)
		h.Services[ServiceQuerier] = NewTempoService("querier", "querier", readinessProbe, nil)
		if err := s.StartAndWaitReady(h.Services[ServiceQueryFrontend], h.Services[ServiceQuerier]); err != nil {
			return fmt.Errorf("failed to start query frontend and querier: %w", err)
		}
	}

	if config.Components&componentsBlockBuilder != 0 {
		blockBuilder := NewTempoService("block-builder-0", "block-builder", readinessProbe, nil)
		h.Services[ServiceBlockBuilder] = blockBuilder
		if err := s.StartAndWaitReady(blockBuilder); err != nil {
			return fmt.Errorf("failed to start block builder: %w", err)
		}
	}

	if config.Components&componentsMetricsGenerator != 0 {
		h.Services[ServiceMetricsGenerator] = NewTempoService("metrics-generator", "metrics-generator", readinessProbe, nil)
		if err := s.StartAndWaitReady(h.Services[ServiceMetricsGenerator]); err != nil {
			return fmt.Errorf("failed to start metrics generator: %w", err)
		}
	}

	if config.Components&componentsBackendSchedulerWorker != 0 {
		scheduler := NewTempoService("backend-scheduler", "backend-scheduler", readinessProbe, nil)
		worker := NewTempoService("backend-worker", "backend-worker", readinessProbe, nil)
		h.Services[ServiceBackendScheduler] = scheduler
		h.Services[ServiceBackendWorker] = worker
		if err := s.StartAndWaitReady(scheduler, worker); err != nil {
			return fmt.Errorf("failed to start backend scheduler and worker: %w", err)
		}
	}

	return nil
}

// startSingleBinary starts Tempo as a single all-in-one binary and waits for it to be ready
func (h *TempoHarness) startSingleBinary(t *testing.T) error {
	t.Helper()

	readinessProbe := e2e.ReadinessProbe(e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299))
	if h.readinessProbe != nil {
		readinessProbe = h.readinessProbe
	}

	// Create single binary service with custom readiness probe
	// Using port 3201 for readiness to avoid conflicts with main HTTP port
	tempo := NewTempoAllInOne(readinessProbe)

	h.Services[ServiceDistributor] = tempo
	h.Services[ServiceQueryFrontend] = tempo
	h.Services[ServiceQuerier] = tempo
	h.Services[ServiceLiveStoreZoneA] = tempo
	h.Services[ServiceLiveStoreZoneB] = tempo
	h.Services[ServiceBlockBuilder] = tempo
	h.Services[ServiceMetricsGenerator] = tempo
	h.Services[ServiceBackendScheduler] = tempo
	h.Services[ServiceBackendWorker] = tempo

	if err := h.TestScenario.StartAndWaitReady(tempo); err != nil {
		return fmt.Errorf("failed to start tempo single binary: %w", err)
	}

	return nil
}

// normalizeTestName creates a valid Docker service name from a test name
func normalizeTestName(testName string) string {
	// max docker name length is 63. otherwise dns fails silently
	// max test name length is 40 to leave room prefix and suffix. the full container name will be e2e_<test name>_<service name>
	// this means that if two tests have the same first 40 characters in their names they will conflict!!
	maxNameLen := 40
	name := strings.TrimPrefix(testName, "Test")
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	// docker only allows a-zA-Z0-9_.- in a service name. replace everything else with _
	re := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	return re.ReplaceAllString(name, "_")
}
