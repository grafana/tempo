package ingest

import (
	"context"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/sasl"
	awssasl "github.com/twmb/franz-go/pkg/sasl/aws"
	"github.com/twmb/franz-go/pkg/sasl/oauth"
	yamlv2 "go.yaml.in/yaml/v2"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

// validSASLBaseConfig returns a KafkaConfig with defaults applied and the
// minimum required fields set so that Validate() passes. Individual tests
// mutate the SASL fields to exercise the mechanism selection logic.
func validSASLBaseConfig() KafkaConfig {
	cfg := KafkaConfig{}
	flagext.DefaultValues(&cfg)
	cfg.Address = "localhost:9092"
	cfg.Topic = "tempo"
	return cfg
}

func TestKafkaConfig_RegisterFlags_DefaultSASLMechanism(t *testing.T) {
	cfg := KafkaConfig{}
	flagext.DefaultValues(&cfg)

	require.Equal(t, SASLMechanismPlain, cfg.SASL.Mechanism)
}

func TestSASLMechanism_Set(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  SASLMechanism
		err   bool
	}{
		{"plain", "PLAIN", SASLMechanismPlain, false},
		{"scram sha256", "SCRAM-SHA-256", SASLMechanismScramSHA256, false},
		{"scram sha512", "SCRAM-SHA-512", SASLMechanismScramSHA512, false},
		{"oauthbearer", "OAUTHBEARER", SASLMechanismOauthbearer, false},
		{"aws msk iam", "AWS_MSK_IAM", SASLMechanismMSKIAM, false},
		{"unknown", "NOPE", "", true},
		{"empty", "", "", true},
		{"wrong case", "plain", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m SASLMechanism
			err := m.Set(tt.value)
			if tt.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, m)
		})
	}
}

// TestKafkaConfig_RegisterFlagsWithPrefix_SASLMechanism ensures the mechanism
// can be configured through the CLI flag, that the flag uses the dotted
// convention, and that invalid values are rejected at parse time.
func TestKafkaConfig_RegisterFlagsWithPrefix_SASLMechanism(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want SASLMechanism
		err  bool
	}{
		{"default when unset", nil, SASLMechanismPlain, false},
		{"explicit scram", []string{"-kafka.sasl-mechanism", "SCRAM-SHA-512"}, SASLMechanismScramSHA512, false},
		{"explicit msk iam", []string{"-kafka.sasl-mechanism", "AWS_MSK_IAM"}, SASLMechanismMSKIAM, false},
		{"invalid value rejected", []string{"-kafka.sasl-mechanism", "NOPE"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := KafkaConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlags(fs)

			err := fs.Parse(tt.args)
			if tt.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, cfg.SASL.Mechanism)
		})
	}
}

// TestKafkaConfig_RegisterFlagsWithPrefix_DottedFlags guards against the
// flag-prefix regression where SASL/TLS flags were registered without the
// leading dot (e.g. "kafkasasl-mechanism").
func TestKafkaConfig_RegisterFlagsWithPrefix_DottedFlags(t *testing.T) {
	cfg := KafkaConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.RegisterFlags(fs)

	for _, name := range []string{
		"kafka.sasl-mechanism",
		"kafka.sasl-username",
		"kafka.sasl-password",
		"kafka.sasl-oauthbearer-token",
		"kafka.sasl-msk-iam-access-key",
		"kafka.tls-enabled",
	} {
		require.NotNil(t, fs.Lookup(name), "expected flag %q to be registered", name)
	}
}

