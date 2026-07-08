package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	awssasl "github.com/twmb/franz-go/pkg/sasl/aws"
	"github.com/twmb/franz-go/pkg/sasl/oauth"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// kafkaAuthOptions returns the kgo options needed to enable SASL authentication
// for the configured mechanism. It returns nil options when SASL is disabled
// (PLAIN with no username) and an error if the configured mechanism
// or its credential sources are invalid.
func kafkaAuthOptions(cfg KafkaAuthConfig) ([]kgo.Opt, error) {
	m, ok, err := saslMechanism(cfg)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return []kgo.Opt{kgo.SASL(m)}, nil
}

// saslMechanism builds the franz-go SASL mechanism for the configured
// authentication mechanism. It first validates the config (normalizing an unset
// mechanism to PLAIN). The bool is false (with a nil mechanism and nil error)
// when SASL is disabled, which is the case for the PLAIN mechanism with no
// username configured. An invalid config returns an error.
func saslMechanism(cfg KafkaAuthConfig) (sasl.Mechanism, bool, error) {
	if err := (&cfg).Validate(); err != nil {
		return nil, false, err
	}
	// For backwards compatibility, PLAIN with no username
	// is treated as SASL disabled.
	if cfg.Mechanism == SASLMechanismPlain && cfg.Username == "" {
		return nil, false, nil
	}

	switch cfg.Mechanism {
	case SASLMechanismScramSHA256:
		return scram.Auth{
			User: cfg.Username,
			Pass: cfg.Password.String(),
		}.AsSha256Mechanism(), true, nil
	case SASLMechanismScramSHA512:
		return scram.Auth{
			User: cfg.Username,
			Pass: cfg.Password.String(),
		}.AsSha512Mechanism(), true, nil
	case SASLMechanismPlain:
		return plain.Auth{
			User: cfg.Username,
			Pass: cfg.Password.String(),
		}.AsMechanism(), true, nil
	case SASLMechanismOauthbearer:
		return cfg.Oauthbearer.mechanism(), true, nil
	case SASLMechanismMSKIAM:
		return cfg.MSKIAM.mechanism(), true, nil
	default:
		return nil, false, fmt.Errorf("unknown SASL mechanism: %v", cfg.Mechanism)
	}
}

// saslSecretConfig configures a static secret. It may be empty.
type saslSecretConfig interface {
	// Validate returns errNoSecret when no static secret is set. It may return
	// other validation errors.
	Validate() error
	// mechanism constructs a sasl.Mechanism from the static secret, if one is set.
	mechanism() (sasl.Mechanism, bool)
}

func (cfg KafkaAuthOauthbearerConfig) mechanism() sasl.Mechanism {
	return saslMechanismFromSources(kafkaSASLConfig[KafkaOauthbearerStaticConfig](cfg), oauth.Oauth)
}

func (s KafkaOauthbearerStaticConfig) mechanism() (sasl.Mechanism, bool) {
	if err := s.Validate(); err != nil {
		return nil, false
	}
	return oauth.Auth{
		Token:      s.Token.String(),
		Zid:        s.Zid,
		Extensions: s.Extensions.Read(),
	}.AsMechanism(), true
}

func (cfg KafkaAuthMSKIAMConfig) mechanism() sasl.Mechanism {
	return saslMechanismFromSources(kafkaSASLConfig[KafkaMSKIAMStaticConfig](cfg), awssasl.ManagedStreamingIAM)
}

func (s KafkaMSKIAMStaticConfig) mechanism() (sasl.Mechanism, bool) {
	if err := s.Validate(); err != nil {
		return nil, false
	}
	return awssasl.Auth{
		AccessKey:    s.AccessKey.String(),
		SecretKey:    s.SecretKey.String(),
		SessionToken: s.SessionToken.String(),
		UserAgent:    s.UserAgent,
	}.AsManagedStreamingIAMMechanism(), true
}

// saslMechanismFromSources returns the sasl.Mechanism to be passed to the Kafka
// client, resolving the secret from a static value, a file, or an HTTP socket.
//
// Validate must have been called on cfg beforehand: a config with no source
// configured panics because it indicates a programmer error.
func saslMechanismFromSources[T saslSecretConfig, A any](cfg kafkaSASLConfig[T], fromCallback func(func(context.Context) (A, error)) sasl.Mechanism) sasl.Mechanism {
	if m, ok := cfg.Secret.mechanism(); ok {
		return m
	}
	if cfg.FilePath != "" {
		return fromCallback(func(_ context.Context) (A, error) {
			f, err := os.ReadFile(cfg.FilePath)
			if err != nil {
				var zero A
				return zero, err
			}
			var a A
			err = json.Unmarshal(f, &a)
			return a, err
		})
	}
	if cfg.HTTPSocketPath != "" {
		return fromCallback(func(ctx context.Context) (A, error) {
			return requestJSONFromSocket[A](ctx, cfg.HTTPSocketPath, cfg.HTTPSocketTimeout)
		})
	}
	panic("invalid kafkaSASLConfig: Validate must be called first")
}

// requestJSONFromSocket performs an HTTP GET against a Unix domain socket and
// decodes the JSON response body into a value of type T.
func requestJSONFromSocket[T any](ctx context.Context, socketPath string, timeout time.Duration) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	transport := &http.Transport{
		Proxy: nil,
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}

	client := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://credentials/", nil)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("creating request for HTTP socket %s: %w", socketPath, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("requesting credentials from HTTP socket %s: %w", socketPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var zero T
		return zero, fmt.Errorf("requesting credentials from HTTP socket %s: unexpected status %s", socketPath, resp.Status)
	}

	var a T
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		var zero T
		return zero, fmt.Errorf("parsing credentials from HTTP socket %s: %w", socketPath, err)
	}
	return a, nil
}
