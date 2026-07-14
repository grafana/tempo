package ingest

import (
	"flag"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kmsg"
)

func TestKafkaConfigValidate(t *testing.T) {
	// validConfig returns a config with all mandatory fields populated with valid
	// defaults so each test case can focus on the SASL fields under test.
	validConfig := func() KafkaConfig {
		cfg := KafkaConfig{}
		cfg.RegisterFlags(flag.NewFlagSet("test", flag.PanicOnError))
		cfg.Address = "localhost:9092"
		cfg.Topic = "test-topic"
		return cfg
	}

	tests := []struct {
		name        string
		mutate      func(cfg *KafkaConfig)
		expectedErr error
	}{
		{
			name:   "valid defaults",
			mutate: func(*KafkaConfig) {},
		},
		{
			name: "plain with credentials",
			mutate: func(cfg *KafkaConfig) {
				cfg.SASLUsername = "user"
				cfg.SASLPassword = flagext.SecretWithValue("pass")
				cfg.SASLMechanism = SASLMechanismPlain
			},
		},
		{
			name: "scram-sha-256 with credentials",
			mutate: func(cfg *KafkaConfig) {
				cfg.SASLUsername = "user"
				cfg.SASLPassword = flagext.SecretWithValue("pass")
				cfg.SASLMechanism = SASLMechanismScramSHA256
			},
		},
		{
			name: "scram-sha-512 with credentials",
			mutate: func(cfg *KafkaConfig) {
				cfg.SASLUsername = "user"
				cfg.SASLPassword = flagext.SecretWithValue("pass")
				cfg.SASLMechanism = SASLMechanismScramSHA512
			},
		},
		{
			name: "empty mechanism is allowed",
			mutate: func(cfg *KafkaConfig) {
				cfg.SASLUsername = "user"
				cfg.SASLPassword = flagext.SecretWithValue("pass")
				cfg.SASLMechanism = ""
			},
		},
		{
			name: "unsupported mechanism",
			mutate: func(cfg *KafkaConfig) {
				cfg.SASLUsername = "user"
				cfg.SASLPassword = flagext.SecretWithValue("pass")
				cfg.SASLMechanism = "SCRAM-SHA-1"
			},
			expectedErr: ErrUnsupportedSASLMechanism,
		},
		{
			name: "username without password",
			mutate: func(cfg *KafkaConfig) {
				cfg.SASLUsername = "user"
			},
			expectedErr: ErrInconsistentSASLCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
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