func TestKafkaConfig_Validate_SASLMechanism(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*KafkaConfig)
		wantErr error
	}{
		{
			name:   "no sasl configured is valid",
			mutate: func(_ *KafkaConfig) {},
		},
		{
			name: "plain with credentials is valid",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismPlain
				c.SASL.Username = "user"
				c.SASL.Password = flagext.SecretWithValue("pass")
			},
		},
		{
			name: "scram with credentials is valid",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismScramSHA256
				c.SASL.Username = "user"
				c.SASL.Password = flagext.SecretWithValue("pass")
			},
		},
		{
			name: "username without password is inconsistent",
			mutate: func(c *KafkaConfig) {
				c.SASL.Username = "user"
			},
			wantErr: ErrInconsistentSASLCredentials,
		},
		{
			name: "password without username is inconsistent",
			mutate: func(c *KafkaConfig) {
				c.SASL.Password = flagext.SecretWithValue("pass")
			},
			wantErr: ErrInconsistentSASLCredentials,
		},
		{
			name: "scram without credentials is rejected",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismScramSHA512
			},
			wantErr: ErrInconsistentSASLCredentials,
		},
		{
			name: "invalid mechanism value is rejected",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanism("NOPE")
			},
			wantErr: ErrInvalidSASLMechanism,
		},
		{
			name: "oauthbearer with static token is valid",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismOauthbearer
				c.SASL.Oauthbearer.Secret.Token = flagext.SecretWithValue("token")
			},
		},
		{
			name: "oauthbearer with no source is rejected",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismOauthbearer
			},
			wantErr: ErrSASLOauthbearerBadConfig,
		},
		{
			name: "oauthbearer with two sources is rejected",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismOauthbearer
				c.SASL.Oauthbearer.Secret.Token = flagext.SecretWithValue("token")
				c.SASL.Oauthbearer.FilePath = "/tmp/token.json"
			},
			wantErr: ErrSASLOauthbearerBadConfig,
		},
		{
			name: "msk iam with static credentials is valid",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismMSKIAM
				c.SASL.MSKIAM.Secret.AccessKey = flagext.SecretWithValue("ak")
				c.SASL.MSKIAM.Secret.SecretKey = flagext.SecretWithValue("sk")
			},
		},
		{
			name: "msk iam with incomplete static credentials is rejected",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismMSKIAM
				c.SASL.MSKIAM.Secret.AccessKey = flagext.SecretWithValue("ak")
			},
			wantErr: errIncompleteMSKIAMSecret,
		},
		{
			name: "msk iam with no source is rejected",
			mutate: func(c *KafkaConfig) {
				c.SASL.Mechanism = SASLMechanismMSKIAM
			},
			wantErr: ErrSASLMSKIAMBadConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validSASLBaseConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestKafkaAuthOptions verifies that kafkaAuthOptions selects the correct
// franz-go SASL mechanism for each configured value. The mechanism name
// reported by the franz-go client is used as the observable assertion.
func TestKafkaAuthOptions(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(*KafkaAuthConfig)
		wantSASL     bool
		wantSASLName string
	}{
		{
			name:     "plain without credentials disables sasl",
			mutate:   func(c *KafkaAuthConfig) { c.Mechanism = SASLMechanismPlain },
			wantSASL: false,
		},
		{
			name: "plain with credentials",
			mutate: func(c *KafkaAuthConfig) {
				c.Mechanism = SASLMechanismPlain
				c.Username = "user"
				c.Password = flagext.SecretWithValue("pass")
			},
			wantSASL:     true,
			wantSASLName: "PLAIN",
		},
		{
			name: "scram sha256",
			mutate: func(c *KafkaAuthConfig) {
				c.Mechanism = SASLMechanismScramSHA256
				c.Username = "user"
				c.Password = flagext.SecretWithValue("pass")
			},
			wantSASL:     true,
			wantSASLName: "SCRAM-SHA-256",
		},
		{
			name: "scram sha512",
			mutate: func(c *KafkaAuthConfig) {
				c.Mechanism = SASLMechanismScramSHA512
				c.Username = "user"
				c.Password = flagext.SecretWithValue("pass")
			},
			wantSASL:     true,
			wantSASLName: "SCRAM-SHA-512",
		},
		{
			name: "oauthbearer static token",
			mutate: func(c *KafkaAuthConfig) {
				c.Mechanism = SASLMechanismOauthbearer
				c.Oauthbearer.Secret.Token = flagext.SecretWithValue("token")
			},
			wantSASL:     true,
			wantSASLName: "OAUTHBEARER",
		},
		{
			name: "aws msk iam static credentials",
			mutate: func(c *KafkaAuthConfig) {
				c.Mechanism = SASLMechanismMSKIAM
				c.MSKIAM.Secret.AccessKey = flagext.SecretWithValue("ak")
				c.MSKIAM.Secret.SecretKey = flagext.SecretWithValue("sk")
			},
			wantSASL:     true,
			wantSASLName: "AWS_MSK_IAM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := KafkaAuthConfig{}
			cfg.RegisterFlagsWithPrefix("", flag.NewFlagSet("", flag.PanicOnError))
			tt.mutate(&cfg)

			// The generated kgo option is opaque, so assert the selected
			// mechanism through the helper it delegates to.
			m, ok, err := saslMechanism(cfg)
			require.NoError(t, err)
			opts, err := kafkaAuthOptions(cfg)
			require.NoError(t, err)

			if !tt.wantSASL {
				require.False(t, ok)
				require.Nil(t, m)
				require.Empty(t, opts)
				return
			}

			require.True(t, ok)
			require.NotNil(t, m)
			require.Equal(t, tt.wantSASLName, m.Name())
			require.Len(t, opts, 1)
		})
	}
}

