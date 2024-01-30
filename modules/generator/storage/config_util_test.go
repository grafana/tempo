package storage

import (
	"net/url"
	"os"
	"testing"

	"github.com/go-kit/log"
	prometheus_common_config "github.com/prometheus/common/config"
	prometheus_config "github.com/prometheus/prometheus/config"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/util"
)

func Test_generateTenantRemoteWriteConfigs(t *testing.T) {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

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

	result := generateTenantRemoteWriteConfigs(original, "my-tenant", nil, addOrgIDHeader, logger)

	assert.Equal(t, original[0].URL, result[0].URL)
	assert.Equal(t, map[string]string{}, original[0].Headers, "Original headers have been modified")
	assert.Equal(t, map[string]string{"X-Scope-OrgID": "my-tenant"}, result[0].Headers)

	assert.Equal(t, original[1].URL, result[1].URL)
	assert.Equal(t, map[string]string{"foo": "bar", "x-scope-orgid": "fake-tenant"}, original[1].Headers, "Original headers have been modified")
	assert.Equal(t, map[string]string{"foo": "bar", "X-Scope-OrgID": "my-tenant"}, result[1].Headers)
}

func Test_generateTenantRemoteWriteConfigs_singleTenant(t *testing.T) {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

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

	result := generateTenantRemoteWriteConfigs(original, util.FakeTenantID, nil, addOrgIDHeader, logger)

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
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

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

	result := generateTenantRemoteWriteConfigs(original, "my-tenant", nil, addOrgIDHeader, logger)

	assert.Equal(t, original[0].URL, result[0].URL)
	assert.Empty(t, original[0].Headers, "X-Scope-OrgID header is not added")

	assert.Equal(t, original[1].URL, result[1].URL)
	assert.Equal(t, map[string]string{"foo": "bar", "x-scope-orgid": "fake-tenant"}, result[1].Headers, "Original headers not modified")
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

func urlMustParse(urlStr string) *url.URL {
	url, err := url.Parse(urlStr)
	if err != nil {
		panic(err)
	}
	return url
}
