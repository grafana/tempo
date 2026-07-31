package ingest

import (
	"testing"

	dstls "github.com/grafana/dskit/crypto/tls"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kmsg"
)

func validKafkaConfig(address string) KafkaConfig {
	return KafkaConfig{
		Address:                    address,
		Topic:                      "test-topic",
		ProducerMaxRecordSizeBytes: maxProducerRecordDataBytesLimit,
		// Both 0 satisfies the consistency check (disabled).
		TargetConsumerLagAtStartup: 0,
		MaxConsumerLagAtStartup:    0,
	}
}

func TestKafkaConfigValidate_TLS(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.NumBrokers(1))
	require.NoError(t, err)
	t.Cleanup(cluster.Close)

	addrs := cluster.ListenAddrs()
	require.Len(t, addrs, 1)

	t.Run("TLS disabled skips validation", func(t *testing.T) {
		cfg := validKafkaConfig(addrs[0])
		cfg.TLSEnabled = false
		cfg.TLS = dstls.ClientConfig{CAPath: "/nonexistent/ca.crt"}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("TLS enabled with empty config is valid", func(t *testing.T) {
		cfg := validKafkaConfig(addrs[0])
		cfg.TLSEnabled = true
		assert.NoError(t, cfg.Validate())
	})

	t.Run("TLS enabled with invalid CA path returns ErrInvalidTLSConfig", func(t *testing.T) {
		cfg := validKafkaConfig(addrs[0])
		cfg.TLSEnabled = true
		cfg.TLS = dstls.ClientConfig{CAPath: "/nonexistent/ca.crt"}
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTLSConfig)
	})

	t.Run("TLS enabled with cert but no key returns ErrInvalidTLSConfig", func(t *testing.T) {
		cfg := validKafkaConfig(addrs[0])
		cfg.TLSEnabled = true
		cfg.TLS = dstls.ClientConfig{CertPath: "/tls/tls.crt"}
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTLSConfig)
	})
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
