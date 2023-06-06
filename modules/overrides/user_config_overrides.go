package overrides

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
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
	overridesKeyPath  = "overrides"
	overridesFileName = "overrides.json"
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

type UserConfigOverridesConfig struct {
	Enabled bool `yaml:"enabled"`

	ReloadPeriod time.Duration `yaml:"reload_period"`

	Backend string        `yaml:"backend"`
	Local   *local.Config `yaml:"local"`
	GCS     *gcs.Config   `yaml:"gcs"`
	S3      *s3.Config    `yaml:"s3"`
	Azure   *azure.Config `yaml:"azure"`
}

func (cfg *UserConfigOverridesConfig) RegisterFlagsAndApplyDefaults(f *flag.FlagSet) {
	cfg.ReloadPeriod = time.Minute

	// TODO should we configure a default backend?
}

// UserConfigOverridesManager can store user-configurable overrides on a bucket.
type UserConfigOverridesManager struct {
	services.Service
	// wrap Interface and only overrides select functions
	Interface

	cfg UserConfigOverridesConfig

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher

	// mtx is used to protect changes to tenantLimits and the backend
	mtx sync.RWMutex

	tenantLimits map[string]*UserConfigurableLimits

	r backend.RawReader
	w backend.RawWriter
}

// NewUserConfigOverrides wraps the given overrides with user-configurable overrides.
func NewUserConfigOverrides(cfg UserConfigOverridesConfig, subOverrides Service) (*UserConfigOverridesManager, error) {
	reader, writer, err := initBackend(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user configurable overrides: %w", err)
	}

	mgr := UserConfigOverridesManager{
		Interface:    subOverrides,
		cfg:          cfg,
		tenantLimits: make(map[string]*UserConfigurableLimits),
		r:            reader,
		w:            writer,
	}

	service := services.NewBasicService(mgr.starting, mgr.loop, mgr.stopping)

	mgr.subservices, err = services.NewManager(subOverrides, service)
	if err != nil {
		return nil, fmt.Errorf("failed to create subservices: %w", err)
	}
	mgr.subservicesWatcher = services.NewFailureWatcher()
	mgr.subservicesWatcher.WatchManager(mgr.subservices)

	return &mgr, nil
}

func initBackend(cfg UserConfigOverridesConfig) (reader backend.RawReader, writer backend.RawWriter, err error) {
	switch cfg.Backend {
	case "local":
		reader, writer, _, err = local.New(cfg.Local)
	case "gcs":
		reader, writer, _, err = gcs.New(cfg.GCS)
	case "s3":
		reader, writer, _, err = s3.New(cfg.S3)
	case "azure":
		reader, writer, _, err = azure.New(cfg.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.Backend)
	}
	// TODO wrap reader and writer in cache
	return
}

func (o *UserConfigOverridesManager) starting(ctx context.Context) error {
	if err := services.StartManagerAndAwaitHealthy(ctx, o.subservices); err != nil {
		return errors.Wrap(err, "unable to start overrides subservices")
	}

	return o.refreshAllTenantLimits(ctx)
}

func (o *UserConfigOverridesManager) loop(ctx context.Context) error {
	ticker := time.NewTicker(o.cfg.ReloadPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			err := o.refreshAllTenantLimits(ctx)
			if err != nil {
				level.Error(log.Logger).Log("msg", "failed to refresh user-configurable config", "err", err)
			}
			continue

		case err := <-o.subservicesWatcher.Chan():
			return errors.Wrap(err, "overrides subservice failed")
		}
	}
}

func (o *UserConfigOverridesManager) stopping(error) error {
	return services.StopManagerAndAwaitStopped(context.Background(), o.subservices)
}

func (o *UserConfigOverridesManager) refreshAllTenantLimits(ctx context.Context) error {
	// List tenants with user-configurable overrides
	// TODO to avoid polling the entire bucket, use a tenant list and keep it in a shared cache
	tenants, err := o.r.List(ctx, []string{overridesKeyPath})
	if err != nil {
		return errors.Wrap(err, "failed to list tenants")
	}

	// For every tenant with user-configurable overrides, download and cache them
	for _, tenant := range tenants {
		err = o.refreshTenantLimits(ctx, tenant)
		if err != nil {
			// TODO should we keep trying the other tenants and return a combined error message?
			//  this implementation gives up after a single failure
			return errors.Wrap(err, "failed to load tenant limits")
		}
	}

	return nil
}

