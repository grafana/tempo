package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	azure "github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
)

const (
	OverridesKeyPath  = "overrides"
	OverridesFileName = "overrides.json"
)

// ErrInvalidSecretsPolicy identifies an invalid persisted policy. Reloads may
// retain the last good policy, but ordinary override decode failures remain fatal.
var ErrInvalidSecretsPolicy = errors.New("invalid secret detection policy JSON")

var tracer = otel.Tracer("modules/overrides/userconfigurable/client")

var (
	metricList = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "overrides_user_configurable_overrides_list_total",
		Help:      "How often the user-configurable overrides was listed",
	})
	metricFetch = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "overrides_user_configurable_overrides_fetch_total",
		Help:      "How often the user-configurable overrides was fetched for this tenant",
	}, []string{"tenant"})
	metricFetchFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "overrides_user_configurable_overrides_fetch_failed_total",
		Help:      "How often fetching the user-configurable overrides failed for this tenant",
	}, []string{"tenant"})
)

type Config struct {
	Backend string `yaml:"backend"`

	// ConfirmVersioning is enabled when creating the backend client. If versioning is disabled no
	// checks against concurrent writes will be performed.
	ConfirmVersioning bool `yaml:"confirm_versioning"`

	Local *local.Config `yaml:"local"`
	GCS   *gcs.Config   `yaml:"gcs"`
	S3    *s3.Config    `yaml:"s3"`
	Azure *azure.Config `yaml:"azure"`
}

func (c *Config) RegisterFlagsAndApplyDefaults(*flag.FlagSet) {
	c.ConfirmVersioning = true

	// pass in a dummy flagset because we don't want to set any flags for this module
	dummyFlagSet := &flag.FlagSet{}

	c.Local = &local.Config{}
	c.Local.RegisterFlagsAndApplyDefaults("", dummyFlagSet)
	c.GCS = &gcs.Config{}
	c.GCS.RegisterFlagsAndApplyDefaults("", dummyFlagSet)
	c.S3 = &s3.Config{}
	c.S3.RegisterFlagsAndApplyDefaults("", dummyFlagSet)
	c.Azure = &azure.Config{}
	c.Azure.RegisterFlagsAndApplyDefaults("", dummyFlagSet)
}

// Client is a collection of methods to manage overrides on a backend.
type Client interface {
	// List tenants that have user-configurable overrides.
	List(ctx context.Context) ([]string, error)
	// Get the user-configurable overrides. Returns backend.ErrDoesNotExist if no limits are set.
	Get(context.Context, string) (*Limits, backend.Version, error)
	// Set the user-configurable overrides. Returns backend.ErrVersionDoesNotMatch if the backend
	// has a newer version.
	Set(context.Context, string, *Limits, backend.Version) (backend.Version, error)
	// Delete the user-configurable overrides.
	Delete(context.Context, string, backend.Version) error
	// Shutdown the client.
	Shutdown()
}

type clientImpl struct {
	rw backend.VersionedReaderWriter
}

var _ Client = (*clientImpl)(nil)

// Persisted overrides may contain newer non-policy fields, but an unsupported
// policy must not be mistaken for an empty replacement that erases exclusions.
type persistedPolicy secrets.Policy

func (p *persistedPolicy) UnmarshalJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode((*secrets.Policy)(p)); err != nil {
		return ErrInvalidSecretsPolicy
	}
	return nil
}

// HasSecretsPolicy reports a policy field in the first JSON document, including
// a null or malformed value. Token decoding recognizes escaped and case-folded
// field names without treating policy-like text in ordinary values as a policy.
func HasSecretsPolicy(data []byte) bool {
	found, _ := findSecretsPolicy(json.NewDecoder(bytes.NewReader(data)), 0)
	return found
}

func findSecretsPolicy(d *json.Decoder, field int) (bool, error) {
	token, err := d.Token()
	if err != nil {
		return false, err
	}
	switch token {
	case json.Delim('{'):
		fields := [...]string{"metrics_generator", "processor", "secret_detection"}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return false, err
			}
			next := -1
			if field >= 0 && strings.EqualFold(key.(string), fields[field]) {
				if field == len(fields)-1 {
					return true, nil
				}
				next = field + 1
			}
			if found, err := findSecretsPolicy(d, next); found || err != nil {
				return found, err
			}
		}
		_, err = d.Token()
	case json.Delim('['):
		for d.More() {
			if found, err := findSecretsPolicy(d, -1); found || err != nil {
				return found, err
			}
		}
		_, err = d.Token()
	}
	return false, err
}

// JSONDecodeError retains safe error categories and offsets, not untrusted field
// names or values. Even malformed JSON can contain policy text before decoding
// reaches the policy field.
func JSONDecodeError(err error) error {
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	switch {
	case errors.Is(err, ErrInvalidSecretsPolicy):
		return ErrInvalidSecretsPolicy
	case errors.As(err, &syntaxError):
		return fmt.Errorf("invalid overrides JSON syntax at byte %d", syntaxError.Offset)
	case errors.As(err, &typeError):
		return fmt.Errorf("invalid overrides JSON value at byte %d: expected %s", typeError.Offset, typeError.Type)
	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("incomplete overrides JSON document")
	case errors.Is(err, io.EOF):
		return err
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		return errors.New("unknown field in overrides JSON")
	default:
		return errors.New("invalid overrides JSON")
	}
}

