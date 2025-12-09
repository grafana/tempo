package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"text/template"
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
	ServiceBackendScheduler = "backend-scheduler"
	ServiceBackendWorker    = "backend-worker"
)

// DeploymentMode specifies whether to run Tempo as a single binary or microservices
type DeploymentMode int

const (
	// DeploymentModeMicroservices runs Tempo as separate microservices (default)
	DeploymentModeMicroservices DeploymentMode = iota // jpe should we run everything in both? should we force some things to single binary?
	// DeploymentModeSingleBinary runs Tempo as a single all-in-one binary
	DeploymentModeSingleBinary
)

// ComponentsMask is a bitmask for controlling which optional Tempo components are started.
// Recent data components (distributor, live stores, query frontend, querier) are always started.
type ComponentsMask uint

const (
	// ComponentsMetricsGenerator starts the metrics generator and Prometheus
	ComponentsMetricsGenerator ComponentsMask = 1 << iota
	// ComponentsBackendQuerying starts block builders for creating backend blocks
	ComponentsBackendQuerying
	// ComponentsBackendWork starts backend scheduler and worker for compaction
	ComponentsBackendWork
	// ComponentsObjectStorage runs the test 3 times, once for each backend (S3, Azure, GCS)
	ComponentsObjectStorage
	// ComponentRecentDataQuerying starts up components required for recent data querying: distributor, live store, query frontend, querier. This is default if nothing is specified.
	ComponentRecentDataQuerying
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
	HTTPClient     *httpclient.Client    // HTTP client for Tempo API jpe - gRPC client as well?
	JaegerExporter *JaegerToOTLPExporter // Client for sending traces via OTLP // jpe - do we need both jaeger and otlp exporter? do we just need a function to send traces?
	OTLPExporter   exporter.Traces       // Direct OTLP exporter

	TestScenario *e2e.Scenario

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

	// ConfigTemplateData provides template variables to expand in the ConfigOverlay
	// Template variables should use Go template syntax: {{ .VariableName }}
	ConfigTemplateData map[string]any

	// ConfigTemplateFunc is called before starting services to populate ConfigTemplateData
	// It receives the scenario and can start any prerequisite services (like etcd, consul)
	// and populate template variables with their connection information
	ConfigTemplateFunc func(*e2e.Scenario, map[string]any) error // jpe - add notes this can be used to start any other services as well? rethink template func. does it need scenario? better to start services otherwise?

	// DeploymentMode specifies whether to run as single binary or microservices
	// Defaults to DeploymentModeMicroservices
	DeploymentMode DeploymentMode // jpe - review tests. anything that doesn't require microservices can use single binary and save time

	// Components is a bitmask controlling which optional Tempo components are started. jpe - microservices only
	// Recent data components are always started. Use this to add optional components.
	// Defaults to ComponentsDefault (no optional components).
	// Example: Components: util.ComponentsMetricsGenerator | util.ComponentsBackendWork
	Components ComponentsMask
}

// WithTempoHarness sets up Tempo and waits for everything to be ready.
//
// Deployment Modes:
// - DeploymentModeMicroservices (default): Runs Tempo as separate microservices
// - DeploymentModeSingleBinary: Runs Tempo as a single all-in-one binary
//
// Components started (microservices mode):
// - Object storage backend (S3/Azure/GCS - auto-detected from config, local backend is skipped)
// - Kafka
// - LiveStore instances (in zone-a/zone-b pairs, default 1 pair = 2 instances)
// - Block Builder instances (optional, configurable count)
// - Distributor
// - Metrics Generator + Prometheus (optional)
// - Query Frontend + Querier (always started)
//
// Components started (single binary mode):
// - Object storage backend (S3/Azure/GCS - auto-detected from config, local backend is skipped)
// - Kafka
// - Tempo (single binary with all components)
//
// Example usage:
//
//	func TestMyFeature(t *testing.T) {
//		util.WithTempoHarness(t, util.TestHarnessConfig{
//			ConfigOverlay: "config-s3.yaml",
//			DeploymentMode: util.DeploymentModeMicroservices, // or DeploymentModeSingleBinary
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
func WithTempoHarness(t *testing.T, config TestHarnessConfig, testFunc func(*TempoHarness)) {
	t.Helper()

	// // If ComponentsTestAllBackends is set, run the test 3 times (once per backend)
	if config.Components&ComponentsObjectStorage != 0 {
		for _, b := range []string{backend.S3, backend.GCS, backend.Azure} { // jpe - restore backend.Azure. TestBackendScheduler hangs on azure?
			t.Run(b, func(t *testing.T) {

				// Run the test with the backend-specific config
				runTempoHarness(t, config, b, testFunc)
			})
		}
		return
	}

	// Run the test once with the provided config
	runTempoHarness(t, config, backend.Local, testFunc)
}