// refreshTenantLimits reads the limits for a tenant fetching it from the backend and caching it in memory.
func (o *UserConfigOverridesManager) refreshTenantLimits(ctx context.Context, userID string) error {
	tenantLimits, err := o.downloadTenantLimits(ctx, userID)

	metricFetch.WithLabelValues(userID).Inc()
	if err != nil {
		metricFetchFailed.WithLabelValues(userID).Inc()
		return err
	}

	o.mtx.Lock()
	defer o.mtx.Unlock()

	o.tenantLimits[userID] = tenantLimits
	return nil
}

func (o *UserConfigOverridesManager) downloadTenantLimits(ctx context.Context, userID string) (*UserConfigurableLimits, error) {
	// TODO ensure tenant limit is read from shared cache

	reader, _, err := o.r.Read(ctx, overridesFileName, []string{overridesKeyPath, userID}, false)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var tenantLimits UserConfigurableLimits
	d := json.NewDecoder(reader)
	err = d.Decode(&tenantLimits)

	return &tenantLimits, err
}

// SetLimits will store the given limits
func (o *UserConfigOverridesManager) SetLimits(ctx context.Context, userID string, limits *UserConfigurableLimits) error {
	// TODO perform validation

	// TODO do this in a constructor or something?
	limits.Version = "v1"

	o.mtx.Lock()
	defer o.mtx.Unlock()

	// Store on the bucket
	data, err := jsoniter.Marshal(limits)
	if err != nil {
		return err
	}

	err = o.w.Write(ctx, overridesFileName, []string{overridesKeyPath, userID}, bytes.NewReader(data), -1, false)
	if err != nil {
		return err
	}

	// TODO future improvement: update/invalidate cache

	o.tenantLimits[userID] = limits
	return nil
}

// DeleteLimits will clear all user configurable limits for the given tenant
func (o *UserConfigOverridesManager) DeleteLimits(ctx context.Context, userID string) error {
	// TODO properly delete from the bucket, hacky workaround to clear limits
	//   we should implement and use a Delete function
	// err = o.w.Delete(ctx, overridesFileName, []string{userID})

	emptyLimits := newUserConfigurableLimits()

	data, err := jsoniter.Marshal(emptyLimits)
	if err != nil {
		return err
	}

	o.mtx.Lock()
	defer o.mtx.Unlock()

	err = o.w.Write(ctx, overridesFileName, []string{userID}, bytes.NewReader(data), -1, false)
	if err != nil {
		return err
	}

	delete(o.tenantLimits, userID)
	return nil
}

func (o *UserConfigOverridesManager) getTenantLimits(userID string) (*UserConfigurableLimits, bool) {
	o.mtx.RLock()
	defer o.mtx.RUnlock()

	tenantLimits, ok := o.tenantLimits[userID]
	return tenantLimits, ok
}

func (o *UserConfigOverridesManager) Forwarders(userID string) []string {
	tenantLimits, ok := o.getTenantLimits(userID)
	if ok && tenantLimits.Forwarders != nil {
		return *tenantLimits.Forwarders
	}
	return o.Interface.Forwarders(userID)
}

func (o *UserConfigOverridesManager) WriteStatusRuntimeConfig(w io.Writer, r *http.Request) error {
	// TODO this is hacky D:
	//   should we make WriteStatusRuntimeConfig part of Interface instead of Service?
	return o.Interface.(Service).WriteStatusRuntimeConfig(w, r)
}

func (o *UserConfigOverridesManager) Describe(ch chan<- *prometheus.Desc) {
	// TODO this is hacky D:
	o.Interface.(Service).Describe(ch)
}

func (o *UserConfigOverridesManager) Collect(ch chan<- prometheus.Metric) {
	// TODO this is hacky D:
	o.Interface.(Service).Collect(ch)
}