// TestKafkaAuthOptions_UnknownMechanism verifies that an unknown SASL mechanism
// is reported as an error rather than panicking, so a caller that skips
// KafkaAuthConfig.Validate cannot crash the process.
func TestKafkaAuthOptions_UnknownMechanism(t *testing.T) {
	cfg := KafkaAuthConfig{
		Mechanism: SASLMechanism("NOPE"),
		Username:  "user",
		Password:  flagext.SecretWithValue("pass"),
	}

	require.NotPanics(t, func() {
		m, ok, err := saslMechanism(cfg)
		require.Error(t, err)
		require.False(t, ok)
		require.Nil(t, m)

		opts, err := kafkaAuthOptions(cfg)
		require.Error(t, err)
		require.Nil(t, opts)
	})
}

// TestOauthbearerCredentialsFromSources verifies OAUTHBEARER tokens are read
// from a file and from a Unix domain socket, and resolved lazily by the
// mechanism callback.
func TestOauthbearerCredentialsFromSources(t *testing.T) {
	secret := oauth.Auth{Token: "some-oauth-token", Zid: "some-zid"}

	setups := map[string]func(t *testing.T) kafkaSASLConfig[KafkaOauthbearerStaticConfig]{
		"file-based": func(t *testing.T) kafkaSASLConfig[KafkaOauthbearerStaticConfig] {
			var cfg KafkaAuthOauthbearerConfig
			cfg.RegisterFlagsWithPrefix("", flag.NewFlagSet("", flag.PanicOnError))
			cfg.FilePath = writeSecretToFile(t, secret)
			return kafkaSASLConfig[KafkaOauthbearerStaticConfig](cfg)
		},
		"socket-based": func(t *testing.T) kafkaSASLConfig[KafkaOauthbearerStaticConfig] {
			var cfg KafkaAuthOauthbearerConfig
			cfg.RegisterFlagsWithPrefix("", flag.NewFlagSet("", flag.PanicOnError))
			cfg.HTTPSocketPath = serveSecretFromSocket(t, secret)
			return kafkaSASLConfig[KafkaOauthbearerStaticConfig](cfg)
		},
	}

	for how, setUp := range setups {
		t.Run(how, func(t *testing.T) {
			cfg := setUp(t)

			var gotCallback func(context.Context) (oauth.Auth, error)
			gotMechanism := saslMechanismFromSources(cfg, func(cb func(context.Context) (oauth.Auth, error)) sasl.Mechanism {
				gotCallback = cb
				return oauth.Oauth(cb)
			})
			require.NotNil(t, gotCallback)
			require.NotNil(t, gotMechanism)

			gotSecret, err := gotCallback(context.Background())
			require.NoError(t, err)
			require.Equal(t, secret, gotSecret)
		})
	}
}

// TestMSKIAMCredentialsFromSources verifies AWS_MSK_IAM credentials are read
// from a file and from a Unix domain socket.
func TestMSKIAMCredentialsFromSources(t *testing.T) {
	secret := awssasl.Auth{
		AccessKey:    "AKIDEXAMPLE",
		SecretKey:    "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		SessionToken: "AQoDYXdzEJr//some/session/token",
	}

	setups := map[string]func(t *testing.T) kafkaSASLConfig[KafkaMSKIAMStaticConfig]{
		"file-based": func(t *testing.T) kafkaSASLConfig[KafkaMSKIAMStaticConfig] {
			var cfg KafkaAuthMSKIAMConfig
			cfg.RegisterFlagsWithPrefix("", flag.NewFlagSet("", flag.PanicOnError))
			cfg.FilePath = writeSecretToFile(t, secret)
			return kafkaSASLConfig[KafkaMSKIAMStaticConfig](cfg)
		},
		"socket-based": func(t *testing.T) kafkaSASLConfig[KafkaMSKIAMStaticConfig] {
			var cfg KafkaAuthMSKIAMConfig
			cfg.RegisterFlagsWithPrefix("", flag.NewFlagSet("", flag.PanicOnError))
			cfg.HTTPSocketPath = serveSecretFromSocket(t, secret)
			return kafkaSASLConfig[KafkaMSKIAMStaticConfig](cfg)
		},
	}

	for how, setUp := range setups {
		t.Run(how, func(t *testing.T) {
			cfg := setUp(t)

			var gotCallback func(context.Context) (awssasl.Auth, error)
			gotMechanism := saslMechanismFromSources(cfg, func(cb func(context.Context) (awssasl.Auth, error)) sasl.Mechanism {
				gotCallback = cb
				return awssasl.ManagedStreamingIAM(cb)
			})
			require.NotNil(t, gotCallback)
			require.NotNil(t, gotMechanism)

			gotSecret, err := gotCallback(context.Background())
			require.NoError(t, err)
			require.Equal(t, secret, gotSecret)
		})
	}
}

