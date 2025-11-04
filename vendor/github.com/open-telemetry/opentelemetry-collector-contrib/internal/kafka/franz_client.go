// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	krb5client "github.com/jcmturner/gokrb5/v8/client"
	krb5config "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/kversion"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/kerberos"
	"github.com/twmb/franz-go/pkg/sasl/oauth"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"github.com/twmb/franz-go/plugin/kzap"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka"
)

const (
	SCRAMSHA512          = "SCRAM-SHA-512"
	SCRAMSHA256          = "SCRAM-SHA-256"
	PLAIN                = "PLAIN"
	AWSMSKIAMOAUTHBEARER = "AWS_MSK_IAM_OAUTHBEARER" //nolint:gosec // These aren't credentials.
)

// NewFranzSyncProducer creates a new Kafka client using the franz-go library.
func NewFranzSyncProducer(ctx context.Context, clientCfg configkafka.ClientConfig,
	cfg configkafka.ProducerConfig,
	timeout time.Duration,
	logger *zap.Logger,
	opts ...kgo.Opt,
) (*kgo.Client, error) {
	codec := compressionCodec(cfg.Compression)
	switch cfg.CompressionParams.Level {
	case 0, configcompression.DefaultCompressionLevel:
	default:
		codec = codec.WithLevel(int(cfg.CompressionParams.Level))
	}
	opts, err := commonOpts(ctx, clientCfg, logger, append(
		opts,
		kgo.ProduceRequestTimeout(timeout),
		kgo.ProducerBatchCompression(codec),
		// Use the UniformBytesPartitioner that is the default in franz-go with
		// the legacy compatibility sarama hashing to avoid hashing to different
		// partitions in case partitioning is enabled.
		kgo.RecordPartitioner(newSaramaCompatPartitioner()),
	)...)
	if err != nil {
		return nil, err
	}
	// Configure required acks
	switch cfg.RequiredAcks {
	case configkafka.WaitForAll:
		opts = append(opts, kgo.RequiredAcks(kgo.AllISRAcks()))
	case configkafka.NoResponse:
		// NOTE(marclop) only disable if acks != all.
		opts = append(opts, kgo.DisableIdempotentWrite(), kgo.RequiredAcks(kgo.NoAck()))
	default: // WaitForLocal
		// NOTE(marclop) only disable if acks != all.
		opts = append(opts, kgo.DisableIdempotentWrite(), kgo.RequiredAcks(kgo.LeaderAck()))
	}

	// Configure max message size
	if cfg.MaxMessageBytes > 0 {
		opts = append(opts, kgo.ProducerBatchMaxBytes(
			int32(cfg.MaxMessageBytes),
		))
	}
	// Configure batch size
	if cfg.FlushMaxMessages > 0 {
		opts = append(opts, kgo.MaxBufferedRecords(cfg.FlushMaxMessages))
	}
	// Configure auto topic creation
	if cfg.AllowAutoTopicCreation {
		opts = append(opts, kgo.AllowAutoTopicCreation())
	}

	return kgo.NewClient(opts...)
}

