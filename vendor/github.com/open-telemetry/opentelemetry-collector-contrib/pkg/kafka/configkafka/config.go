// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configkafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka"

import (
	"errors"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
)

const (
	LatestOffset   = "latest"
	EarliestOffset = "earliest"
)

// GroupRebalanceStrategy defines the strategy to use for partition assignment.
type GroupRebalanceStrategy string

const (
	// RangeBalanceStrategy assigns partitions on a per-topic basis.
	RangeBalanceStrategy GroupRebalanceStrategy = "range"
	// RoundRobinBalanceStrategy assigns partitions to all consumers in a round-robin fashion.
	RoundRobinBalanceStrategy GroupRebalanceStrategy = "roundrobin"
	// StickyBalanceStrategy attempts to preserve previous assignments when rebalancing.
	StickyBalanceStrategy GroupRebalanceStrategy = "sticky"
	// CooperativeStickyBalanceStrategy is similar to sticky but uses cooperative rebalancing.
	CooperativeStickyBalanceStrategy GroupRebalanceStrategy = "cooperative-sticky"
)

type ClientConfig struct {
	// Brokers holds the list of Kafka bootstrap servers (default localhost:9092).
	Brokers []string `mapstructure:"brokers"`

	// ResolveCanonicalBootstrapServersOnly configures the Kafka client to perform
	// a DNS lookup on each of the provided brokers, and then perform a reverse
	// lookup on the resulting IPs to obtain the canonical hostnames to use as the
	// bootstrap servers. This can be required in SASL environments.
	ResolveCanonicalBootstrapServersOnly bool `mapstructure:"resolve_canonical_bootstrap_servers_only"`

	// ProtocolVersion defines the Kafka protocol version that the client will
	// assume it is running against.
	ProtocolVersion string `mapstructure:"protocol_version"`

	// ClientID holds the client ID advertised to Kafka, which can be used for
	// enforcing ACLs, throttling quotas, and more (default "otel-collector")
	ClientID string `mapstructure:"client_id"`

	// Authentication holds Kafka authentication details.
	Authentication AuthenticationConfig `mapstructure:"auth"`

	// TLS holds TLS-related configuration for connecting to Kafka brokers.
	//
	// By default the client will use an insecure connection unless
	// SASL/AWS_MSK_IAM_OAUTHBEARER auth is configured.
	TLS *configtls.ClientConfig `mapstructure:"tls"`

	// Metadata holds metadata-related configuration for producers and consumers.
	Metadata MetadataConfig `mapstructure:"metadata"`

	// RackID provides the rack identifier for this client to enable rack-aware
	// replica selection when supported by the brokers. This maps to Kafka's
	// standard "client.rack" setting. By default, this is empty.
	RackID string `mapstructure:"rack_id"`

	// When enabled, the consumer uses the leader epoch returned by brokers (KIP-320)
	// to detect log truncation. Setting this to false clears the leader epoch from
	// fetch offsets, disabling KIP-320. Disabling can improve compatibility with
	// brokers that don’t fully support leader epochs (e.g., Azure Event Hubs),
	// at the cost of losing automatic log-truncation safety.
	//
	// NOTE: this is experimental and may be removed in a future release.
	UseLeaderEpoch bool `mapstructure:"use_leader_epoch"`

	// ConnIdleTimeout specifies the time after which idle connections are not reused and may be closed.
	ConnIdleTimeout time.Duration `mapstructure:"conn_idle_timeout"`
}

func NewDefaultClientConfig() ClientConfig {
	return ClientConfig{
		Brokers:         []string{"localhost:9092"},
		ClientID:        "otel-collector",
		Metadata:        NewDefaultMetadataConfig(),
		UseLeaderEpoch:  true,
		ConnIdleTimeout: 9 * time.Minute,
	}
}

func (c ClientConfig) Validate() error {
	if len(c.Brokers) == 0 {
		return errors.New("brokers must be specified")
	}
	if c.ProtocolVersion != "" {
		if _, err := sarama.ParseKafkaVersion(c.ProtocolVersion); err != nil {
			return fmt.Errorf("invalid protocol version: %w", err)
		}
	}
	if c.ConnIdleTimeout <= 0 {
		return fmt.Errorf("conn_idle_timeout (%s) must be positive", c.ConnIdleTimeout)
	}
	return nil
}

