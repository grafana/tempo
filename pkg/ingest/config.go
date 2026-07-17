package ingest

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/backoff"
	dskittls "github.com/grafana/dskit/crypto/tls"
	"github.com/grafana/dskit/flagext"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// SASLMechanism identifies the SASL authentication mechanism used to
// authenticate with the Kafka backend.
type SASLMechanism string

// Set implements flag.Value.
func (s *SASLMechanism) Set(v string) error {
	if !slices.Contains(saslMechanismOptions, v) {
		return ErrInvalidSASLMechanism
	}
	*s = SASLMechanism(v)
	return nil
}

// String implements flag.Value.
func (s *SASLMechanism) String() string {
	return string(*s)
}

// FlagType implements flagext.Value so the generated docs describe the flag as a string.
func (SASLMechanism) FlagType() string {
	return "string"
}

const (
	SASLMechanismPlain       SASLMechanism = "PLAIN"
	SASLMechanismScramSHA256 SASLMechanism = "SCRAM-SHA-256"
	SASLMechanismScramSHA512 SASLMechanism = "SCRAM-SHA-512"
	SASLMechanismOauthbearer SASLMechanism = "OAUTHBEARER"
	SASLMechanismMSKIAM      SASLMechanism = "AWS_MSK_IAM"
)

var saslMechanismOptions = []string{
	string(SASLMechanismPlain),
	string(SASLMechanismScramSHA256),
	string(SASLMechanismScramSHA512),
	string(SASLMechanismOauthbearer),
	string(SASLMechanismMSKIAM),
}

const (
	// writerRequestTimeoutOverhead is the overhead applied by the Writer to every Kafka timeout.
	// You can think about this overhead as an extra time for requests sitting in the client's buffer
	// before being sent on the wire and the actual time it takes to send it over the network and
	// start being processed by Kafka.
	writerRequestTimeoutOverhead = 2 * time.Second

	// producerBatchMaxBytes is the max allowed size of a batch of Kafka records.
	producerBatchMaxBytes = 16_000_000

	// maxProducerRecordDataBytesLimit is the max allowed size of a single record data. Given we have a limit
	// on the max batch size (producerBatchMaxBytes), a Kafka record data can't be bigger than the batch size
	// minus some overhead required to serialise the batch and the record itself. We use 16KB as such overhead
	// in the worst case scenario, which is expected to be way above the actual one.
	maxProducerRecordDataBytesLimit = producerBatchMaxBytes - 16384
	minProducerRecordDataBytesLimit = 1024 * 1024
)

var (
	ErrMissingKafkaAddress               = errors.New("the Kafka address has not been configured")
	ErrMissingKafkaTopic                 = errors.New("the Kafka topic has not been configured")
	ErrInconsistentConsumerLagAtStartup  = errors.New("the target and max consumer lag at startup must be either both set to 0 or to a value greater than 0")
	ErrInvalidMaxConsumerLagAtStartup    = errors.New("the configured max consumer lag at startup must greater or equal than the configured target consumer lag")
	ErrInvalidProducerMaxRecordSizeBytes = fmt.Errorf("the configured producer max record size bytes must be a value between %d and %d", minProducerRecordDataBytesLimit, maxProducerRecordDataBytesLimit)
	ErrInconsistentSASLCredentials       = errors.New("the SASL username and password must be both configured to enable SASL authentication")
	ErrInvalidSASLMechanism              = fmt.Errorf("the configured SASL mechanism is invalid, must be one of: %s", strings.Join(saslMechanismOptions, ", "))
	ErrSASLMSKIAMBadConfig               = errors.New("exactly one of static credentials, file path, or HTTP socket path must be configured to enable SASL AWS_MSK_IAM authentication")
	ErrSASLOauthbearerBadConfig          = errors.New("exactly one of OAuth token, file path, or HTTP socket path must be configured to enable SASL OAUTHBEARER authentication")
)

