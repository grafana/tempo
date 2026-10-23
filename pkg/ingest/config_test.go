package ingest

import (
	"flag"
	"slices"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.yaml.in/yaml/v2"
)

func TestKafkaConfig_AddressList(t *testing.T) {
	tests := []struct {
		name            string
		yaml            string
		flags           []string
		expectedBrokers []string
	}{
		{
			name:            "default address",
			expectedBrokers: []string{"localhost:9092"},
		},
		{
			name:            "legacy address",
			yaml:            "address: legacy:9092",
			expectedBrokers: []string{"legacy:9092"},
		},
		{
			name:            "single address in AddressList",
			yaml:            "address_list: broker-one:9092",
			expectedBrokers: []string{"broker-one:9092"},
		},
		{
			name:            "AddressList overrides legacy address",
			yaml:            "address: ignored:invalid\naddress_list: 'broker-one:9092, broker-two:9093'",
			expectedBrokers: []string{"broker-one:9092", "broker-two:9093"},
		},
		{
			name:            "AddressList overrides empty legacy address",
			yaml:            "address: ''\naddress_list: 'broker-one:9092, broker-two:9093'",
			expectedBrokers: []string{"broker-one:9092", "broker-two:9093"},
		},
		{
			name:            "empty AddressList falls back to legacy address",
			yaml:            "address: legacy:9092\naddress_list: ''",
			expectedBrokers: []string{"legacy:9092"},
		},
		{
			name:            "CSV flag",
			flags:           []string{"-ingest.address-list=broker-one:9092, broker-two:9093"},
			expectedBrokers: []string{"broker-one:9092", "broker-two:9093"},
		},
		{
			name:            "last AddressList flag wins",
			flags:           []string{"-ingest.address-list=broker-one:9092", "-ingest.address-list=broker-two:9093"},
			expectedBrokers: []string{"broker-two:9093"},
		},
		{
			name:            "AddressList flag overrides YAML",
			yaml:            "address_list: ignored:9092",
			flags:           []string{"-ingest.address-list=broker-one:9092, broker-two:9093"},
			expectedBrokers: []string{"broker-one:9092", "broker-two:9093"},
		},
		{
			name:            "empty AddressList flag falls back to legacy address",
			yaml:            "address: legacy:9092\naddress_list: broker-one:9092",
			flags:           []string{"-ingest.address-list="},
			expectedBrokers: []string{"legacy:9092"},
		},
		{
			name:            "whitespace around CSV addresses",
			yaml:            "address_list: ' broker-one:9092 , broker-two:9093 '",
			expectedBrokers: []string{"broker-one:9092", "broker-two:9093"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg KafkaConfig
			f := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("ingest", f)
			cfg.Topic = "test"
			require.NoError(t, yaml.UnmarshalStrict([]byte(tt.yaml), &cfg))
			require.NoError(t, f.Parse(tt.flags))
			require.NoError(t, cfg.Validate())
			originalAddresses := slices.Clone(cfg.AddressList)

			opts, err := commonKafkaClientOptions(cfg, nil, log.NewNopLogger())
			require.NoError(t, err)
			client, err := kgo.NewClient(opts...)
			require.NoError(t, err)
			t.Cleanup(client.Close)
			require.Equal(t, []any{tt.expectedBrokers}, client.OptValues(kgo.SeedBrokers))
			require.Equal(t, originalAddresses, cfg.AddressList, "normalization must not mutate configuration")
		})
	}
}

func TestKafkaConfig_AddressListRejectsYAMLList(t *testing.T) {
	var cfg KafkaConfig
	require.Error(t, yaml.UnmarshalStrict([]byte("address_list: [broker-one:9092, broker-two:9093]"), &cfg))
}

func TestKafkaConfig_AddressListFlag_SplitsCSV(t *testing.T) {
	var cfg KafkaConfig
	f := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.RegisterFlagsWithPrefix("ingest", f)

	require.NoError(t, f.Parse([]string{"-ingest.address-list=broker-one:9092, broker-two:9093"}))
	require.Equal(t, []string{"broker-one:9092", " broker-two:9093"}, []string(cfg.AddressList))
}

func TestKafkaConfig_SeedBrokers_AddressIsUnchanged(t *testing.T) {
	address := " broker-one:9092, broker-two:9093 "
	cfg := KafkaConfig{Address: address}

	require.Equal(t, []string{address}, cfg.seedBrokers())
}

func TestKafkaConfig_Validate_AddressList(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		flags []string
	}{
		{name: "missing brokers", yaml: "address: ''"},
		{name: "empty AddressList without legacy address", yaml: "address: ''\naddress_list: ''"},
		{name: "empty YAML CSV entry", yaml: "address_list: 'broker-one:9092,'"},
		{name: "whitespace broker", yaml: "address_list: '  '"},
		{name: "empty CSV flag entry", flags: []string{"-ingest.address-list=broker-one:9092,"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg KafkaConfig
			f := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("ingest", f)
			cfg.Topic = "test"
			require.NoError(t, yaml.UnmarshalStrict([]byte(tt.yaml), &cfg))
			require.NoError(t, f.Parse(tt.flags))
			require.ErrorIs(t, cfg.Validate(), ErrMissingKafkaAddress)
		})
	}
}

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
