package storage

import (
	"log/slog"
	"net/url"
	"os"
	"testing"

	prometheus_common_config "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	prometheus_config "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/util"
)

func Test_generateTenantRemoteWriteConfigs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	original := []prometheus_config.RemoteWriteConfig{
		{
			URL:     &prometheus_common_config.URL{URL: urlMustParse("http://prometheus-1/api/prom/push")},
			Headers: map[string]string{},
		},
		{
			URL: &prometheus_common_config.URL{URL: urlMustParse("http://prometheus-2/api/prom/push")},
			Headers: map[string]string{
				"foo":           "bar",
				"x-scope-orgid": "fake-tenant",
			},
		},
	}

	addOrgIDHeader := true

	result := generateTenantRemoteWriteConfigs(original, "my-tenant", nil, addOrgIDHeader, logger, false)

	// First case doesn't have a header, gets set.
	assert.Equal(t, original[0].URL, result[0].URL)
	assert.Equal(t, map[string]string{}, original[0].Headers, "Original headers have been modified")
	assert.Equal(t, map[string]string{"X-Scope-OrgID": "my-tenant"}, result[0].Headers)

	// Second case already contains header, not overwritten
	// Also checks case-insensitivity
	assert.Equal(t, original[1].URL, result[1].URL)
	assert.Equal(t, map[string]string{"foo": "bar", "x-scope-orgid": "fake-tenant"}, original[1].Headers, "Original headers have been modified")
	assert.Equal(t, map[string]string{"foo": "bar", "x-scope-orgid": "fake-tenant"}, result[1].Headers, "Existing header was incorrectly overwritten")
}

func Test_generateTenantRemoteWriteConfigs_singleTenant(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	original := []prometheus_config.RemoteWriteConfig{
		{
			URL:     &prometheus_common_config.URL{URL: urlMustParse("http://prometheus-1/api/prom/push")},
			Headers: map[string]string{},
		},
		{
			URL: &prometheus_common_config.URL{URL: urlMustParse("http://prometheus-2/api/prom/push")},
			Headers: map[string]string{
				"x-scope-orgid": "my-custom-tenant-id",
			},
		},
	}

	addOrgIDHeader := true

	result := generateTenantRemoteWriteConfigs(original, util.FakeTenantID, nil, addOrgIDHeader, logger, false)

	assert.Equal(t, original[0].URL, result[0].URL)

	assert.Equal(t, original[0].URL, result[0].URL)
	assert.Equal(t, map[string]string{}, original[0].Headers, "Original headers have been modified")
	// X-Scope-OrgID has not been injected
	assert.Equal(t, map[string]string{}, result[0].Headers)

	assert.Equal(t, original[1].URL, result[1].URL)
	assert.Equal(t, map[string]string{"x-scope-orgid": "my-custom-tenant-id"}, original[1].Headers, "Original headers have been modified")
	// X-Scope-OrgID has not been modified
	assert.Equal(t, map[string]string{"x-scope-orgid": "my-custom-tenant-id"}, result[1].Headers)
}

func Test_generateTenantRemoteWriteConfigs_addOrgIDHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	original := []prometheus_config.RemoteWriteConfig{
		{
			URL:     &prometheus_common_config.URL{URL: urlMustParse("http://prometheus-1/api/prom/push")},
			Headers: map[string]string{},
		},
		{
			URL: &prometheus_common_config.URL{URL: urlMustParse("http://prometheus-2/api/prom/push")},
			Headers: map[string]string{
				"foo":           "bar",
				"x-scope-orgid": "fake-tenant",
			},
		},
	}

	addOrgIDHeader := false

	result := generateTenantRemoteWriteConfigs(original, "my-tenant", nil, addOrgIDHeader, logger, false)

	assert.Equal(t, original[0].URL, result[0].URL)
	assert.Empty(t, original[0].Headers, "X-Scope-OrgID header is not added")

	assert.Equal(t, original[1].URL, result[1].URL)
	assert.Equal(t, map[string]string{"foo": "bar", "x-scope-orgid": "fake-tenant"}, result[1].Headers, "Original headers not modified")
}

func Test_generateTenantRemoteWriteConfigs_sendNativeHistograms(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	original := []prometheus_config.RemoteWriteConfig{
		{
			URL:     &prometheus_common_config.URL{URL: urlMustParse("http://prometheus-1/api/prom/push")},
			Headers: map[string]string{},
		},
	}

	result := generateTenantRemoteWriteConfigs(original, "my-tenant", nil, false, logger, true)
	assert.Equal(t, true, result[0].SendNativeHistograms, "SendNativeHistograms should be true")

	result = generateTenantRemoteWriteConfigs(original, "my-tenant", nil, false, logger, false)
	assert.Equal(t, false, result[0].SendNativeHistograms, "SendNativeHistograms should be true")
}

func Test_copyMap(t *testing.T) {
	original := map[string]string{
		"k1": "v1",
		"k2": "v2",
	}

	copied := copyMap(original)

	assert.Equal(t, original, copied)

	copied["k2"] = "other value"
	copied["k3"] = "v3"

	assert.Len(t, original, 2)
	assert.Equal(t, "v2", original["k2"])
	assert.Equal(t, "", original["k3"])
}

func Test_generateTenantRemoteWriteConfigs_writeRelabelConfigs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := Config{
		RemoteWrite: []prometheus_config.RemoteWriteConfig{
			{
				URL:     &prometheus_common_config.URL{URL: urlMustParse("http://prometheus-1/api/prom/push")},
				Headers: map[string]string{},
				WriteRelabelConfigs: []*relabel.Config{
					{
						SourceLabels: model.LabelNames{"deployment_environment"},
						Separator:    relabel.DefaultRelabelConfig.Separator,
						Regex:        relabel.DefaultRelabelConfig.Regex,
						Replacement:  relabel.DefaultRelabelConfig.Replacement,
						Action:       relabel.Replace,
						TargetLabel:  "client_deployment_environment",
					},
				},
			},
		},
	}

	// Validate once at startup, as the generator does in Config.Validate().
	require.NoError(t, cfg.Validate())

	result := generateTenantRemoteWriteConfigs(cfg.RemoteWrite, "my-tenant", nil, false, logger, false)

	// Simulate what the Prometheus remote write queue manager does: run the
	// relabel configs against a label set. Before the fix this panics with
	// "Invalid name validation scheme requested: unset" because
	// NameValidationScheme was never initialized on the relabel configs.
	lbls := labels.FromStrings("deployment_environment", "production")

	var newLbls labels.Labels
	var keep bool
	assert.NotPanics(t, func() {
		newLbls, keep = relabel.Process(lbls, result[0].WriteRelabelConfigs...)
	})
	assert.True(t, keep)
	assert.Equal(t, "production", newLbls.Get("client_deployment_environment"))
}

func urlMustParse(urlStr string) *url.URL {
	url, err := url.Parse(urlStr)
	if err != nil {
		panic(err)
	}
	return url
}
