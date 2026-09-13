package client

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

func TestUserConfigOverridesClient(t *testing.T) {
	// Other backends are tested through the integration tests

	ctx := context.Background()
	tenant := "foo"
	dir := t.TempDir()

	cfg := &Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: dir,
		},
	}

	client, err := New(cfg)
	require.NoError(t, err)

	// List
	list, err := client.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, list)

	// Set
	limits := &Limits{
		Forwarders: &[]string{"my-forwarder"},
	}
	_, err = client.Set(ctx, tenant, limits, "")
	assert.NoError(t, err)

	// Get
	retrievedLimits, _, err := client.Get(ctx, tenant)
	assert.NoError(t, err)
	assert.Equal(t, limits, retrievedLimits)

	// List
	list, err = client.List(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []string{tenant}, list)

	// Delete
	assert.NoError(t, client.Delete(ctx, tenant, ""))

	// Get - does not exist
	_, _, err = client.Get(ctx, tenant)
	assert.ErrorIs(t, err, backend.ErrDoesNotExist)
}

func TestUserConfigOverridesClientPersistedDocumentCompatibility(t *testing.T) {
	dir := t.TempDir()
	c, err := New(&Config{Backend: backend.Local, Local: &local.Config{Path: dir}})
	require.NoError(t, err)
	t.Cleanup(c.Shutdown)
	tenantDir := filepath.Join(dir, OverridesKeyPath, "tenant")
	require.NoError(t, os.MkdirAll(tenantDir, 0o700))
	const private = "synthetic-private-policy-marker"
	document := `{"forwarders":["updated"],"future_root":{"private":"` + private + `"},"metrics_generator":{"future_generator":true,"processor":{"future_processor":true,"service_graphs":{"dimensions":["http.method"],"future_service_graphs":true},"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"` + private + `"}]}}}}`
	want := &Limits{
		Forwarders: &[]string{"updated"},
		MetricsGenerator: LimitsMetricsGenerator{Processor: LimitsMetricsGeneratorProcessor{
			ServiceGraphs:   LimitsMetricsGeneratorProcessorServiceGraphs{Dimensions: &[]string{"http.method"}},
			SecretDetection: &secrets.Policy{CustomRules: []secrets.CustomRule{{ID: "safe-rule", Regex: private}}},
		}},
	}
	for _, tc := range []struct {
		name   string
		body   string
		reject bool
	}{
		{"forward-version-fields", document, false},
		{"trailing-whitespace", document + " \n\t", false},
		{"second-document", document + `{}`, true},
		{"trailing-null", document + `null`, true},
		{"malformed-suffix", document + private, true},
		{"malformed-policy", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":"` + private + `"}}}}`, true},
		{"legacy-policy", `{"metrics_generator":{"processor":{"secret_detection":{"optional_rules":[]}}}}`, true},
		{"unknown-policy-field", `{"metrics_generator":{"processor":{"secret_detection":{"` + private + `":true}}}}`, true},
		{"unknown-rule-field", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"safe","` + private + `":true}]}}}}`, true},
		{"removed-description", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"` + private + `","description":"` + private + `"}]}}}}`, true},
		{"removed-entropy", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"` + private + `","entropy":1.5}]}}}}`, true},
		{"removed-secret-group", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"(` + private + `)","secret_group":1}]}}}}`, true},
		{"unsupported-policy-before-empty", `{"metrics_generator":{"processor":{"secret_detection":{"optional_rules":[]},"secret_detection":{}}}}`, true},
		{"malformed-policy-syntax", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":["` + private + `",]}}}}`, true},
		{"escaped-policy-with-suffix", `{"metrics_generator":{"processor":{"secret_\u0064etection":{}}}} {}`, true},
		{"case-folded-policy-with-suffix", `{"metrics_generator":{"processor":{"SECRET_DETECTION":{}}}} {}`, true},
		{"null-policy-with-suffix", `{"metrics_generator":{"processor":{"secret_detection":null}}}} {}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(filepath.Join(tenantDir, OverridesFileName), []byte(tc.body), 0o600))
			limits, _, err := c.Get(context.Background(), "tenant")
			if tc.reject {
				require.True(t, errors.Is(err, ErrInvalidSecretsPolicy), "invalid persisted policy must return a safe policy error")
				require.True(t, limits == nil, "invalid document must not return publishable limits")
				require.False(t, strings.Contains(err.Error(), private), "parse error disclosed policy text")
				return
			}
			require.True(t, err == nil, "forward-compatible persisted document must load")
			require.True(t, reflect.DeepEqual(want, limits), "known fields and complete policy must survive decoding")
		})
	}
	t.Run("non-secret-decoder-compatibility", func(t *testing.T) {
		body := []byte(`{"forwarders":["legacy","secret_detection"]} {"forwarders":["later"]}`)
		require.NoError(t, os.WriteFile(filepath.Join(tenantDir, OverridesFileName), body, 0o600))
		limits, _, err := c.Get(context.Background(), "tenant")
		require.NoError(t, err)
		require.Equal(t, &Limits{Forwarders: &[]string{"legacy", "secret_detection"}}, limits)
	})
	t.Run("ordinary-decode-failure", func(t *testing.T) {
		for _, body := range []string{
			`{"forwarders":123}`,
			`{"forwarders":123,"metrics_generator":{"processor":{"secret_detection":{}}}}`,
		} {
			require.NoError(t, os.WriteFile(filepath.Join(tenantDir, OverridesFileName), []byte(body), 0o600))
			limits, _, err := c.Get(context.Background(), "tenant")
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrInvalidSecretsPolicy)
			require.Nil(t, limits)
		}
	})
}