// NewFranzConsumerGroup creates a new Kafka consumer client using the franz-go library.
func NewFranzConsumerGroup(ctx context.Context, clientCfg configkafka.ClientConfig,
	consumerCfg configkafka.ConsumerConfig,
	topics []string,
	logger *zap.Logger,
	opts ...kgo.Opt,
) (*kgo.Client, error) {
	opts, err := commonOpts(ctx, clientCfg, logger, append([]kgo.Opt{
		kgo.ConsumeTopics(topics...),
		kgo.ConsumerGroup(consumerCfg.GroupID),
	}, opts...)...)
	if err != nil {
		return nil, err
	}

	for _, t := range topics {
		// Similar to librdkafka, if the topic starts with `^`, it is a regex topic:
		// https://github.com/confluentinc/librdkafka/blob/b871fdabab84b2ea1be3866a2ded4def7e31b006/src/rdkafka.h#L3899-L3938
		if strings.HasPrefix(t, "^") {
			opts = append(opts, kgo.ConsumeRegex())
			break
		}
	}

	// Configure session timeout
	if consumerCfg.SessionTimeout > 0 {
		opts = append(opts, kgo.SessionTimeout(consumerCfg.SessionTimeout))
	}

	// Configure heartbeat interval
	if consumerCfg.HeartbeatInterval > 0 {
		opts = append(opts, kgo.HeartbeatInterval(consumerCfg.HeartbeatInterval))
	}

	// Configure fetch sizes
	if consumerCfg.MinFetchSize > 0 {
		opts = append(opts, kgo.FetchMinBytes(consumerCfg.MinFetchSize))
	}
	if consumerCfg.DefaultFetchSize > 0 {
		opts = append(opts, kgo.FetchMaxBytes(consumerCfg.DefaultFetchSize))
	}
	if consumerCfg.MaxPartitionFetchSize > 0 {
		opts = append(opts, kgo.FetchMaxPartitionBytes(consumerCfg.MaxPartitionFetchSize))
	}

	// Configure max fetch wait
	if consumerCfg.MaxFetchWait > 0 {
		opts = append(opts, kgo.FetchMaxWait(consumerCfg.MaxFetchWait))
	}

	interval := consumerCfg.AutoCommit.Interval
	if !consumerCfg.AutoCommit.Enable {
		// Set auto-commit interval to a very high value to "disable" it, but
		// still allow using marks.
		interval = time.Hour
	}
	// Configure auto-commit to use marks, this simplifies the committing
	// logic and makes it more consistent with the Sarama client.
	opts = append(opts, kgo.AutoCommitMarks(),
		kgo.AutoCommitInterval(interval),
	)

	// Configure the offset to reset to if an exception is found (or no current
	// partition offset is found.
	switch consumerCfg.InitialOffset {
	case configkafka.EarliestOffset:
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	case configkafka.LatestOffset:
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}

	// Configure group instance ID if provided
	if consumerCfg.GroupInstanceID != "" {
		opts = append(opts, kgo.InstanceID(consumerCfg.GroupInstanceID))
	}

	// Configure rebalance strategy
	switch consumerCfg.GroupRebalanceStrategy {
	case "range": // Sarama default.
		opts = append(opts, kgo.Balancers(kgo.RangeBalancer()))
	case "roundrobin":
		opts = append(opts, kgo.Balancers(kgo.RoundRobinBalancer()))
	case "sticky":
		opts = append(opts, kgo.Balancers(kgo.StickyBalancer()))
	// NOTE(marclop): This is a new type of balancer, document accordingly.
	case "cooperative-sticky":
		opts = append(opts, kgo.Balancers(kgo.CooperativeStickyBalancer()))
	}
	return kgo.NewClient(opts...)
}

// NewFranzClient creates a franz-go client using the same commonOpts used for producer/consumer.
func NewFranzClient(
	ctx context.Context,
	clientCfg configkafka.ClientConfig,
	logger *zap.Logger,
	opts ...kgo.Opt,
) (*kgo.Client, error) {
	opts, err := commonOpts(ctx, clientCfg, logger, opts...)
	if err != nil {
		return nil, err
	}
	return kgo.NewClient(opts...)
}

// NewFranzClusterAdminClient creates a kadm admin client from a freshly created franz client.
func NewFranzClusterAdminClient(
	ctx context.Context,
	clientCfg configkafka.ClientConfig,
	logger *zap.Logger,
	opts ...kgo.Opt,
) (*kadm.Client, *kgo.Client, error) {
	cl, err := NewFranzClient(ctx, clientCfg, logger, opts...)
	if err != nil {
		return nil, nil, err
	}
	return kadm.NewClient(cl), cl, nil
}

// NewFranzAdminFromClient returns a kadm admin bound to an existing kgo client.
func NewFranzAdminFromClient(cl *kgo.Client) *kadm.Client {
	return kadm.NewClient(cl)
}

func commonOpts(ctx context.Context, clientCfg configkafka.ClientConfig,
	logger *zap.Logger,
	opts ...kgo.Opt,
) ([]kgo.Opt, error) {
	opts = append(opts,
		kgo.WithLogger(kzap.New(logger.Named("franz"))),
		kgo.SeedBrokers(clientCfg.Brokers...),
		// Disable client metrics, since some brokers may falsely indicate
		// that they support them when they don't, causing errors to be
		// logged. We may want to make this configurable in the future.
		kgo.DisableClientMetrics(),
	)
	tlsConfig := clientCfg.TLS
	if tlsConfig == nil {
		tlsConfig = clientCfg.Authentication.TLS
	}
	// Configure TLS if needed
	if tlsConfig != nil {
		tlsCfg, err := tlsConfig.LoadTLSConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		if tlsCfg != nil {
			opts = append(opts, kgo.DialTLSConfig(tlsCfg))
		}
	} else {
		// Add a hook that gives a hint that TLS is not configured
		// in case we try to connect to a TLS-only broker.
		var hook kgo.HookBrokerConnect = hookBrokerConnectPlaintext{logger: logger}
		opts = append(opts, kgo.WithHooks(hook))
	}
	// Configure authentication
	if clientCfg.Authentication.PlainText != nil {
		auth := plain.Auth{
			User: clientCfg.Authentication.PlainText.Username,
			Pass: clientCfg.Authentication.PlainText.Password,
		}
		opts = append(opts, kgo.SASL(auth.AsMechanism()))
	}
	if clientCfg.Authentication.SASL != nil {
		saslOpt, err := configureKgoSASL(clientCfg.Authentication.SASL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure SASL: %w", err)
		}
		opts = append(opts, saslOpt)
	}
	if clientCfg.Authentication.Kerberos != nil {
		opt, err := configureKgoKerberos(clientCfg.Authentication.Kerberos)
		if err != nil {
			return nil, fmt.Errorf("failed to configure Kerberos: %w", err)
		}
		opts = append(opts, opt)
	}
	// Configure client ID
	if clientCfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(clientCfg.ClientID))
	}
	// Configure client rack if provided
	if clientCfg.RackID != "" {
		opts = append(opts, kgo.Rack(clientCfg.RackID))
	}
	// Reuse existing metadata refresh interval for franz-go metadataMaxAge
	if clientCfg.Metadata.RefreshInterval > 0 {
		opts = append(opts, kgo.MetadataMaxAge(clientCfg.Metadata.RefreshInterval))
	}
	// Configure the min/max protocol version if provided
	if clientCfg.ProtocolVersion != "" {
		keyVersions := make(map[string]any)
		versions := kversion.FromString(clientCfg.ProtocolVersion)
		versions.EachMaxKeyVersion(func(k, v int16) {
			name := kmsg.NameForKey(k)
			keyVersions[name] = v
		})
		logger.Info(
			"setting kafka protocol version",
			zap.String("version", clientCfg.ProtocolVersion),
			zap.Any("key_versions", keyVersions),
		)
		opts = append(opts, kgo.MinVersions(versions), kgo.MaxVersions(versions))
	}
	return opts, nil
}