func New(cfg *Config) (Client, error) {
	rw, err := initBackend(cfg)
	if err != nil {
		return nil, err
	}
	return &clientImpl{rw}, nil
}

func (o *clientImpl) Shutdown() {
	o.rw.Shutdown()
}

func initBackend(cfg *Config) (rw backend.VersionedReaderWriter, err error) {
	switch cfg.Backend {
	case backend.Local:
		r, w, _, err := local.New(cfg.Local)
		if err != nil {
			return nil, err
		}
		// Create overrides directory with necessary permissions
		err = os.MkdirAll(path.Join(cfg.Local.Path, OverridesKeyPath), 0o700)
		if err != nil {
			return nil, err
		}
		rw = backend.NewFakeVersionedReaderWriter(r, w)
	case backend.GCS:
		rw, err = gcs.NewVersionedReaderWriter(cfg.GCS, cfg.ConfirmVersioning)
	case backend.S3:
		rw, err = s3.NewVersionedReaderWriter(cfg.S3)
	case backend.Azure:
		rw, err = azure.NewVersionedReaderWriter(cfg.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.Backend)
	}
	if err != nil {
		return nil, err
	}
	if cfg.Backend == backend.Local || cfg.Backend == backend.S3 || cfg.Backend == backend.Azure {
		level.Warn(log.Logger).Log(
			"msg", "versioned backend requests are best-effort for the configured backend, concurrent requests modifying user-configurable overrides might cause data races",
			"backend", cfg.Backend,
		)
	}
	return rw, nil
}

func (o *clientImpl) List(ctx context.Context) ([]string, error) {
	ctx, span := tracer.Start(ctx, "clientImpl.List")
	defer span.End()

	metricList.Inc()

	return o.rw.List(ctx, []string{OverridesKeyPath})
}

func (o *clientImpl) Get(ctx context.Context, userID string) (tenantLimits *Limits, version backend.Version, err error) {
	ctx, span := tracer.Start(ctx, "clientImpl.Get", trace.WithAttributes(attribute.String("tenant", userID)))
	defer span.End()

	metricFetch.WithLabelValues(userID).Inc()
	defer func() {
		if err != nil {
			metricFetchFailed.WithLabelValues(userID).Inc()
		}
	}()

	reader, version, err := o.rw.ReadVersioned(ctx, OverridesFileName, []string{OverridesKeyPath, userID})
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()

	tenantLimits = &Limits{}
	document := &struct {
		*Limits
		MetricsGenerator struct {
			*LimitsMetricsGenerator
			Processor struct {
				*LimitsMetricsGeneratorProcessor
				SecretDetection *persistedPolicy `json:"secret_detection"`
			} `json:"processor"`
		} `json:"metrics_generator"`
	}{Limits: tenantLimits}
	document.MetricsGenerator.LimitsMetricsGenerator = &tenantLimits.MetricsGenerator
	document.MetricsGenerator.Processor.LimitsMetricsGeneratorProcessor = &tenantLimits.MetricsGenerator.Processor

	var input bytes.Buffer
	d := json.NewDecoder(io.TeeReader(reader, &input))
	if err = d.Decode(&document); err != nil {
		var syntaxError *json.SyntaxError
		if errors.Is(err, ErrInvalidSecretsPolicy) || ((errors.As(err, &syntaxError) || errors.Is(err, io.ErrUnexpectedEOF)) && HasSecretsPolicy(input.Bytes())) {
			return nil, version, ErrInvalidSecretsPolicy
		}
		return nil, version, JSONDecodeError(err)
	}
	if document == nil {
		return nil, version, nil
	}
	// Keep the existing decoder behavior for non-secret overrides. Only the
	// newly introduced policy requires a complete, unambiguous document.
	if document.MetricsGenerator.Processor.SecretDetection != nil || HasSecretsPolicy(input.Bytes()) {
		if err = d.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
			return nil, version, ErrInvalidSecretsPolicy
		}
	}
	tenantLimits.MetricsGenerator.Processor.SecretDetection = (*secrets.Policy)(document.MetricsGenerator.Processor.SecretDetection)
	err = nil
	return
}

func (o *clientImpl) Set(ctx context.Context, userID string, limits *Limits, version backend.Version) (backend.Version, error) {
	ctx, span := tracer.Start(ctx, "clientImpl.Set", trace.WithAttributes(attribute.String("tenant", userID)))
	defer span.End()

	data, err := json.Marshal(limits)
	if err != nil {
		return "", err
	}

	return o.rw.WriteVersioned(ctx, OverridesFileName, []string{OverridesKeyPath, userID}, bytes.NewReader(data), int64(len(data)), version)
}

func (o *clientImpl) Delete(ctx context.Context, userID string, version backend.Version) error {
	ctx, span := tracer.Start(ctx, "clientImpl.Delete", trace.WithAttributes(attribute.String("tenant", userID)))
	defer span.End()

	return o.rw.DeleteVersioned(ctx, OverridesFileName, []string{OverridesKeyPath, userID}, version)
}