type Config struct {
	Kafka KafkaConfig `yaml:"kafka"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Kafka.RegisterFlagsWithPrefix(prefix, f)
}

func (cfg *Config) Validate() error {
	return cfg.Kafka.Validate()
}

// KafkaConfig holds the generic config for the Kafka backend.
type KafkaConfig struct {
	Address      string        `yaml:"address"`
	Topic        string        `yaml:"topic"`
	ClientID     string        `yaml:"client_id"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`

	SASL       KafkaAuthConfig `yaml:",inline"`
	TLSEnabled bool            `yaml:"tls_enabled"`
	TLS        TLSClientConfig `yaml:",inline"`

	ConsumerGroup                     string        `yaml:"consumer_group"`
	ConsumerGroupOffsetCommitInterval time.Duration `yaml:"consumer_group_offset_commit_interval"`

	LastProducedOffsetRetryTimeout time.Duration `yaml:"last_produced_offset_retry_timeout"`

	AutoCreateTopicEnabled           bool `yaml:"auto_create_topic_enabled"`
	AutoCreateTopicDefaultPartitions int  `yaml:"auto_create_topic_default_partitions"`

	ProducerMaxRecordSizeBytes int   `yaml:"producer_max_record_size_bytes"`
	ProducerMaxBufferedBytes   int64 `yaml:"producer_max_buffered_bytes"`

	TargetConsumerLagAtStartup time.Duration `yaml:"target_consumer_lag_at_startup"`
	MaxConsumerLagAtStartup    time.Duration `yaml:"max_consumer_lag_at_startup"`

	DisableKafkaTelemetry bool `yaml:"disable_kafka_telemetry"`

	ConsumerGroupLagMetricUpdateInterval time.Duration `yaml:"consumer_group_lag_metric_update_interval"`

	// The fetch backoff config to use in the concurrent fetchers (when enabled). This setting
	// is just used to change the default backoff in tests.
	concurrentFetchersFetchBackoffConfig backoff.Config `yaml:"-"`
}

func (cfg *KafkaConfig) RegisterFlags(f *flag.FlagSet) {
	cfg.RegisterFlagsWithPrefix("kafka", f)
}

func (cfg *KafkaConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Address, prefix+".address", "localhost:9092", "The Kafka backend address.")
	f.StringVar(&cfg.Topic, prefix+".topic", "", "The Kafka topic name.")
	f.StringVar(&cfg.ClientID, prefix+".client-id", "", "The Kafka client ID.")
	f.DurationVar(&cfg.DialTimeout, prefix+".dial-timeout", 2*time.Second, "The maximum time allowed to open a connection to a Kafka broker.")
	f.DurationVar(&cfg.WriteTimeout, prefix+".write-timeout", 10*time.Second, "How long to wait for an incoming write request to be successfully committed to the Kafka backend.")

	cfg.SASL.RegisterFlagsWithPrefix(prefix+".sasl-", f)
	f.BoolVar(&cfg.TLSEnabled, prefix+".tls-enabled", false, "Enable TLS for the Kafka client connection.")
	cfg.TLS.RegisterFlagsWithPrefix(prefix, f)

	f.StringVar(&cfg.ConsumerGroup, prefix+".consumer-group", "", "The consumer group used by the consumer to track the last consumed offset. The consumer group must be different for each ingester. If the configured consumer group contains the '<partition>' placeholder, it is replaced with the actual partition ID owned by the ingester. When empty (recommended), Tempo uses the ingester instance ID to guarantee uniqueness.")
	f.DurationVar(&cfg.ConsumerGroupOffsetCommitInterval, prefix+".consumer-group-offset-commit-interval", time.Second, "How frequently a consumer should commit the consumed offset to Kafka. The last committed offset is used at startup to continue the consumption from where it was left.")

	f.DurationVar(&cfg.LastProducedOffsetRetryTimeout, prefix+".last-produced-offset-retry-timeout", 10*time.Second, "How long to retry a failed request to get the last produced offset.")

	f.BoolVar(&cfg.AutoCreateTopicEnabled, prefix+".auto-create-topic-enabled", true, "Enable auto-creation of Kafka topic if it doesn't exist.")
	f.IntVar(&cfg.AutoCreateTopicDefaultPartitions, prefix+".auto-create-topic-default-partitions", 1000, "When auto-creation of Kafka topic is enabled and this value is positive, Kafka's num.partitions configuration option is set on Kafka brokers with this value when Tempo component that uses Kafka starts. This configuration option specifies the default number of partitions that the Kafka broker uses for auto-created topics. Note that this is a Kafka-cluster wide setting, and applies to any auto-created topic. If the setting of num.partitions fails, Tempo proceeds anyways, but auto-created topics could have an incorrect number of partitions.")

	f.IntVar(&cfg.ProducerMaxRecordSizeBytes, prefix+".producer-max-record-size-bytes", maxProducerRecordDataBytesLimit, "The maximum size of a Kafka record data that should be generated by the producer. An incoming write request larger than this size is split into multiple Kafka records. We strongly recommend to not change this setting unless for testing purposes.")
	f.Int64Var(&cfg.ProducerMaxBufferedBytes, prefix+".producer-max-buffered-bytes", 1024*1024*1024, "The maximum size of (uncompressed) buffered and unacknowledged produced records sent to Kafka. The produce request fails once this limit is reached. This limit is per Kafka client. 0 to disable the limit.")

	consumerLagUsage := fmt.Sprintf("Set both -%s and -%s to 0 to disable waiting for maximum consumer lag being honored at startup.", prefix+".target-consumer-lag-at-startup", prefix+".max-consumer-lag-at-startup")
	f.DurationVar(&cfg.TargetConsumerLagAtStartup, prefix+".target-consumer-lag-at-startup", 2*time.Second, "The best-effort maximum lag a consumer tries to achieve at startup. "+consumerLagUsage)
	f.DurationVar(&cfg.MaxConsumerLagAtStartup, prefix+".max-consumer-lag-at-startup", 15*time.Second, "The guaranteed maximum lag before a consumer is considered to have caught up reading from a partition at startup, becomes ACTIVE in the hash ring and passes the readiness check. "+consumerLagUsage)

	f.BoolVar(&cfg.DisableKafkaTelemetry, prefix+".disable-kafka-telemetry", false, "Disable KIP-714 Kafka client metrics")

	f.DurationVar(&cfg.ConsumerGroupLagMetricUpdateInterval, prefix+".consumer_group_lag_metric_update_interval", 1*time.Minute, "How often the lag metric is updated. Set to 0 to disable metric calculation and export ")
}

func (cfg *KafkaConfig) Validate() error {
	if cfg.Address == "" {
		return ErrMissingKafkaAddress
	}
	if cfg.Topic == "" {
		return ErrMissingKafkaTopic
	}
	if cfg.ProducerMaxRecordSizeBytes < minProducerRecordDataBytesLimit || cfg.ProducerMaxRecordSizeBytes > maxProducerRecordDataBytesLimit {
		return ErrInvalidProducerMaxRecordSizeBytes
	}
	if (cfg.TargetConsumerLagAtStartup == 0 && cfg.MaxConsumerLagAtStartup != 0) || (cfg.TargetConsumerLagAtStartup != 0 && cfg.MaxConsumerLagAtStartup == 0) {
		return ErrInconsistentConsumerLagAtStartup
	}
	if cfg.MaxConsumerLagAtStartup < cfg.TargetConsumerLagAtStartup {
		return ErrInvalidMaxConsumerLagAtStartup
	}

	if err := cfg.SASL.Validate(); err != nil {
		return err
	}

	if cfg.TLSEnabled {
		if _, err := cfg.TLS.GetTLSConfig(); err != nil {
			return fmt.Errorf("invalid Kafka TLS config: %w", err)
		}
	}

	return nil
}

// GetConsumerGroup returns the consumer group to use for the given instanceID and partitionID.
func (cfg *KafkaConfig) GetConsumerGroup(instanceID string, partitionID int32) string {
	if cfg.ConsumerGroup == "" {
		return instanceID
	}

	return strings.ReplaceAll(cfg.ConsumerGroup, "<partition>", strconv.Itoa(int(partitionID)))
}

// SetDefaultNumberOfPartitionsForAutocreatedTopics tries to set num.partitions config option on brokers.
// This is best-effort, if setting the option fails, error is logged, but not returned.
func (cfg KafkaConfig) SetDefaultNumberOfPartitionsForAutocreatedTopics(logger log.Logger) {
	if cfg.AutoCreateTopicDefaultPartitions <= 0 {
		return
	}

	opts, err := commonKafkaClientOptions(cfg, nil, logger)
	if err != nil {
		level.Error(logger).Log("msg", "failed to build kafka client options", "err", err)
		return
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		level.Error(logger).Log("msg", "failed to create kafka client", "err", err)
		return
	}

	adm := kadm.NewClient(cl)
	defer adm.Close()

	defaultNumberOfPartitions := fmt.Sprintf("%d", cfg.AutoCreateTopicDefaultPartitions)
	_, err = adm.AlterBrokerConfigsState(context.Background(), []kadm.AlterConfig{
		{
			Op:    kadm.SetConfig,
			Name:  "num.partitions",
			Value: &defaultNumberOfPartitions,
		},
	})
	if err != nil {
		level.Error(logger).Log("msg", "failed to alter default number of partitions", "err", err)
		return
	}

	level.Info(logger).Log("msg", "configured Kafka-wide default number of partitions for auto-created topics (num.partitions)", "value", cfg.AutoCreateTopicDefaultPartitions)
}

// KafkaAuthConfig holds the SASL authentication config for the Kafka backend.
type KafkaAuthConfig struct {
	Mechanism SASLMechanism `yaml:"sasl_mechanism"`

	// For PLAIN and SCRAM-SHA-* mechanisms.
	Username string         `yaml:"sasl_username"`
	Password flagext.Secret `yaml:"sasl_password"`

	Oauthbearer KafkaAuthOauthbearerConfig `yaml:",inline"`

	MSKIAM KafkaAuthMSKIAMConfig `yaml:",inline"`
}

func (cfg *KafkaAuthConfig) RegisterFlags(f *flag.FlagSet) {
	cfg.RegisterFlagsWithPrefix("", f)
}

func (cfg *KafkaAuthConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	cfg.Mechanism = SASLMechanismPlain
	f.Var(&cfg.Mechanism, prefix+"mechanism", fmt.Sprintf("The SASL mechanism used to authenticate to Kafka. Supported values: %s. For backwards-compatibility, PLAIN with no username nor password disables SASL.", strings.Join(saslMechanismOptions, ", ")))
	f.StringVar(&cfg.Username, prefix+"username", "", "The username used to authenticate to Kafka using SASL. To enable SASL, configure both the username and password.")
	f.Var(&cfg.Password, prefix+"password", "The password used to authenticate the username to Kafka using SASL. To enable SASL, configure both the username and password.")
	cfg.Oauthbearer.RegisterFlagsWithPrefix(prefix+"oauthbearer-", f)
	cfg.MSKIAM.RegisterFlagsWithPrefix(prefix+"msk-iam-", f)
}

func (cfg *KafkaAuthConfig) Validate() error {
	if cfg.Mechanism == "" {
		cfg.Mechanism = SASLMechanismPlain
	}
	switch cfg.Mechanism {
	case SASLMechanismPlain:
		if (cfg.Username == "") != (cfg.Password.String() == "") {
			return ErrInconsistentSASLCredentials
		}

	case SASLMechanismScramSHA256, SASLMechanismScramSHA512:
		if cfg.Username == "" || cfg.Password.String() == "" {
			return ErrInconsistentSASLCredentials
		}

	case SASLMechanismOauthbearer:
		return cfg.Oauthbearer.Validate()

	case SASLMechanismMSKIAM:
		return cfg.MSKIAM.Validate()

	default:
		return ErrInvalidSASLMechanism
	}

	return nil
}

// kafkaSASLConfig defines how to get a SASL secret: either from a static secret
// provided in config, from a file, or with HTTP through a domain socket.
//
// Exactly one source must be configured; call Validate to enforce this.
//
// Because this struct is generic, it cannot have distinct YAML struct tags for
// each kind of saslSecretConfig. Concrete structs with the same shape (so they
// can be converted to kafkaSASLConfig) are used instead.
type kafkaSASLConfig[T saslSecretConfig] struct {
	Secret            T
	FilePath          string
	HTTPSocketPath    string
	HTTPSocketTimeout time.Duration
}

var (
	errNoSecret               = errors.New("no static credentials provided")
	errIncompleteMSKIAMSecret = errors.New("both access key and secret key must be configured for static AWS_MSK_IAM credentials")
)

// Validate returns errNoSingleSource unless exactly one of the static secret,
// file path, or HTTP socket path is configured.
func (cfg kafkaSASLConfig[T]) Validate(errNoSingleSource error) error {
	err := cfg.Secret.Validate()
	if err != nil && !errors.Is(err, errNoSecret) {
		return err
	}
	hasStaticSecret := err == nil
	sourceFound := false
	for _, source := range []bool{
		hasStaticSecret,
		cfg.FilePath != "",
		cfg.HTTPSocketPath != "",
	} {
		if source {
			if sourceFound {
				return errNoSingleSource
			}
			sourceFound = true
		}
	}
	if !sourceFound {
		return errNoSingleSource
	}
	return nil
}

// KafkaAuthOauthbearerConfig holds OAUTHBEARER-specific SASL configuration.
type KafkaAuthOauthbearerConfig struct {
	Secret            KafkaOauthbearerStaticConfig `yaml:",inline"`
	FilePath          string                       `yaml:"sasl_oauthbearer_file_path"`
	HTTPSocketPath    string                       `yaml:"sasl_oauthbearer_http_socket_path"`
	HTTPSocketTimeout time.Duration                `yaml:"sasl_oauthbearer_http_socket_timeout"`
}

func (cfg *KafkaAuthOauthbearerConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	cfg.Secret.RegisterFlagsWithPrefix(prefix, f)
	f.StringVar(&cfg.FilePath, prefix+"file-path", "", `Path to a file containing an OAuth token to authenticate to Kafka. Mutually exclusive with `+prefix+`http-socket-path. The file is read anew on every reauthentication, so it can be updated with fresh tokens. The file must be in JSON format, adhering to this JSON schema: {"type": "object", "required": ["token"], "properties": {"token": {"type": "string"}, "zid": {"type": "string"}, "extensions": {"type": "object", "additionalProperties": {"type": "string"}}}}`)
	f.StringVar(&cfg.HTTPSocketPath, prefix+"http-socket-path", "", `Path to a Unix domain socket to fetch an OAuth token from via HTTP. Mutually exclusive with `+prefix+`file-path. On every authentication or reauthentication, an HTTP GET / request is made to the socket and the response body is read as JSON. The JSON schema is the same as for `+prefix+`file-path.`)
	f.DurationVar(&cfg.HTTPSocketTimeout, prefix+"http-socket-timeout", 10*time.Second, "Timeout for requesting the token from the HTTP socket. Effective when "+prefix+"http-socket-path is set.")
}

func (cfg *KafkaAuthOauthbearerConfig) Validate() error {
	if cfg.HTTPSocketPath != "" && cfg.HTTPSocketTimeout == 0 {
		cfg.HTTPSocketTimeout = 10 * time.Second
	}
	return kafkaSASLConfig[KafkaOauthbearerStaticConfig]{
		Secret:            cfg.Secret,
		FilePath:          cfg.FilePath,
		HTTPSocketPath:    cfg.HTTPSocketPath,
		HTTPSocketTimeout: cfg.HTTPSocketTimeout,
	}.Validate(ErrSASLOauthbearerBadConfig)
}

// KafkaOauthbearerStaticConfig holds static OAUTHBEARER credentials.
type KafkaOauthbearerStaticConfig struct {
	Token      flagext.Secret        `yaml:"sasl_oauthbearer_token"`
	Zid        string                `yaml:"sasl_oauthbearer_zid"`
	Extensions oauthbearerExtensions `yaml:"sasl_oauthbearer_extensions"`
}

// oauthbearerExtensions wraps flagext.LimitsMap so it can be decoded by
// Tempo's yaml.v2 config loader. flagext.LimitsMap only implements the
// yaml.v3 Unmarshaler interface (UnmarshalYAML(*yaml.Node)), which yaml.v2
// does not recognize; decoding sasl_oauthbearer_extensions directly into a
// LimitsMap therefore fails ("field not found") or, for a zero-value map,
// panics on assignment into its nil backing map. This wrapper provides a
// yaml.v2 Unmarshaler that initializes the map before loading values.
type oauthbearerExtensions struct {
	flagext.LimitsMap[string]
}

// UnmarshalYAML implements the yaml.v2 Unmarshaler interface.
func (e *oauthbearerExtensions) UnmarshalYAML(unmarshal func(interface{}) error) error {
	raw := map[string]string{}
	if err := unmarshal(&raw); err != nil {
		return err
	}
	if !e.IsInitialized() {
		e.LimitsMap = flagext.NewLimitsMap[string](nil)
	}
	data := e.LimitsMap.Read()
	clear(data)
	for k, v := range raw {
		data[k] = v
	}
	return nil
}

// Validate returns errNoSecret when no token has been set.
func (s KafkaOauthbearerStaticConfig) Validate() error {
	if s.Token.String() == "" {
		return errNoSecret
	}
	return nil
}

func (s *KafkaOauthbearerStaticConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.Var(&s.Token, prefix+"token", "The OAuth token to use to authenticate to Kafka. Consider "+prefix+"file-path instead.")
	f.StringVar(&s.Zid, prefix+"zid", "", "Optional authorization ID to use when authenticating to Kafka using SASL OAUTHBEARER.")
	if !s.Extensions.IsInitialized() {
		s.Extensions.LimitsMap = flagext.NewLimitsMap[string](nil)
	}
	f.Var(&s.Extensions.LimitsMap, prefix+"extensions", "Optional additional OAuth extensions to include when authenticating to Kafka using SASL OAUTHBEARER as a JSON object.")
}

// KafkaAuthMSKIAMConfig holds AWS_MSK_IAM-specific SASL configuration.
type KafkaAuthMSKIAMConfig struct {
	Secret            KafkaMSKIAMStaticConfig `yaml:",inline"`
	FilePath          string                  `yaml:"sasl_msk_iam_file_path"`
	HTTPSocketPath    string                  `yaml:"sasl_msk_iam_http_socket_path"`
	HTTPSocketTimeout time.Duration           `yaml:"sasl_msk_iam_http_socket_timeout"`
}

func (cfg *KafkaAuthMSKIAMConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	cfg.Secret.RegisterFlagsWithPrefix(prefix, f)
	f.StringVar(&cfg.FilePath, prefix+"file-path", "", `Path to a file containing AWS credentials to authenticate to Kafka using SASL AWS_MSK_IAM. Mutually exclusive with `+prefix+`http-socket-path. The file is read anew on every reauthentication, so it can be updated with fresh credentials. The file must be in JSON format, adhering to this JSON schema: {"type": "object", "required": ["AccessKey", "SecretKey"], "properties": {"AccessKey": {"type": "string"}, "SecretKey": {"type": "string"}, "SessionToken": {"type": "string"}, "UserAgent": {"type": "string"}}}`)
	f.StringVar(&cfg.HTTPSocketPath, prefix+"http-socket-path", "", `Path to a Unix domain socket to fetch AWS credentials from via HTTP. Mutually exclusive with `+prefix+`file-path. On every authentication or reauthentication, an HTTP GET / request is made to the socket and the response body is read as JSON. The JSON schema is the same as for `+prefix+`file-path.`)
	f.DurationVar(&cfg.HTTPSocketTimeout, prefix+"http-socket-timeout", 10*time.Second, "Timeout for requesting AWS credentials from the HTTP socket. Effective when "+prefix+"http-socket-path is set.")
}

func (cfg *KafkaAuthMSKIAMConfig) Validate() error {
	if cfg.HTTPSocketPath != "" && cfg.HTTPSocketTimeout == 0 {
		cfg.HTTPSocketTimeout = 10 * time.Second
	}
	return kafkaSASLConfig[KafkaMSKIAMStaticConfig](*cfg).Validate(ErrSASLMSKIAMBadConfig)
}

// KafkaMSKIAMStaticConfig holds static AWS_MSK_IAM credentials.
type KafkaMSKIAMStaticConfig struct {
	AccessKey    flagext.Secret `yaml:"sasl_msk_iam_access_key"`
	SecretKey    flagext.Secret `yaml:"sasl_msk_iam_secret_key"`
	SessionToken flagext.Secret `yaml:"sasl_msk_iam_session_token"`
	UserAgent    string         `yaml:"sasl_msk_iam_user_agent"`
}

// Validate returns errNoSecret when no access key and secret key have been set,
// and errIncompleteMSKIAMSecret when only one of them is set.
func (s KafkaMSKIAMStaticConfig) Validate() error {
	hasAccessKey := s.AccessKey.String() != ""
	hasSecretKey := s.SecretKey.String() != ""

	if !hasAccessKey && !hasSecretKey {
		return errNoSecret
	}
	if !hasAccessKey || !hasSecretKey {
		return errIncompleteMSKIAMSecret
	}
	return nil
}

func (s *KafkaMSKIAMStaticConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.Var(&s.AccessKey, prefix+"access-key", "The AWS access key ID to authenticate to Kafka using SASL AWS_MSK_IAM. Consider "+prefix+"file-path instead.")
	f.Var(&s.SecretKey, prefix+"secret-key", "The AWS secret access key to authenticate to Kafka using SASL AWS_MSK_IAM. Consider "+prefix+"file-path instead.")
	f.Var(&s.SessionToken, prefix+"session-token", "Optional AWS session token to authenticate to Kafka using SASL AWS_MSK_IAM.")
	f.StringVar(&s.UserAgent, prefix+"user-agent", "", "Optional user agent to use when authenticating to Kafka using SASL AWS_MSK_IAM.")
}

// TLSClientConfig holds the TLS configuration for the Kafka client.
type TLSClientConfig struct {
	dskittls.ClientConfig `yaml:",inline"`
}

func (c *TLSClientConfig) GetTLSConfig() (*tls.Config, error) {
	return c.ClientConfig.GetTLSConfig()
}