type ConsumerConfig struct {
	// SessionTimeout controls the Kafka consumer group session timeout.
	// The session timeout is used to detect the consumer's liveness.
	SessionTimeout time.Duration `mapstructure:"session_timeout"`

	// HeartbeatInterval controls the Kafka consumer group coordination
	// heartbeat interval. Heartbeats ensure the consumer's session remains
	// active.
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`

	// GroupID specifies the ID of the consumer group that will be
	// consuming messages from (default "otel-collector").
	GroupID string `mapstructure:"group_id"`

	// InitialOffset specifies the initial offset to use if no offset was
	// previously committed. Must be `latest` or `earliest` (default "latest").
	InitialOffset string `mapstructure:"initial_offset"`

	// AutoCommit controls the auto-commit functionality of the consumer.
	AutoCommit AutoCommitConfig `mapstructure:"autocommit"`

	// The minimum bytes per fetch from Kafka (default "1")
	MinFetchSize int32 `mapstructure:"min_fetch_size"`

	// The maximum bytes per fetch from Kafka (default "1048576")
	MaxFetchSize int32 `mapstructure:"max_fetch_size"`

	// The maximum amount of time to wait for MinFetchSize bytes to be
	// available before the broker returns a response (default 250ms)
	MaxFetchWait time.Duration `mapstructure:"max_fetch_wait"`

	// MaxPartitionFetchSize defines the maximum number of bytes to fetch
	// per partition (default "1048576")
	MaxPartitionFetchSize int32 `mapstructure:"max_partition_fetch_size"`

	// GroupRebalanceStrategy specifies the strategy to use for partition assignment.
	// Possible values are "range", "roundrobin", "sticky", and "cooperative-sticky".
	//
	// Defaults to "cooperative-sticky"
	GroupRebalanceStrategy GroupRebalanceStrategy `mapstructure:"group_rebalance_strategy,omitempty"`

	// GroupInstanceID specifies the ID of the consumer
	GroupInstanceID string `mapstructure:"group_instance_id,omitempty"`
}

func NewDefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		SessionTimeout:    10 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		GroupID:           "otel-collector",
		InitialOffset:     LatestOffset,
		AutoCommit: AutoCommitConfig{
			Enable:   true,
			Interval: time.Second,
		},
		MinFetchSize:          1,
		MaxFetchSize:          1048576,
		MaxFetchWait:          250 * time.Millisecond,
		MaxPartitionFetchSize: 1048576,
	}
}

func (c ConsumerConfig) Validate() error {
	switch c.InitialOffset {
	case LatestOffset, EarliestOffset:
		// Valid
	default:
		return fmt.Errorf(
			"initial_offset should be one of 'latest' or 'earliest'. configured value %v",
			c.InitialOffset,
		)
	}

	if c.GroupRebalanceStrategy != "" {
		switch c.GroupRebalanceStrategy {
		case RangeBalanceStrategy, RoundRobinBalanceStrategy, StickyBalanceStrategy, CooperativeStickyBalanceStrategy:
			// Valid
		default:
			return fmt.Errorf(
				"rebalance_strategy should be one of '%s', '%s', '%s', or '%s'. configured value %v",
				RangeBalanceStrategy, RoundRobinBalanceStrategy, StickyBalanceStrategy, CooperativeStickyBalanceStrategy,
				c.GroupRebalanceStrategy,
			)
		}
	}

	// Validate fetch size constraints
	if c.MinFetchSize < 0 {
		return fmt.Errorf("min_fetch_size (%d) must be non-negative", c.MinFetchSize)
	}
	if c.MaxFetchSize < 0 {
		return fmt.Errorf("max_fetch_size (%d) must be non-negative", c.MaxFetchSize)
	}
	if c.MaxPartitionFetchSize < 0 {
		return fmt.Errorf("max_partition_fetch_size (%d) must be non-negative", c.MaxPartitionFetchSize)
	}
	if c.MaxFetchSize < c.MinFetchSize {
		return fmt.Errorf(
			"max_fetch_size (%d) cannot be less than min_fetch_size (%d)",
			c.MaxFetchSize,
			c.MinFetchSize,
		)
	}

	return nil
}

type AutoCommitConfig struct {
	// Whether or not to auto-commit updated offsets back to the broker.
	// (default enabled).
	Enable bool `mapstructure:"enable"`

	// How frequently to commit updated offsets. Ineffective unless
	// auto-commit is enabled (default 1s)
	Interval time.Duration `mapstructure:"interval"`
}

type ProducerConfig struct {
	// Maximum message bytes the producer will accept to produce (default 1000000)
	MaxMessageBytes int `mapstructure:"max_message_bytes"`

	// RequiredAcks holds the number acknowledgements required before producing
	// returns successfully. See:
	// https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#acks
	//
	// Acceptable values are:
	//   0 (NoResponse)   Does not wait for any acknowledgements.
	//   1 (WaitForLocal) Waits for only the leader to write the record to its local log,
	//                    but does not wait for followers to acknowledge. (default)
	//  -1 (WaitForAll)   Waits for all in-sync replicas to acknowledge.
	//                    In YAML configuration, "all" is accepted as an alias for -1.
	RequiredAcks RequiredAcks `mapstructure:"required_acks"`

	// Compression Codec used to produce messages
	// https://pkg.go.dev/github.com/twmb/franz-go/pkg/kgo#CompressionCodec
	// The options are: 'none' (default), 'gzip', 'snappy', 'lz4', and 'zstd'
	Compression string `mapstructure:"compression"`

	// CompressionParams defines compression parameters for the producer.
	CompressionParams configcompression.CompressionParams `mapstructure:"compression_params"`

	// The maximum number of messages the producer will send in a single
	// broker request. Defaults to 10000 (franz-go default). Similar to
	// `queue.buffering.max.messages` in the JVM producer.
	FlushMaxMessages int `mapstructure:"flush_max_messages"`

	// Whether or not to allow automatic topic creation.
	// (default enabled).
	AllowAutoTopicCreation bool `mapstructure:"allow_auto_topic_creation"`

	// Linger controls the linger time for the producer.
	// (default 10ms).
	Linger time.Duration `mapstructure:"linger"`
}

func NewDefaultProducerConfig() ProducerConfig {
	return ProducerConfig{
		MaxMessageBytes:        1000000,
		RequiredAcks:           WaitForLocal,
		Compression:            "none",
		FlushMaxMessages:       10000,
		AllowAutoTopicCreation: true,
		Linger:                 10 * time.Millisecond,
	}
}

func (c ProducerConfig) Validate() error {
	switch c.Compression {
	case "none", "gzip", "snappy", "lz4", "zstd":
		ct := configcompression.Type(c.Compression)
		if ct.IsCompressed() {
			if err := ct.ValidateParams(c.CompressionParams); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf(
			"compression should be one of 'none', 'gzip', 'snappy', 'lz4', or 'zstd'. configured value is %q",
			c.Compression,
		)
	}
	if c.MaxMessageBytes < 0 {
		return fmt.Errorf("max_message_bytes (%d) must be non-negative", c.MaxMessageBytes)
	}
	if c.FlushMaxMessages < 1 {
		return fmt.Errorf("flush_max_messages (%d) must be at least 1", c.FlushMaxMessages)
	}
	return nil
}

// Unmarshal unmarshals into ProducerConfig, allowing the user to specify any of ["all", -1, 0, 1]
// for required_acks. This is in line with standard Kafka producer configuration as described at
// https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#acks
//
// Note that confmap.Unmarshaler may only be implemented by structs, so we cannot define this method
// on RequiredAcks itself.
func (c *ProducerConfig) Unmarshal(conf *confmap.Conf) error {
	if conf.Get("required_acks") == "all" {
		if err := conf.Merge(confmap.NewFromStringMap(
			map[string]any{"required_acks": WaitForAll},
		)); err != nil {
			return err
		}
	}
	return conf.Unmarshal(c)
}

// RequiredAcks defines record acknowledgement behavior for producers.
type RequiredAcks int

const (
	// NoResponse doesn't send any response, the TCP ACK is all you get.
	NoResponse RequiredAcks = 0
	// WaitForLocal waits for only the local commit to succeed before responding.
	WaitForLocal RequiredAcks = 1
	// WaitForAll waits for all in-sync replicas to commit before responding.
	// The minimum number of in-sync replicas is configured on the broker via
	// the `min.insync.replicas` configuration key.
	WaitForAll RequiredAcks = -1
)

func (r RequiredAcks) Validate() error {
	if r < -1 || r > 1 {
		return fmt.Errorf("expected 'all' (-1), 0, or 1; configured value is %v", r)
	}
	return nil
}

type MetadataConfig struct {
	// Whether to maintain a full set of metadata for all topics, or just
	// the minimal set that has been necessary so far. The full set is simpler
	// and usually more convenient, but can take up a substantial amount of
	// memory if you have many topics and partitions. Defaults to true.
	Full bool `mapstructure:"full"`

	// RefreshInterval controls the frequency at which cluster metadata is
	// refreshed. Defaults to 10 minutes.
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`

	// Retry configuration for metadata.
	// This configuration is useful to avoid race conditions when broker
	// is starting at the same time as collector.
	Retry MetadataRetryConfig `mapstructure:"retry"`
}