func configureKgoSASL(cfg *configkafka.SASLConfig) (kgo.Opt, error) {
	var m sasl.Mechanism
	switch cfg.Mechanism {
	case PLAIN:
		m = plain.Auth{User: cfg.Username, Pass: cfg.Password}.AsMechanism()
	case SCRAMSHA256:
		m = scram.Auth{User: cfg.Username, Pass: cfg.Password}.AsSha256Mechanism()
	case SCRAMSHA512:
		m = scram.Auth{User: cfg.Username, Pass: cfg.Password}.AsSha512Mechanism()
	case AWSMSKIAMOAUTHBEARER:
		m = oauth.Oauth(func(ctx context.Context) (oauth.Auth, error) {
			token, _, err := signer.GenerateAuthToken(ctx, cfg.AWSMSK.Region)
			return oauth.Auth{Token: token}, err
		})
	default:
		return nil, fmt.Errorf("unsupported SASL mechanism: %s", cfg.Mechanism)
	}
	return kgo.SASL(m), nil
}

func configureKgoKerberos(cfg *configkafka.KerberosConfig) (kgo.Opt, error) {
	kAuth := kerberos.Auth{Service: cfg.ServiceName}
	commonCfg := krb5config.New()
	if cfg.ConfigPath != "" {
		c, err := krb5config.Load(cfg.ConfigPath)
		if err != nil {
			return nil, err
		}
		commonCfg = c
	}

	disableFAST := krb5client.DisablePAFXFAST(cfg.DisablePAFXFAST)
	if cfg.KeyTabPath != "" {
		kt, err := keytab.Load(cfg.KeyTabPath)
		if err != nil {
			return nil, err
		}
		kAuth.Client = krb5client.NewWithKeytab(
			cfg.Username, cfg.Realm, kt, commonCfg, disableFAST,
		)
	} else {
		kAuth.Client = krb5client.NewWithPassword(
			cfg.Username, cfg.Realm, cfg.Password, commonCfg, disableFAST,
		)
	}
	return kgo.SASL(kAuth.AsMechanism()), nil
}

func compressionCodec(compression string) kgo.CompressionCodec {
	switch compression {
	case "gzip":
		return kgo.GzipCompression()
	case "snappy":
		return kgo.SnappyCompression()
	case "lz4":
		return kgo.Lz4Compression()
	case "zstd":
		return kgo.ZstdCompression()
	case "none":
		return kgo.NoCompression()
	default:
		return kgo.NoCompression()
	}
}

func newSaramaCompatPartitioner() kgo.Partitioner {
	return kgo.StickyKeyPartitioner(kgo.SaramaCompatHasher(saramaHashFn))
}

func saramaHashFn(b []byte) uint32 {
	h := fnv.New32a()
	h.Reset()
	h.Write(b)
	return h.Sum32()
}

type hookBrokerConnectPlaintext struct {
	logger *zap.Logger
}

func (h hookBrokerConnectPlaintext) OnBrokerConnect(
	meta kgo.BrokerMetadata, _ time.Duration, conn net.Conn, err error,
) {
	if conn == nil {
		// Could not make a network connection, this is not related to TLS.
		return
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		h.logger.Warn(
			"failed to connect to broker, it may require TLS but TLS is not configured",
			zap.String("host", meta.Host),
			zap.Error(err),
		)
		return
	}
}
