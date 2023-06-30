package overrides

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/yaml.v2"

	tempo_log "github.com/grafana/tempo/pkg/util/log"
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

	// PollInterval controls how often the overrides will be refreshed by polling the backend
	PollInterval time.Duration `yaml:"reload_period"`

	Backend string        `yaml:"backend"`
	Local   *local.Config `yaml:"local"`
	GCS     *gcs.Config   `yaml:"gcs"`
	S3      *s3.Config    `yaml:"s3"`
	Azure   *azure.Config `yaml:"azure"`
}

func (cfg *UserConfigOverridesConfig) RegisterFlagsAndApplyDefaults(f *flag.FlagSet) {
	cfg.PollInterval = time.Minute
}

// userConfigOverridesManager can store user-configurable overrides on a bucket.
type userConfigOverridesManager struct {
	services.Service
	// wrap Interface and only overrides select functions
	Interface

	cfg UserConfigOverridesConfig

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher

	// mtx is used to protect changes to tenantLimits and the backend
	mtx sync.RWMutex

	// tenantLimits is an in-memory cache that is refreshed at PollInterval or when a GET request
	// is received
	tenantLimits tenantLimits

	r backend.RawReader
	w backend.RawWriter

	logger log.Logger
}

// statusUserConfigOverrides used to marshal UserConfigurableLimits for tenants
type statusUserConfigOverrides struct {
	TenantLimits tenantLimits `yaml:"user_configurable_overrides" json:"user_configurable_overrides"`
}

type tenantLimits map[string]*UserConfigurableLimits

// newUserConfigOverrides wraps the given overrides with user-configurable overrides.
func newUserConfigOverrides(cfg UserConfigOverridesConfig, subOverrides Service) (*userConfigOverridesManager, error) {
	reader, writer, err := initBackend(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize backend for user-configurable overrides: %w", err)
	}

	mgr := userConfigOverridesManager{
		Interface:    subOverrides,
		cfg:          cfg,
		tenantLimits: make(tenantLimits),
		r:            reader,
		w:            writer,
		logger:       log.With(tempo_log.Logger, "component", "user-configurable overrides"),
	}

	mgr.subservices, err = services.NewManager(subOverrides)
	if err != nil {
		return nil, fmt.Errorf("failed to create subservices: %w", err)
	}
	mgr.subservicesWatcher = services.NewFailureWatcher()
	mgr.subservicesWatcher.WatchManager(mgr.subservices)

	mgr.Service = services.NewBasicService(mgr.starting, mgr.loop, mgr.stopping)

	return &mgr, nil
}