// MetadataRetryConfig defines retry configuration for Metadata.
type MetadataRetryConfig struct {
	// The total number of times to retry a metadata request when the
	// cluster is in the middle of a leader election or at startup (default 3).
	Max int `mapstructure:"max"`
	// How long to wait for leader election to occur before retrying
	// (default 250ms). Similar to the JVM's `retry.backoff.ms`.
	Backoff time.Duration `mapstructure:"backoff"`
}

func NewDefaultMetadataConfig() MetadataConfig {
	return MetadataConfig{
		Full:            true,
		RefreshInterval: 10 * time.Minute,
		Retry: MetadataRetryConfig{
			Max:     3,
			Backoff: time.Millisecond * 250,
		},
	}
}

// AuthenticationConfig defines authentication-related configuration.
type AuthenticationConfig struct {
	// PlainText is an alias for SASL/PLAIN authentication.
	//
	// Deprecated [v0.123.0]: use SASL with Mechanism set to PLAIN instead.
	PlainText *PlainTextConfig `mapstructure:"plain_text"`

	// SASL holds SASL authentication configuration.
	SASL *SASLConfig `mapstructure:"sasl"`

	// Kerberos holds Kerberos authentication configuration.
	Kerberos *KerberosConfig `mapstructure:"kerberos"`

	// TLS holds TLS configuration for connecting to Kafka brokers.
	//
	// Deprecated [v0.124.0]: use ClientConfig.TLS instead. This will
	// be used only if ClientConfig.TLS is not set.
	TLS *configtls.ClientConfig `mapstructure:"tls"`
}