// runTempoHarness is the internal implementation that sets up and runs a single harness instance
func runTempoHarness(t *testing.T, harnessCfg TestHarnessConfig, requestedBackend string, testFunc func(*TempoHarness)) {
	t.Helper()

	if harnessCfg.Components == 0 {
		harnessCfg.Components = ComponentRecentDataQuerying
	}

	// Create scenario with normalized test name
	name := normalizeTestName(t.Name())
	s, err := e2e.NewScenario("e2e_" + name)
	require.NoError(t, err)
	defer s.Close()

	harness := &TempoHarness{
		Services:     map[string]*e2e.HTTPService{},
		TestScenario: s,
	}

	// Setup config and infrastructure
	tempoCfg := setupConfig(t, s, &harnessCfg, requestedBackend, harness)

	// Start object storage backend if not using local filesystem
	if tempoCfg.StorageConfig.Trace.Backend != backend.Local {
		harness.Backend, err = startBackend(t, s, tempoCfg)
		require.NoError(t, err, "failed to start backend")
	}

	// Start Kafka  jpe - 14:13:41 Error response from daemon: removal of container ae43fe6613da is already in progress ??
	if harnessCfg.Components&ComponentRecentDataQuerying != 0 {
		harness.Kafka = e2edb.NewKafka()
		require.NoError(t, s.StartAndWaitReady(harness.Kafka), "failed to start Kafka")
	}

	// Start Tempo services based on deployment mode
	if harnessCfg.DeploymentMode == DeploymentModeSingleBinary {
		require.NoError(t, startSingleBinary(t, s, harness, harnessCfg), "failed to start single binary")
	} else {
		// Default to microservices mode
		require.NoError(t, startMicroservices(t, s, harness, harnessCfg), "failed to start microservices")
	}

	if harnessCfg.Components&ComponentRecentDataQuerying != 0 { // jpe - test flag func, consolidate with logic in startMicroservices to set endpoints and clients?
		// Create HTTP client
		harness.HTTPClient = httpclient.New("http://"+harness.QueryFrontendHTTPEndpoint, "")

		// Create Jaeger to OTLP exporter - jpe - do we need both of these?
		harness.JaegerExporter, err = NewJaegerToOTLPExporter(harness.DistributorOTLPEndpoint)
		require.NoError(t, err, "failed to create Jaeger to OTLP exporter")
		require.NotNil(t, harness.JaegerExporter)

		// Create OTLP exporter (jpe - do we need both of these?)
		harness.OTLPExporter, err = NewOtelGRPCExporter(harness.DistributorOTLPEndpoint)
		require.NoError(t, err, "failed to create OTLP exporter")
		require.NotNil(t, harness.OTLPExporter)
	}

	// Run the test function
	testFunc(harness)
}

// normalizeTestName creates a valid Docker service name from a test name
func normalizeTestName(testName string) string {
	// max docker name length is 63. otherwise dns fails silently
	// max test name length is 40 to leave room prefix and suffix. the full container name will be e2e_<test name>_<service name>
	// this means that if two tests have the same first 40 characters in their names they will conflict!!
	maxNameLen := 40
	name := testName[len("Test"):] // strip "Test" prefix
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	// docker only allows a-zA-Z0-9_.- in a service name. replace everything else with _
	re := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	return re.ReplaceAllString(name, "_")
}