func initBackend(cfg UserConfigOverridesConfig) (reader backend.RawReader, writer backend.RawWriter, err error) {
	switch cfg.Backend {
	case "local":
		reader, writer, _, err = local.New(cfg.Local)
		if err != nil {
			return
		}
		// Create overrides directory with necessary permissions
		err = os.MkdirAll(path.Join(cfg.Local.Path, overridesKeyPath), os.ModePerm)
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

func (o *userConfigOverridesManager) starting(ctx context.Context) error {
	if err := services.StartManagerAndAwaitHealthy(ctx, o.subservices); err != nil {
		return errors.Wrap(err, "unable to start overrides subservices")
	}

	return o.reloadAllTenantLimits(ctx)
}

func (o *userConfigOverridesManager) loop(ctx context.Context) error {
	ticker := time.NewTicker(o.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			err := o.reloadAllTenantLimits(ctx)
			if err != nil {
				level.Error(o.logger).Log("msg", "failed to refresh user-configurable config", "err", err)
			}
			continue

		case err := <-o.subservicesWatcher.Chan():
			return errors.Wrap(err, "overrides subservice failed")
		}
	}
}

func (o *userConfigOverridesManager) stopping(error) error {
	return services.StopManagerAndAwaitStopped(context.Background(), o.subservices)
}

func (o *userConfigOverridesManager) reloadAllTenantLimits(ctx context.Context) error {
	level.Info(o.logger).Log("msg", "reloading all tenant limits")

	// TODO can we make this lock smaller? this will block all operations that read from the overrides
	o.mtx.Lock()
	defer o.mtx.Unlock()

	// List tenants with user-configurable overrides
	// TODO to avoid polling the entire bucket, use a tenant list and keep it in a shared cache
	tenants, err := o.r.List(ctx, []string{overridesKeyPath})
	if err != nil {
		return errors.Wrap(err, "failed to list tenants")
	}

	// Clean up cached tenants that have been removed from the backend
	for cachedTenant := range o.tenantLimits {
		if doesNotContain(tenants, cachedTenant) {
			delete(o.tenantLimits, cachedTenant)
		}
	}

	// For every tenant with user-configurable overrides, download and cache them
	for _, tenant := range tenants {
		_, err = o.getTenantLimitsUnderLock(ctx, tenant)
		if err != nil {
			// TODO should we keep trying the other tenants and return a combined error message?
			//  this implementation gives up after a single failure
			return errors.Wrap(err, "failed to load tenant limits")
		}
	}

	return nil
}

// getTenantLimits will look up the UserConfigurableLimits for a tenant, it performs a backend request
// and will update the entry in the local cache.
func (o *userConfigOverridesManager) getTenantLimits(ctx context.Context, userID string) (tenantLimits *UserConfigurableLimits, err error) {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	return o.getTenantLimitsUnderLock(ctx, userID)
}

// getTenantLimitsUnderLock does the same as getTenantLimits but requires a write lock.
func (o *userConfigOverridesManager) getTenantLimitsUnderLock(ctx context.Context, userID string) (tenantLimits *UserConfigurableLimits, err error) {
	// TODO save backend request by reading tenant limit from shared cache?

	metricFetch.WithLabelValues(userID).Inc()
	defer func() {
		if err != nil {
			metricFetchFailed.WithLabelValues(userID).Inc()
		}
	}()

	reader, _, err := o.r.Read(ctx, overridesFileName, []string{overridesKeyPath, userID}, false)
	if err == backend.ErrDoesNotExist {
		delete(o.tenantLimits, userID)
		return nil, nil
	}
	if err != nil {
		return
	}
	defer reader.Close()

	d := json.NewDecoder(reader)
	err = d.Decode(&tenantLimits)
	if err != nil {
		return
	}

	o.tenantLimits[userID] = tenantLimits
	return
}

// setTenantLimits will store the given limits
func (o *userConfigOverridesManager) setTenantLimits(ctx context.Context, userID string, limits *UserConfigurableLimits) error {
	// TODO perform validation

	// TODO do this in a constructor or something?
	limits.Version = "v1"

	data, err := jsoniter.Marshal(limits)
	if err != nil {
		return err
	}

	o.mtx.Lock()
	defer o.mtx.Unlock()

	// Store on the bucket
	err = o.w.Write(ctx, overridesFileName, []string{overridesKeyPath, userID}, bytes.NewReader(data), -1, false)
	if err != nil {
		return err
	}

	// TODO future improvement: update/invalidate cache

	o.tenantLimits[userID] = limits
	return nil
}

// deleteTenantLimits will clear all user configurable limits for the given tenant
func (o *userConfigOverridesManager) deleteTenantLimits(ctx context.Context, userID string) error {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	err := o.w.Delete(ctx, overridesFileName, []string{overridesKeyPath, userID})
	if err != nil {
		return err
	}
	delete(o.tenantLimits, userID)
	return nil
}

// getCachedTenantLimits will return the UserConfigurableLimits for a tenant from the local cache
func (o *userConfigOverridesManager) getCachedTenantLimits(userID string) (*UserConfigurableLimits, bool) {
	o.mtx.RLock()
	defer o.mtx.RUnlock()

	tenantLimits, ok := o.tenantLimits[userID]
	return tenantLimits, ok
}

func (o *userConfigOverridesManager) getAllCachedTenantLimits() map[string]*UserConfigurableLimits {
	o.mtx.RLock()
	defer o.mtx.RUnlock()

	return o.tenantLimits
}

func (o *userConfigOverridesManager) Forwarders(userID string) []string {
	tenantLimits, ok := o.getCachedTenantLimits(userID)
	if ok && tenantLimits.Forwarders != nil {
		return *tenantLimits.Forwarders
	}
	return o.Interface.Forwarders(userID)
}

func (o *userConfigOverridesManager) WriteStatusRuntimeConfig(w io.Writer, r *http.Request) error {
	// fetch runtimeConfig and Runtime per tenant Overrides
	err := o.Interface.WriteStatusRuntimeConfig(w, r)
	if err != nil {
		return err
	}

	// now write per tenant user configured overrides
	// wrap in userConfigOverrides struct to return correct yaml
	l := o.getAllCachedTenantLimits()
	ucl := statusUserConfigOverrides{TenantLimits: l}
	out, err := yaml.Marshal(ucl)
	if err != nil {
		return err
	}

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (o *userConfigOverridesManager) Describe(ch chan<- *prometheus.Desc) {
	// TODO this is hacky D:
	o.Interface.(Service).Describe(ch)
}

func (o *userConfigOverridesManager) Collect(ch chan<- prometheus.Metric) {
	// TODO this is hacky D:
	o.Interface.(Service).Collect(ch)
}
