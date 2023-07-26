package userconfigurableapi

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/go-kit/log/level"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
)

const (
	OverridesKeyPath  = "overrides"
	OverridesFileName = "overrides.json"
)

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

type UserConfigurableOverridesClientConfig struct {
	Backend string `yaml:"backend"`

	// ConfirmVersioning is enabled when creating the backend client. If versioning is disabled no
	// checks against concurrent writes will be performed.
	ConfirmVersioning bool `yaml:"confirm_versioning"`

	Local *local.Config `yaml:"local"`
	GCS   *gcs.Config   `yaml:"gcs"`
	S3    *s3.Config    `yaml:"s3"`
	Azure *azure.Config `yaml:"azure"`
}

func (c *UserConfigurableOverridesClientConfig) RegisterFlagsAndApplyDefaults(*flag.FlagSet) {
	c.ConfirmVersioning = true

	c.Local = &local.Config{}
	c.GCS = &gcs.Config{}
	c.S3 = &s3.Config{}
	c.Azure = &azure.Config{}
}

// Client is a collection of methods to manage overrides on a backend.
type Client interface {
	// List tenants that have user-configurable overrides.
	List(ctx context.Context) ([]string, error)
	// Get the user-configurable overrides. Returns backend.ErrDoesNotExist if no limits are set.
	Get(context.Context, string) (*UserConfigurableLimits, backend.Version, error)
	// Set the user-configurable overrides. Returns backend.ErrVersionDoesNotMatch if the backend
	// has a newer version.
	Set(context.Context, string, *UserConfigurableLimits, backend.Version) (backend.Version, error)
	// Delete the user-configurable overrides.
	Delete(context.Context, string, backend.Version) error
	// Shutdown the client.
	Shutdown()
}

type userConfigOverridesClient struct {
	rw backend.VersionedReaderWriter
}

var _ Client = (*userConfigOverridesClient)(nil)

func NewUserConfigOverridesClient(cfg *UserConfigurableOverridesClientConfig) (Client, error) {
	rw, err := initBackend(cfg)
	if err != nil {
		return nil, err
	}
	return &userConfigOverridesClient{rw}, nil
}

func (o *userConfigOverridesClient) Shutdown() {
	o.rw.Shutdown()
}

func initBackend(cfg *UserConfigurableOverridesClientConfig) (rw backend.VersionedReaderWriter, err error) {
	switch cfg.Backend {
	case backend.Local:
		r, w, _, err := local.New(cfg.Local)
		if err != nil {
			return nil, err
		}
		// Create overrides directory with necessary permissions
		err = os.MkdirAll(path.Join(cfg.Local.Path, OverridesKeyPath), os.ModePerm)
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

func (o *userConfigOverridesClient) List(ctx context.Context) ([]string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesClient.List")
	defer span.Finish()

	metricList.Inc()

	return o.rw.List(ctx, []string{OverridesKeyPath})
}

func (o *userConfigOverridesClient) Get(ctx context.Context, userID string) (tenantLimits *UserConfigurableLimits, version backend.Version, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesClient.Get", opentracing.Tag{Key: "tenant", Value: userID})
	defer span.Finish()

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

	d := json.NewDecoder(reader)
	err = d.Decode(&tenantLimits)
	return
}

func (o *userConfigOverridesClient) Set(ctx context.Context, userID string, limits *UserConfigurableLimits, version backend.Version) (backend.Version, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesClient.Set", opentracing.Tag{Key: "tenant", Value: userID})
	defer span.Finish()

	data, err := jsoniter.Marshal(limits)
	if err != nil {
		return "", err
	}

	return o.rw.WriteVersioned(ctx, OverridesFileName, []string{OverridesKeyPath, userID}, bytes.NewReader(data), version)
}

func (o *userConfigOverridesClient) Delete(ctx context.Context, userID string, version backend.Version) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesClient.Delete", opentracing.Tag{Key: "tenant", Value: userID})
	defer span.Finish()

	return o.rw.DeleteVersioned(ctx, OverridesFileName, []string{OverridesKeyPath, userID}, version)
}