// setupConfig loads and merges config files, creates the overrides file, and validates the config
func setupConfig(t *testing.T, s *e2e.Scenario, config *TestHarnessConfig, requestedBackend string, harness *TempoHarness) app.Config {
	t.Helper()

	// Initialize template data if needed
	if config.ConfigTemplateData == nil {
		config.ConfigTemplateData = make(map[string]any)
	}

	// Call ConfigTemplateFunc if provided to populate template data
	if config.ConfigTemplateFunc != nil {
		err := config.ConfigTemplateFunc(s, config.ConfigTemplateData)
		require.NoError(t, err, "failed to execute config template function")
	}

	// Copy base config to shared directory
	baseConfigPath := "../util/config-base.yaml" // jpe - read from the context of the other folder. need to make these consts somewhere with a note
	err := CopyFileToSharedDir(s, baseConfigPath, "config.yaml")
	require.NoError(t, err, "failed to copy base config to shared dir")

	// Apply single binary specific config if in single binary mode
	if config.DeploymentMode == DeploymentModeSingleBinary {
		err := applyConfigOverlay(s, "../util/config-single-binary.yaml", nil)
		require.NoError(t, err, "failed to apply single binary config overlay")
	}

	// backend overlay
	if requestedBackend != backend.Local {
		backendOverlay := fmt.Sprintf("../util/config-backend-%s.yaml", requestedBackend)
		err := applyConfigOverlay(s, backendOverlay, nil)
		require.NoError(t, err, "failed to apply backend config overlay", requestedBackend)
	}

	// Apply config overlay if provided
	if config.ConfigOverlay != "" {
		err := applyConfigOverlay(s, config.ConfigOverlay, config.ConfigTemplateData)
		require.NoError(t, err, "failed to apply config overlay")
	}

	// Create empty overrides file
	overridesPath := s.SharedDir() + "/overrides.yaml"
	err = os.WriteFile(overridesPath, []byte("overrides: {}\n"), 0644)
	require.NoError(t, err, "failed to write initial overrides file")
	harness.overridesPath = overridesPath

	// Read and parse the final config
	configPath := s.SharedDir() + "/config.yaml" // jpe - make a shared func somewhere
	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err, "failed to read merged config file")

	var cfg app.Config
	err = yaml.UnmarshalStrict(configBytes, &cfg)
	require.NoError(t, err, "failed to unmarshal merged config into app.Config")

	return cfg
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

