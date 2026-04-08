package ingest

import (
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kmsg"
	"go.yaml.in/yaml/v2"
)

func TestConfigUnmarshalYAMLRejectsLegacyEnabled(t *testing.T) {
	var cfg Config

	err := yaml.UnmarshalStrict([]byte(`
enabled: false
kafka:
  address: localhost:9092
  topic: tempo
`), &cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "field enabled not found")
}

func TestConfigUnmarshalYAMLStrictUnknownField(t *testing.T) {
	var cfg Config

	err := yaml.UnmarshalStrict([]byte(`
kafka:
  address: localhost:9092
  topic: tempo
unknown: true
`), &cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "field unknown not found")
}

func TestConfigUnmarshalYAMLPreservesDefaults(t *testing.T) {
	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("ingest", flag.NewFlagSet("", flag.PanicOnError))

	require.NoError(t, yaml.UnmarshalStrict([]byte(`
kafka:
  topic: tempo
`), &cfg))
	require.Equal(t, "localhost:9092", cfg.Kafka.Address)
	require.Equal(t, "tempo", cfg.Kafka.Topic)
	require.Equal(t, 2*time.Second, cfg.Kafka.DialTimeout)
}

func TestSetDefaultNumberOfPartitionsForAutocreatedTopics(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.NumBrokers(1))
	require.NoError(t, err)
	t.Cleanup(cluster.Close)

	addrs := cluster.ListenAddrs()
	require.Len(t, addrs, 1)

	cfg := KafkaConfig{
		Address:                          addrs[0],
		AutoCreateTopicDefaultPartitions: 100,
	}

	cluster.ControlKey(kmsg.AlterConfigs.Int16(), func(request kmsg.Request) (kmsg.Response, error, bool) {
		r := request.(*kmsg.AlterConfigsRequest)

		require.Len(t, r.Resources, 1)
		res := r.Resources[0]
		require.Equal(t, kmsg.ConfigResourceTypeBroker, res.ResourceType)
		require.Len(t, res.Configs, 1)
		cfg := res.Configs[0]
		require.Equal(t, "num.partitions", cfg.Name)
		require.NotNil(t, *cfg.Value)
		require.Equal(t, "100", *cfg.Value)

		return &kmsg.AlterConfigsResponse{}, nil, true
	})

	cfg.SetDefaultNumberOfPartitionsForAutocreatedTopics(log.NewNopLogger())
}
