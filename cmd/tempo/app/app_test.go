package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	userconfigurableoverrides "github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestStatusConfigDoesNotDiscloseSecretPolicy(t *testing.T) {
	const private = "synthetic-private-policy-marker"
	cfg := NewDefaultConfig()
	policy := &secrets.Policy{CustomRules: []secrets.CustomRule{{ID: private, Regex: private}}}
	cfg.Overrides.Defaults.MetricsGenerator.Processor.SecretDetection = policy
	app := &App{cfg: *cfg}
	for _, mode := range []string{"", "diff", "defaults"} {
		t.Run("mode-"+mode, func(t *testing.T) {
			var out bytes.Buffer
			err := app.writeStatusConfig(&out, httptest.NewRequest(http.MethodGet, "/status/config?mode="+mode, nil))
			require.True(t, err == nil, "status config must serialize safely")
			require.False(t, strings.Contains(out.String(), private), "status config disclosed private policy")
			var decoded map[string]any
			require.True(t, yaml.Unmarshal(out.Bytes(), &decoded) == nil, "status config must remain valid YAML")
			require.True(t, reflect.DeepEqual(policy, app.cfg.Overrides.Defaults.MetricsGenerator.Processor.SecretDetection), "status must not mutate the configured policy")
		})
	}
	// Persistence encodes tenant limits, not unrelated App fields such as server callbacks.
	limits := userconfigurableoverrides.Limits{MetricsGenerator: userconfigurableoverrides.LimitsMetricsGenerator{
		Processor: userconfigurableoverrides.LimitsMetricsGeneratorProcessor{
			SecretDetection: app.cfg.Overrides.Defaults.MetricsGenerator.Processor.SecretDetection,
		},
	}}
	for _, codec := range []struct {
		name      string
		marshal   func(any) ([]byte, error)
		unmarshal func([]byte, any) error
	}{
		{"json", json.Marshal, json.Unmarshal},
		{"yaml", yaml.Marshal, yaml.Unmarshal},
	} {
		t.Run("persistence-"+codec.name, func(t *testing.T) {
			data, err := codec.marshal(limits)
			require.True(t, err == nil, "configured policy must remain serializable")
			require.True(t, bytes.Contains(data, []byte(private)), "normal config serialization must preserve private policy")
			var restored userconfigurableoverrides.Limits
			require.True(t, codec.unmarshal(data, &restored) == nil, "persisted policy must remain readable")
			require.True(t, reflect.DeepEqual(limits, restored), "persistence must preserve the complete configured policy")
		})
	}
}

func TestAppNewSetsStorageBlockFormatUsageStat(t *testing.T) {
	config := NewDefaultConfig()
	config.StorageConfig.Trace.Block.Version = "vParquet4"

	_, err := New(*config)
	require.NoError(t, err)

	report := usagestats.BuildStats()
	require.Equal(t, "vParquet4", report.Metrics["storage_block_format"])
}

func TestAppNewAppliesNativeRuleSelection(t *testing.T) {
	for _, tc := range []struct {
		name string
		ids  *[]string
		want []secrets.Match
	}{
		{"omitted", nil, []secrets.Match{{RuleID: "slack-bot-token"}, {RuleID: "stripe-access-token"}}},
		{"selected", &[]string{"stripe-access-token"}, []secrets.Match{{RuleID: "stripe-access-token"}}},
		{"empty", &[]string{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := NewDefaultConfig()
			config.Secrets = secrets.FeatureConfig{DetectionEnabled: true, EnabledRules: tc.ids}
			app, err := New(*config)
			require.NoError(t, err)
			compiled, err := app.cfg.Generator.Processor.SecretDetection.PolicyCompiler.CompilePolicy(secrets.Policy{})
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, compiled.Detect("sk_test_"+"0123456789abcdefghijklmn"+"\n"+"xoxb-"+"1234567890-1234567890123-abcdefghijklmnopqrstuvwx").Matches)
		})
	}
}

func TestAppNewRejectsUnknownGlobalRulesEvenWhenDisabled(t *testing.T) {
	const privateID = "private-operator-content"
	for _, enabled := range []bool{false, true} {
		config := NewDefaultConfig()
		config.Secrets = secrets.FeatureConfig{DetectionEnabled: enabled, EnabledRules: &[]string{privateID}}
		app, err := New(*config)
		require.Error(t, err)
		require.Nil(t, app)
		require.NotContains(t, err.Error(), privateID)
	}
}

func TestApp_RunStop(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tempo-test-app-*")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(tempDir)
		require.NoError(t, err)
	}()

	config := NewDefaultConfig()
	config.Target = BackendScheduler
	config.Server.HTTPListenPort = util.MustGetFreePort()
	config.Server.GRPCListenPort = util.MustGetFreePort() // not used in the test; set to ensure conflict-free start
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
	healthCheckURL := fmt.Sprintf("http://localhost:%d/ready", config.Server.HTTPListenPort)
	require.Eventually(t, func() bool {
		t.Log("Checking Tempo is up...")
		// #nosec G107 -- nosemgrep: tainted-url-host
		resp, httpErr := http.Get(healthCheckURL)
		return httpErr == nil && resp.StatusCode == http.StatusOK
	}, 30*time.Second, 1*time.Second)

	// stop Tempo
	app.Stop()

	// check health endpoint is not reachable anymore
	require.Eventually(t, func() bool {
		t.Log("Checking Tempo is down...")
		// #nosec G107 -- nosemgrep: tainted-url-host
		_, httpErr := http.Get(healthCheckURL)
		return httpErr != nil
	}, 60*time.Second, 1*time.Second)
}
