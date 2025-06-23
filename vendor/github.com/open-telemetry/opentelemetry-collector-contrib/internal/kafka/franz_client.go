// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	krb5client "github.com/jcmturner/gokrb5/v8/client"
	krb5config "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/aws"
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
	AWSMSKIAM            = "AWS_MSK_IAM"
	AWSMSKIAMOAUTHBEARER = "AWS_MSK_IAM_OAUTHBEARER" //nolint:gosec // These aren't credentials.
)

// NewFranzSyncProducer creates a new Kafka client using the franz-go library.
func NewFranzSyncProducer(clientCfg configkafka.ClientConfig,
	cfg configkafka.ProducerConfig,
	timeout time.Duration,
	logger *zap.Logger,
) (*kgo.Client, error) {
	codec := compressionCodec(cfg.Compression)
	switch cfg.CompressionParams.Level {
	case 0, configcompression.DefaultCompressionLevel:
	default:
		codec = codec.WithLevel(int(cfg.CompressionParams.Level))
	}
	opts := []kgo.Opt{
		kgo.SeedBrokers(clientCfg.Brokers...),
		kgo.WithLogger(kzap.New(logger.Named("kafka"))),
		kgo.ProduceRequestTimeout(timeout),
		kgo.ProducerBatchCompression(codec),
		// Use the UniformBytesPartitioner that is the default in franz-go with
		// the legacy compatibility sarama hashing to avoid hashing to different
		// partitions in case partitioning is enabled.
		kgo.RecordPartitioner(newSaramaCompatPartitioner()),
	}
	// Configure required acks
	switch cfg.RequiredAcks {
	case configkafka.WaitForAll:
		opts = append(opts, kgo.RequiredAcks(kgo.AllISRAcks()))
	case configkafka.NoResponse:
		// NOTE(marclop) only disable if acks != all.
		opts = append(opts, kgo.DisableIdempotentWrite())
		opts = append(opts, kgo.RequiredAcks(kgo.NoAck()))
	default: // WaitForLocal
		// NOTE(marclop) only disable if acks != all.
		opts = append(opts, kgo.DisableIdempotentWrite())
		opts = append(opts, kgo.RequiredAcks(kgo.LeaderAck()))
	}
	// Configure TLS if needed
	if clientCfg.TLS != nil {
		tlsCfg, err := clientCfg.TLS.LoadTLSConfig(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		if tlsCfg != nil {
			opts = append(opts, kgo.DialTLSConfig(tlsCfg))
		}
	}
	// Configure Auth
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

	return kgo.NewClient(opts...)
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
	case AWSMSKIAM:
		m = aws.ManagedStreamingIAM(func(ctx context.Context) (auth aws.Auth, _ error) {
			awscfg, err := awsconfig.LoadDefaultConfig(ctx)
			if err != nil {
				return auth, fmt.Errorf("kafka: error loading AWS config: %w", err)
			}
			creds, err := awscfg.Credentials.Retrieve(ctx)
			if err != nil {
				return auth, err
			}
			return aws.Auth{
				AccessKey:    creds.AccessKeyID,
				SecretKey:    creds.SecretAccessKey,
				SessionToken: creds.SessionToken,
			}, nil
		})
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