// applyConfigOverlay applies a config overlay file onto the shared config.yaml file,
// with optional template rendering. The overlay is merged onto the existing shared config
// and written back to shared config.yaml.
func applyConfigOverlay(s *e2e.Scenario, overlayPath string, templateData map[string]any) error {
	configPath := s.SharedDir() + "/config.yaml" // make a shared func somewhere

	// Read and parse current shared config
	baseBuff, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read shared config file: %w", err)
	}

	var baseMap map[any]any
	err = yaml.Unmarshal(baseBuff, &baseMap)
	if err != nil {
		return fmt.Errorf("failed to parse shared config file: %w", err)
	}

	// If there's an overlay, apply it
	if overlayPath != "" {
		// Read overlay file
		overlayBuff, err := os.ReadFile(overlayPath)
		if err != nil {
			return fmt.Errorf("failed to read config overlay file: %w", err)
		}

		// Apply template rendering if template data is provided
		if len(templateData) > 0 {
			tmpl, err := template.New("config").Parse(string(overlayBuff))
			if err != nil {
				return fmt.Errorf("failed to parse config overlay template: %w", err)
			}

			var renderedBuff bytes.Buffer
			err = tmpl.Execute(&renderedBuff, templateData)
			if err != nil {
				return fmt.Errorf("failed to execute config overlay template: %w", err)
			}

			overlayBuff = renderedBuff.Bytes()
		}

		// Parse overlay
		var overlayMap map[any]any
		err = yaml.Unmarshal(overlayBuff, &overlayMap)
		if err != nil {
			return fmt.Errorf("failed to parse config overlay file: %w", err)
		}

		// Merge overlay onto base
		baseMap = mergeMaps(baseMap, overlayMap)
	}

	// Marshal and write the result back to shared config
	outputBytes, err := yaml.Marshal(baseMap)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	err = os.WriteFile(configPath, outputBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
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

// GetConfig reads and parses the Tempo configuration file that was set up by the harness.
// Returns the parsed app.Config or an error if reading/parsing fails.
func (h *TempoHarness) GetConfig() (app.Config, error) {
	configPath := h.TestScenario.SharedDir() + "/config.yaml"

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

// RestartServiceWithConfigOverlay stops a service, applies a config overlay, and restarts the service.
// The overlay file is merged onto the existing config, with overlay values taking precedence.
//
// Example usage:
//
//	queryFrontend := h.Services[util.ServiceQueryFrontend]
//	err := h.RestartServiceWithConfigOverlay(queryFrontend, "./config-query-backend.yaml")
func (h *TempoHarness) RestartServiceWithConfigOverlay(t *testing.T, service *e2e.HTTPService, overlayPath string) error { // jpe -restart of single binary takes forever
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

	// reset endpoints and rebuild clients jpe - make helper func? - some of this doesn't make sense depending on the ComponentsMask.
	h.QueryFrontendHTTPEndpoint = h.Services[ServiceQueryFrontend].Endpoint(3200)
	h.QueryFrontendGRPCEndpoint = h.Services[ServiceQueryFrontend].Endpoint(3200)
	h.DistributorOTLPEndpoint = h.Services[ServiceDistributor].Endpoint(4317)

	// Create HTTP client
	h.HTTPClient = httpclient.New("http://"+h.QueryFrontendHTTPEndpoint, "")

	// Create Jaeger to OTLP exporter - jpe - do we need both of these?
	h.JaegerExporter, err = NewJaegerToOTLPExporter(h.DistributorOTLPEndpoint)
	require.NoError(t, err, "failed to create Jaeger to OTLP exporter")
	require.NotNil(t, h.JaegerExporter)

	// Create OTLP exporter (jpe - do we need both of these?)
	h.OTLPExporter, err = NewOtelGRPCExporter(h.DistributorOTLPEndpoint)
	require.NoError(t, err, "failed to create OTLP exporter")
	require.NotNil(t, h.OTLPExporter)

	return nil
}

// startMicroservices starts all Tempo microservices and waits for them to be ready - jpe - start multiple concurrently
func startMicroservices(t *testing.T, s *e2e.Scenario, harness *TempoHarness, config TestHarnessConfig) error {
	t.Helper()

	if config.Components&ComponentRecentDataQuerying != 0 {
		// Always start core components for recent data ingestion and querying
		// Start LiveStores
		liveStoreZoneA := NewNamedTempoLiveStore(
			"live-store-zone-a",
			0,
		)
		harness.Services[ServiceLiveStoreZoneA] = liveStoreZoneA
		if err := s.StartAndWaitReady(liveStoreZoneA); err != nil {
			return fmt.Errorf("failed to start live store zone a: %w", err)
		}

		liveStoreZoneB := NewNamedTempoLiveStore(
			"live-store-zone-b",
			0,
		)
		harness.Services[ServiceLiveStoreZoneB] = liveStoreZoneB
		if err := s.StartAndWaitReady(liveStoreZoneB); err != nil {
			return fmt.Errorf("failed to start live store zone b: %w", err)
		}

		// Wait for live stores to join the partition ring
		matchers := []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "state", Value: "Active"},
			{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
		}
		if err := liveStoreZoneA.WaitSumMetricsWithOptions(
			e2e.Equals(float64(1)),
			[]string{"tempo_partition_ring_partitions"},
			e2e.WithLabelMatchers(matchers...),
		); err != nil {
			return fmt.Errorf("live stores failed to join partition ring: %w", err)
		}

		// Start Distributor
		harness.Services[ServiceDistributor] = NewTempoDistributor()
		if err := s.StartAndWaitReady(harness.Services[ServiceDistributor]); err != nil {
			return fmt.Errorf("failed to start distributor: %w", err)
		}

		// Start Query Frontend and Querier
		harness.Services[ServiceQueryFrontend] = NewTempoQueryFrontend()
		harness.Services[ServiceQuerier] = NewTempoQuerier()
		if err := s.StartAndWaitReady(harness.Services[ServiceQueryFrontend], harness.Services[ServiceQuerier]); err != nil {
			return fmt.Errorf("failed to start query frontend and querier: %w", err)
		}

		// Set endpoints
		harness.QueryFrontendHTTPEndpoint = harness.Services[ServiceQueryFrontend].Endpoint(3200)
		harness.QueryFrontendGRPCEndpoint = harness.Services[ServiceQueryFrontend].Endpoint(3200)
		harness.DistributorOTLPEndpoint = harness.Services[ServiceDistributor].Endpoint(4317)
	}

	// Start optional components based on bitmask

	// Start Block Builder for backend work
	if config.Components&ComponentsBackendQuerying != 0 {
		blockBuilder := NewTempoBlockBuilder(0)
		harness.Services[ServiceBlockBuilder] = blockBuilder
		if err := s.StartAndWaitReady(blockBuilder); err != nil {
			return fmt.Errorf("failed to start block builder: %w", err)
		}
	}

	// Start Metrics Generator and Prometheus
	if config.Components&ComponentsMetricsGenerator != 0 {
		harness.Services[ServiceMetricsGenerator] = NewTempoMetricsGenerator()
		harness.Prometheus = NewPrometheus()
		if err := s.StartAndWaitReady(harness.Services[ServiceMetricsGenerator], harness.Prometheus); err != nil {
			return fmt.Errorf("failed to start metrics generator and prometheus: %w", err)
		}
	}

	// Start Backend Scheduler and Worker for compaction
	if config.Components&ComponentsBackendWork != 0 {
		scheduler := NewTempoTarget("backend-scheduler", "config.yaml")
		worker := NewTempoTarget("backend-worker", "config.yaml")
		harness.Services[ServiceBackendScheduler] = scheduler
		harness.Services[ServiceBackendWorker] = worker
		if err := s.StartAndWaitReady(scheduler, worker); err != nil {
			return fmt.Errorf("failed to start backend scheduler and worker: %w", err)
		}
	}

	return nil
}

// startSingleBinary starts Tempo as a single all-in-one binary and waits for it to be ready
func startSingleBinary(t *testing.T, s *e2e.Scenario, harness *TempoHarness, config TestHarnessConfig) error {
	t.Helper()

	// Create single binary service with custom readiness probe
	// Using port 3201 for readiness to avoid conflicts with main HTTP port
	tempo := NewTempoAllInOneWithReadinessProbe(
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
	)

	harness.Services[ServiceDistributor] = tempo
	harness.Services[ServiceQueryFrontend] = tempo
	harness.Services[ServiceQuerier] = tempo
	harness.Services[ServiceLiveStoreZoneA] = tempo
	harness.Services[ServiceLiveStoreZoneB] = tempo
	harness.Services[ServiceBlockBuilder] = tempo
	harness.Services[ServiceMetricsGenerator] = tempo
	harness.Services[ServiceBackendScheduler] = tempo
	harness.Services[ServiceBackendWorker] = tempo

	if err := s.StartAndWaitReady(tempo); err != nil {
		return fmt.Errorf("failed to start tempo single binary: %w", err)
	}

	// Wait for partition ring to be ready (same as microservices mode)
	matchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: "Active"},
		{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
	}
	if err := tempo.WaitSumMetricsWithOptions(
		e2e.Equals(float64(1)),
		[]string{"tempo_partition_ring_partitions"},
		e2e.WithLabelMatchers(matchers...),
	); err != nil {
		return fmt.Errorf("partition ring failed to become ready: %w", err)
	}

	// Set endpoints (all pointing to the same service)
	harness.QueryFrontendHTTPEndpoint = tempo.Endpoint(3200)
	harness.QueryFrontendGRPCEndpoint = tempo.Endpoint(3200)
	harness.DistributorOTLPEndpoint = tempo.Endpoint(4317)

	return nil
}
