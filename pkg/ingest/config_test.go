package ingest

import (
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/v3/pkg/util/test"
)

func TestKafkaConfig_ClientRackFlag(t *testing.T) {
	var cfg KafkaConfig
	f := flag.NewFlagSet("test", flag.PanicOnError)
	cfg.RegisterFlags(f)

	// Defaults to empty so rack-aware fetching is opt-in.
	require.Empty(t, cfg.ClientRack)

	fl := f.Lookup("kafka.client-rack")
	require.NotNil(t, fl, "kafka.client-rack flag should be registered")
	require.Equal(t, "", fl.DefValue)

	require.NoError(t, f.Parse([]string{"-kafka.client-rack=us-east-1a"}))
	require.Equal(t, "us-east-1a", cfg.ClientRack)
}

func TestSetDefaultNumberOfPartitionsForAutocreatedTopics(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.NumBrokers(1))
	require.NoError(t, err)
	t.Cleanup(cluster.Close)

	client, err := kgo.NewClient(kgo.SeedBrokers(cluster.ListenAddrs()...))
	require.NoError(t, err)
	adm := kadm.NewClient(client)
	t.Cleanup(adm.Close)

	retention := "5256000"
	responses, err := adm.AlterBrokerConfigs(t.Context(), []kadm.AlterConfig{
		{Op: kadm.SetConfig, Name: "offsets.retention.minutes", Value: &retention},
	})
	require.NoError(t, err)
	for _, response := range responses {
		require.NoError(t, response.Err)
	}

	cfg := KafkaConfig{
		Address:                          cluster.ListenAddrs()[0],
		AutoCreateTopicDefaultPartitions: 100,
	}

	cfg.SetDefaultNumberOfPartitionsForAutocreatedTopics(log.NewNopLogger())

	configs, err := adm.DescribeBrokerConfigs(t.Context())
	require.NoError(t, err)
	brokerConfig, err := configs.On("", nil)
	require.NoError(t, err)
	require.NoError(t, brokerConfig.Err)

	values := make(map[string]string, len(brokerConfig.Configs))
	for _, config := range brokerConfig.Configs {
		values[config.Key] = config.MaybeValue()
	}
	require.Equal(t, "100", values["num.partitions"])
	require.Equal(t, retention, values["offsets.retention.minutes"],
		"setting default partitions must preserve unrelated broker configuration")
}

func TestParseProducerCompression(t *testing.T) {
	tests := map[string]struct {
		value     string
		expectErr bool
	}{
		"empty is valid (leaves client default unchanged)":           {value: ""},
		"whitespace-only is valid (leaves client default unchanged)": {value: "   "},
		"none is valid":             {value: compressionNone},
		"gzip is valid":             {value: compressionGzip},
		"snappy is valid":           {value: compressionSnappy},
		"lz4 is valid":              {value: compressionLz4},
		"zstd is valid":             {value: compressionZstd},
		"is case-insensitive":       {value: "GZIP"},
		"invalid value is rejected": {value: "unsupported", expectErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := parseProducerCompression(tc.value)
			if tc.expectErr {
				require.ErrorIs(t, err, ErrInvalidProducerCompression)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseProducerCompression_UnsetVsExplicitNone(t *testing.T) {
	// Empty and whitespace only values must return set=false, so callers know to leave the
	// Kafka client's default codec preference unchanged rather than forcing no compression.
	for _, value := range []string{"", "   "} {
		_, set, err := parseProducerCompression(value)
		require.NoError(t, err)
		require.False(t, set, "value %q should be treated as unset", value)
	}

	// An explicit "none" must still return set=true so the caller applies it.
	codec, set, err := parseProducerCompression(compressionNone)
	require.NoError(t, err)
	require.True(t, set)
	require.Equal(t, kgo.NoCompression(), codec)
}

func TestKafkaConfig_Validate_ProducerCompression(t *testing.T) {
	cfg := KafkaConfig{
		Address:                    "localhost:9092",
		Topic:                      "test",
		ProducerMaxRecordSizeBytes: minProducerRecordDataBytesLimit,
	}

	cfg.ProducerCompression = compressionGzip
	require.NoError(t, cfg.Validate())

	// validate that a whitespace only value is treated as unset rather than
	// rejected or coerced into an explicit codec.
	cfg.ProducerCompression = "   "
	require.NoError(t, cfg.Validate())

	// validate that an invalid value raises an error.
	cfg.ProducerCompression = "unsupported"
	require.ErrorIs(t, cfg.Validate(), ErrInvalidProducerCompression)
}

func TestKafkaConfig_ProducerLingerFlag(t *testing.T) {
	var cfg KafkaConfig
	f := flag.NewFlagSet("test", flag.PanicOnError)
	cfg.RegisterFlags(f)

	// The default is what the writer client used before it became configurable.
	fl := f.Lookup("kafka.producer-linger")
	require.NotNil(t, fl, "kafka.producer-linger flag should be registered")
	require.Equal(t, "50ms", fl.DefValue)
	require.Equal(t, 50*time.Millisecond, cfg.ProducerLinger)

	// Zero is a value, not "unset": a caller that waits for every Produce
	// acknowledgement wants no buffering at all.
	require.NoError(t, f.Parse([]string{"-kafka.producer-linger=0"}))
	require.Equal(t, time.Duration(0), cfg.ProducerLinger)
}

// The config file overlays the flag defaults, so an explicit zero has to survive
// the decode instead of being read as "unset".
func TestKafkaConfig_ProducerLingerYAML(t *testing.T) {
	var cfg KafkaConfig
	cfg.RegisterFlags(flag.NewFlagSet("test", flag.PanicOnError))
	require.Equal(t, 50*time.Millisecond, cfg.ProducerLinger)

	require.NoError(t, yaml.UnmarshalStrict([]byte("producer_linger: 0s\n"), &cfg))
	require.Equal(t, time.Duration(0), cfg.ProducerLinger)

	var untouched KafkaConfig
	untouched.RegisterFlags(flag.NewFlagSet("test", flag.PanicOnError))
	require.NoError(t, yaml.UnmarshalStrict([]byte("topic: traces\n"), &untouched))
	require.Equal(t, 50*time.Millisecond, untouched.ProducerLinger)
}

// The writer client is the only thing that makes the setting do something.
func TestNewWriterClient_ProducerLinger(t *testing.T) {
	for _, linger := range []time.Duration{0, 7 * time.Millisecond} {
		var cfg KafkaConfig
		cfg.RegisterFlags(flag.NewFlagSet("test", flag.PanicOnError))
		cfg.Topic = "traces"
		cfg.ProducerLinger = linger

		client, err := NewWriterClient(
			cfg,
			10,
			test.NewTestingLogger(t),
			prometheus.NewPedanticRegistry(),
		)
		require.NoError(t, err)
		require.Equal(t, linger, client.OptValue(kgo.ProducerLinger))
		client.Close()
	}
}
