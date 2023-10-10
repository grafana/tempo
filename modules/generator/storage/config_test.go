package storage

import (
	"net/url"
	"testing"
	"time"

	prometheus_common_config "github.com/prometheus/common/config"
	prometheus_config "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/tsdb/wlog"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestConfig(t *testing.T) {
	cfgStr := `
path: /var/wal/tempo
wal:
  wal_compression: true
remote_write_flush_deadline: 5m
remote_write:
  - url: http://prometheus/api/prom/push
    headers:
      foo: bar
`

	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	err := yaml.UnmarshalStrict([]byte(cfgStr), &cfg)
	assert.NoError(t, err)

	walCfg := agentDefaultOptions()
	walCfg.WALCompression = wlog.CompressionSnappy

	remoteWriteConfig := prometheus_config.DefaultRemoteWriteConfig
	prometheusURL, err := url.Parse("http://prometheus/api/prom/push")
	assert.NoError(t, err)
	remoteWriteConfig.URL = &prometheus_common_config.URL{URL: prometheusURL}
	remoteWriteConfig.Headers = map[string]string{
		"foo": "bar",
	}

	expectedCfg := Config{
		Path:                      "/var/wal/tempo",
		Wal:                       walCfg,
		RemoteWriteFlushDeadline:  5 * time.Minute,
		RemoteWriteAddOrgIDHeader: true,
		RemoteWrite: []prometheus_config.RemoteWriteConfig{
			remoteWriteConfig,
		},
	}
	assert.Equal(t, expectedCfg, cfg)
}
