package ingest

import (
	"flag"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	yamlv2 "go.yaml.in/yaml/v2"
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
		Address:                          KafkaAddresses{cluster.ListenAddrs()[0]},
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
		Address:                    KafkaAddresses{"localhost:9092"},
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

func TestKafkaAddresses(t *testing.T) {
	t.Run("flag default is a single localhost broker", func(t *testing.T) {
		var cfg KafkaConfig
		f := flag.NewFlagSet("test", flag.PanicOnError)
		cfg.RegisterFlags(f)

		require.Equal(t, KafkaAddresses{"localhost:9092"}, cfg.Address)
		require.NoError(t, f.Parse(nil))
		require.Equal(t, KafkaAddresses{"localhost:9092"}, cfg.Address)
	})

	t.Run("flag replaces the default with a comma-separated list", func(t *testing.T) {
		var cfg KafkaConfig
		f := flag.NewFlagSet("test", flag.PanicOnError)
		cfg.RegisterFlags(f)

		require.NoError(t, f.Parse([]string{"-kafka.address=kafka-1:9092, kafka-2:9092"}))
		require.Equal(t, KafkaAddresses{"kafka-1:9092", "kafka-2:9092"}, cfg.Address)
	})

	tests := []struct {
		name string
		yaml string
		want KafkaAddresses
	}{
		{
			name: "single string",
			yaml: "address: kafka-1:9092\n",
			want: KafkaAddresses{"kafka-1:9092"},
		},
		{
			name: "comma-separated string",
			yaml: "address: kafka-1:9092, kafka-2:9092\n",
			want: KafkaAddresses{"kafka-1:9092", "kafka-2:9092"},
		},
		{
			name: "yaml list",
			yaml: "address:\n  - kafka-1:9092\n  - kafka-2:9092\n",
			want: KafkaAddresses{"kafka-1:9092", "kafka-2:9092"},
		},
	}
	for _, tc := range tests {
		t.Run("yaml "+tc.name, func(t *testing.T) {
			var cfg KafkaConfig
			require.NoError(t, yamlv2.Unmarshal([]byte(tc.yaml), &cfg))
			require.Equal(t, tc.want, cfg.Address)
		})
	}
}

func TestKafkaConfig_Validate_Address(t *testing.T) {
	cfg := KafkaConfig{
		Address:                    KafkaAddresses{"localhost:9092"},
		Topic:                      "test",
		ProducerMaxRecordSizeBytes: minProducerRecordDataBytesLimit,
	}
	require.NoError(t, cfg.Validate())

	cfg.Address = nil
	require.ErrorIs(t, cfg.Validate(), ErrMissingKafkaAddress)

	cfg.Address = KafkaAddresses{}
	require.ErrorIs(t, cfg.Validate(), ErrMissingKafkaAddress)
}

func TestCommonKafkaClientOptions_MultipleSeedBrokers(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.NumBrokers(2))
	require.NoError(t, err)
	t.Cleanup(cluster.Close)

	cfg := KafkaConfig{
		Address: KafkaAddresses(cluster.ListenAddrs()),
		Topic:   "test",
	}
	opts, err := commonKafkaClientOptions(cfg, nil, log.NewNopLogger())
	require.NoError(t, err)

	client, err := kgo.NewClient(opts...)
	require.NoError(t, err)
	t.Cleanup(client.Close)
}
