package overrides

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/weaveworks/common/tracing"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"

	api "github.com/grafana/tempo/modules/overrides/user_configurable_api"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
)

type UserConfigOverridesConfig struct {
	Enabled bool `yaml:"enabled"`

	// PollInterval controls how often the overrides will be refreshed by polling the backend
	PollInterval time.Duration `yaml:"poll_interval"`

	ClientConfig api.UserConfigOverridesClientConfig `yaml:"client"`
}

func (cfg *UserConfigOverridesConfig) RegisterFlagsAndApplyDefaults(*flag.FlagSet) {
	cfg.PollInterval = time.Minute
}

// userConfigOverridesManager can store user-configurable overrides on a bucket.
type userConfigOverridesManager struct {
	services.Service
	// wrap Interface and only overrides select functions
	Interface

	cfg *UserConfigOverridesConfig

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher

	mtx          sync.RWMutex
	tenantLimits tenantLimits

	client api.Client

	logger log.Logger
}

// statusUserConfigOverrides used to marshal UserConfigurableLimits for tenants
type statusUserConfigOverrides struct {
	TenantLimits tenantLimits `yaml:"user_configurable_overrides" json:"user_configurable_overrides"`
}

type tenantLimits map[string]*api.UserConfigurableLimits

// newUserConfigOverrides wraps the given overrides with user-configurable overrides.
func newUserConfigOverrides(cfg *UserConfigOverridesConfig, subOverrides Service) (*userConfigOverridesManager, error) {
	client, err := api.NewUserConfigOverridesClient(&cfg.ClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize backend client for user-configurable overrides: %w", err)
	}

	mgr := userConfigOverridesManager{
		Interface:    subOverrides,
		cfg:          cfg,
		tenantLimits: make(tenantLimits),
		client:       client,
		logger:       log.With(tempo_log.Logger, "component", "user-configurable overrides"),
	}

	mgr.subservices, err = services.NewManager(subOverrides)
	if err != nil {
		return nil, fmt.Errorf("failed to create subservices: %w", err)
	}
	mgr.subservicesWatcher = services.NewFailureWatcher()
	mgr.subservicesWatcher.WatchManager(mgr.subservices)

	mgr.Service = services.NewBasicService(mgr.starting, mgr.running, mgr.stopping)

	return &mgr, nil
}

func (o *userConfigOverridesManager) starting(ctx context.Context) error {
	if err := services.StartManagerAndAwaitHealthy(ctx, o.subservices); err != nil {
		return errors.Wrap(err, "unable to start overrides subservices")
	}

	return o.reloadAllTenantLimits(ctx)
}

func (o *userConfigOverridesManager) running(ctx context.Context) error {
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesManager.reloadAllTenantLimits")
	defer span.Finish()

	traceID, _ := tracing.ExtractTraceID(ctx)
	level.Info(o.logger).Log("msg", "reloading all tenant limits", "traceID", traceID)

	// List tenants with user-configurable overrides
	tenants, err := o.client.List(ctx)
	if err != nil {
		return err
	}

	// Clean up cached tenants that have been removed from the backend
	for cachedTenant := range o.tenantLimits {
		if !slices.Contains(tenants, cachedTenant) {
			o.deleteTenantLimit(cachedTenant)
		}
	}

	// For every tenant with user-configurable overrides, download and cache them
	for _, tenant := range tenants {
		limits, err := o.client.Get(ctx, tenant)
		if err != nil {
			return errors.Wrapf(err, "failed to load tenant limits for tenant %v", tenant)
		}
		o.setTenantLimit(tenant, limits)
	}

	return nil
}

func (o *userConfigOverridesManager) getTenantLimits(userID string) (*api.UserConfigurableLimits, bool) {
	o.mtx.RLock()
	defer o.mtx.RUnlock()

	tenantLimits, ok := o.tenantLimits[userID]
	return tenantLimits, ok
}

func (o *userConfigOverridesManager) getAllTenantLimits() tenantLimits {
	o.mtx.RLock()
	defer o.mtx.RUnlock()

	return o.tenantLimits
}

func (o *userConfigOverridesManager) setTenantLimit(userID string, limits *api.UserConfigurableLimits) {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	o.tenantLimits[userID] = limits
}

func (o *userConfigOverridesManager) deleteTenantLimit(userID string) {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	delete(o.tenantLimits, userID)
}

func (o *userConfigOverridesManager) Forwarders(userID string) []string {
	tenantLimits, ok := o.getTenantLimits(userID)
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
	l := o.getAllTenantLimits()
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
	// TODO for now just pass along to the underlying overrides, in the future we should export
	//  the user-config overrides as well
	o.Interface.Describe(ch)
}

func (o *userConfigOverridesManager) Collect(ch chan<- prometheus.Metric) {
	// TODO for now just pass along to the underlying overrides, in the future we should export
	//  the user-config overrides as well
	o.Interface.Collect(ch)
}