// PlainTextConfig defines plaintext authentication.
type PlainTextConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// SASLConfig defines the configuration for the SASL authentication.
type SASLConfig struct {
	// Username to be used on authentication
	Username string `mapstructure:"username"`
	// Password to be used on authentication
	Password string `mapstructure:"password"`
	// SASL Mechanism to be used, possible values are: (PLAIN, AWS_MSK_IAM_OAUTHBEARER, SCRAM-SHA-256 or SCRAM-SHA-512).
	Mechanism string `mapstructure:"mechanism"`
	// SASL Protocol Version to be used, possible values are: (0, 1). Defaults to 0.
	Version int `mapstructure:"version"`
	// AWSMSK holds configuration specific to AWS MSK.
	AWSMSK AWSMSKConfig `mapstructure:"aws_msk"`
}

func (c SASLConfig) Validate() error {
	switch c.Mechanism {
	case "AWS_MSK_IAM_OAUTHBEARER":
		// TODO validate c.AWSMSK
	case "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512":
		// Do nothing, valid mechanism
		if c.Username == "" {
			return errors.New("username is required")
		}
		if c.Password == "" {
			return errors.New("password is required")
		}
	default:
		return fmt.Errorf(
			"mechanism should be one of 'PLAIN', 'AWS_MSK_IAM_OAUTHBEARER', 'SCRAM-SHA-256' or 'SCRAM-SHA-512'. configured value %v",
			c.Mechanism,
		)
	}
	if c.Version < 0 || c.Version > 1 {
		return fmt.Errorf("version has to be either 0 or 1. configured value %v", c.Version)
	}
	return nil
}

// AWSMSKConfig defines the additional SASL authentication
// measures needed to use the AWS_MSK_IAM_OAUTHBEARER mechanism
type AWSMSKConfig struct {
	// Region is the AWS region the MSK cluster is based in
	Region string `mapstructure:"region"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// KerberosConfig defines kerberos configuration.
type KerberosConfig struct {
	ServiceName     string `mapstructure:"service_name"`
	Realm           string `mapstructure:"realm"`
	UseKeyTab       bool   `mapstructure:"use_keytab"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password" json:"-"`
	ConfigPath      string `mapstructure:"config_file"`
	KeyTabPath      string `mapstructure:"keytab_file"`
	DisablePAFXFAST bool   `mapstructure:"disable_fast_negotiation"`
}
