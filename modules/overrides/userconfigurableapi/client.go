package userconfigurableapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

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

	Local *local.Config `yaml:"local"`
	GCS   *gcs.Config   `yaml:"gcs"`
	S3    *s3.Config    `yaml:"s3"`
	Azure *azure.Config `yaml:"azure"`
}

type Client interface {
	List(ctx context.Context) ([]string, error)
	Get(context.Context, string) (*UserConfigurableLimits, error)
	Set(context.Context, string, *UserConfigurableLimits) error
	Delete(context.Context, string) error
}

type userConfigOverridesClient struct {
	r backend.RawReader
	w backend.RawWriter
}

var _ Client = (*userConfigOverridesClient)(nil)

func NewUserConfigOverridesClient(cfg *UserConfigurableOverridesClientConfig) (Client, error) {
	r, w, err := initBackend(cfg)
	if err != nil {
		return nil, err
	}
	return &userConfigOverridesClient{r, w}, nil
}

func initBackend(cfg *UserConfigurableOverridesClientConfig) (reader backend.RawReader, writer backend.RawWriter, err error) {
	switch cfg.Backend {
	case "local":
		reader, writer, _, err = local.New(cfg.Local)
		if err != nil {
			return
		}
		// Create overrides directory with necessary permissions
		err = os.MkdirAll(path.Join(cfg.Local.Path, OverridesKeyPath), os.ModePerm)
	case "gcs":
		reader, writer, _, err = gcs.New(cfg.GCS)
	case "s3":
		reader, writer, _, err = s3.New(cfg.S3)
	case "azure":
		reader, writer, _, err = azure.New(cfg.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.Backend)
	}
	return
}

func (o *userConfigOverridesClient) List(ctx context.Context) ([]string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesClient.List")
	defer span.Finish()

	tenantDirectories, err := o.r.List(ctx, []string{OverridesKeyPath})
	if err != nil {
		return nil, err
	}

	var tenants []string
	for _, tenant := range tenantDirectories {
		_, _, err := o.r.Read(ctx, OverridesFileName, []string{OverridesKeyPath, tenant}, false)
		if err != nil {
			continue
		}
		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

// Get downloads the tenant limits from the backend. Returns nil, nil if no limits are set.
func (o *userConfigOverridesClient) Get(ctx context.Context, userID string) (tenantLimits *UserConfigurableLimits, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesClient.Get")
	defer span.Finish()
	span.SetTag("tenant", userID)

	metricFetch.WithLabelValues(userID).Inc()
	defer func() {
		if err != nil {
			metricFetchFailed.WithLabelValues(userID).Inc()
		}
	}()

	reader, _, err := o.r.Read(ctx, OverridesFileName, []string{OverridesKeyPath, userID}, false)
	if err == backend.ErrDoesNotExist {
		return nil, nil
	}
	if err != nil {
		return
	}
	defer reader.Close()

	d := json.NewDecoder(reader)
	err = d.Decode(&tenantLimits)
	return
}

// Set stores the user-configurable limits on the backend.
func (o *userConfigOverridesClient) Set(ctx context.Context, userID string, limits *UserConfigurableLimits) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesClient.Set")
	defer span.Finish()
	span.SetTag("tenant", userID)

	data, err := jsoniter.Marshal(limits)
	if err != nil {
		return err
	}

	// Store on the bucket
	return o.w.Write(ctx, OverridesFileName, []string{OverridesKeyPath, userID}, bytes.NewReader(data), -1, false)
}

// Delete will clear all user-configurable limits for the given tenant.
func (o *userConfigOverridesClient) Delete(ctx context.Context, userID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesClient.Delete")
	defer span.Finish()
	span.SetTag("tenant", userID)

	return o.w.Delete(ctx, OverridesFileName, []string{OverridesKeyPath, userID})
}