// TestKafkaConfig_Validate_TLS ensures TLS misconfiguration is surfaced when
// TLS is enabled and ignored when it is disabled.
func TestKafkaConfig_Validate_TLS(t *testing.T) {
	t.Run("enabled with bad cert path fails", func(t *testing.T) {
		cfg := validSASLBaseConfig()
		cfg.TLSEnabled = true
		cfg.TLS.CertPath = "/does/not/exist.pem" // key missing -> invalid pair
		require.Error(t, cfg.Validate())
	})

	t.Run("disabled ignores TLS config", func(t *testing.T) {
		cfg := validSASLBaseConfig()
		cfg.TLSEnabled = false
		cfg.TLS.CertPath = "/does/not/exist.pem"
		require.NoError(t, cfg.Validate())
	})
}

// TestKafkaClientConstructors_TLSError verifies that a failure building the TLS
// config surfaces as an error from the client constructors rather than a panic.
// This guards the error plumbing in commonKafkaClientOptions.
func TestKafkaClientConstructors_TLSError(t *testing.T) {
	badTLSConfig := func() KafkaConfig {
		cfg := validSASLBaseConfig()
		cfg.TLSEnabled = true
		// A certificate without a key is an invalid pair, so GetTLSConfig fails
		// without any network I/O.
		cfg.TLS.CertPath = "/does/not/exist.pem"
		return cfg
	}

	t.Run("NewWriterClient", func(t *testing.T) {
		require.NotPanics(t, func() {
			_, err := NewWriterClient(badTLSConfig(), 1, log.NewNopLogger(), prometheus.NewPedanticRegistry())
			require.Error(t, err)
		})
	})

	t.Run("NewReaderClient", func(t *testing.T) {
		require.NotPanics(t, func() {
			_, err := NewReaderClient(badTLSConfig(), nil, log.NewNopLogger())
			require.Error(t, err)
		})
	})
}

func writeSecretToFile(t *testing.T, secret any) string {
	t.Helper()

	js, err := json.Marshal(secret)
	require.NoError(t, err)
	filePath := filepath.Join(t.TempDir(), "secret.json")
	require.NoError(t, os.WriteFile(filePath, js, 0o600))

	return filePath
}

func serveSecretFromSocket(t *testing.T, secret any) string {
	t.Helper()

	js, err := json.Marshal(secret)
	require.NoError(t, err)

	// Unix socket paths have a low length limit (~104 bytes on macOS), so use a
	// short base directory under /tmp rather than t.TempDir().
	sockDir, err := os.MkdirTemp("/tmp", "tempo-kafka")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(sockDir) })

	socketPath := filepath.Join(sockDir, "secret.sock")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)

	server := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(js)
		}),
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	return socketPath
}

// TestOauthbearerExtensionsYAML verifies that sasl_oauthbearer_extensions
// decodes correctly through Tempo's yaml.v2 config loader. flagext.LimitsMap
// only implements the yaml.v3 Unmarshaler interface, so without the
// oauthbearerExtensions wrapper this field either fails to decode or panics
// on a nil backing map when no flags have been registered first.
func TestOauthbearerExtensionsYAML(t *testing.T) {
	t.Run("direct type", func(t *testing.T) {
		var cfg KafkaOauthbearerStaticConfig
		y := []byte("sasl_oauthbearer_token: tok\n" +
			"sasl_oauthbearer_zid: zid\n" +
			"sasl_oauthbearer_extensions:\n  foo: bar\n  baz: qux\n")

		require.NotPanics(t, func() {
			require.NoError(t, yamlv2.UnmarshalStrict(y, &cfg))
		})
		require.Equal(t, "tok", cfg.Token.String())
		require.Equal(t, "zid", cfg.Zid)
		require.Equal(t, map[string]string{"foo": "bar", "baz": "qux"}, cfg.Extensions.Read())
	})

	t.Run("inline chain via KafkaConfig", func(t *testing.T) {
		var cfg KafkaConfig
		y := []byte("sasl_oauthbearer_extensions:\n  a: b\n")

		require.NotPanics(t, func() {
			require.NoError(t, yamlv2.Unmarshal(y, &cfg))
		})
		require.Equal(t, map[string]string{"a": "b"}, cfg.SASL.Oauthbearer.Secret.Extensions.Read())
	})

	t.Run("extensions absent does not panic", func(t *testing.T) {
		var cfg KafkaConfig
		require.NotPanics(t, func() {
			require.NoError(t, yamlv2.Unmarshal([]byte("sasl_oauthbearer_token: tok\n"), &cfg))
		})
	})
}
