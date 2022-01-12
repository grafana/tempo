package generator

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/storage"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

const userMetricsScrapeEndpoint = "/api/trace-metrics"

type AppendableFactory func(userID string) storage.Appendable

type Generator struct {
	services.Service

	cfg *Config

	instancesMtx sync.RWMutex
	instances    map[string]*instance

	lifecycler *ring.Lifecycler
	overrides  *overrides.Overrides

	// TODO figure out how to gather metrics from a prometheus.Registerer and
	//  push them into a storage.Appender
	//
	// Issue: the code to create metrics uses a Registerer, this is a concept from
	// prometheus/client_golang, i.e. the instrumentation library.
	// The code to work with storage uses storage.Appendable / storage.Appender, which
	// is from prometheus/prometheus, i.e. the server implementation.
	//
	// Unfortunately these two worlds have different data models, you cannot combine
	// a Registerer with an Appender. In a normal set up they never interact with each
	// other directly but through HTTP scrapes.
	//
	// Current implementation:
	// - create a single Registry dedicated for user metrics, this is used by processors
	// - expose this Registry on an HTTP endpoint
	// - an external Prometheus instance scrapes this endpoint
	//
	// Ideal scenario:
	// - appendableFactory allows you to create a storage.Appender per tenant
	// - for every instance/tenant:
	//	 - create a dedicated Registerer to be used by the processors
	//   - each tenant has a 'Collector' which scrapes the Registerer and pushes data into storage.Appender

	// TODO since we only set up a single scrape endpoint, we have to use a global Registerer
	//  until we move this Registerer into instance we cannot support multi-tenancy
	userMetricsRegisterer     prometheus.Registerer
	UserMetricsScrapeEndpoint string
	UserMetricsHandler        http.Handler

	appendableFactory AppendableFactory

	subservicesWatcher *services.FailureWatcher
}

// New makes a new Generator.
func New(cfg Config, overrides *overrides.Overrides, reg prometheus.Registerer) (*Generator, error) {
	if cfg.RemoteWrite.Enabled && cfg.RemoteWrite.Client.URL == nil {
		return nil, errors.New("remote-write enabled but client URL is not configured")
	}

	g := &Generator{
		cfg:       &cfg,
		instances: map[string]*instance{},
		overrides: overrides,
	}

	lc, err := ring.NewLifecycler(cfg.LifecyclerConfig, g, "metrics-generator", cfg.OverrideRingKey, true, log.Logger, prometheus.WrapRegistererWithPrefix("cortex_", reg))
	if err != nil {
		return nil, fmt.Errorf("NewLifecycler failed %w", err)
	}
	g.lifecycler = lc

	g.subservicesWatcher = services.NewFailureWatcher()
	g.subservicesWatcher.WatchService(g.lifecycler)

	userMetricsRegistry := prometheus.NewRegistry()
	g.userMetricsRegisterer = userMetricsRegistry
	g.UserMetricsScrapeEndpoint = userMetricsScrapeEndpoint
	g.UserMetricsHandler = promhttp.HandlerFor(userMetricsRegistry, promhttp.HandlerOpts{})

	remoteWriteMetrics := newRemoteWriteMetrics(reg)
	g.appendableFactory = func(userID string) storage.Appendable {
		return newRemoteWriteAppendable(cfg, log.Logger, userID, remoteWriteMetrics)
	}

	g.Service = services.NewBasicService(g.starting, g.running, g.stopping)
	return g, nil
}

func (g *Generator) starting(ctx context.Context) error {
	// Important: we want to keep lifecycler running until we ask it to stop, so we need to give it independent context
	if err := g.lifecycler.StartAsync(context.Background()); err != nil {
		return fmt.Errorf("failed to start lifecycler %w", err)
	}
	if err := g.lifecycler.AwaitRunning(ctx); err != nil {
		return fmt.Errorf("failed to start lifecycle %w", err)
	}

	return nil
}

func (g *Generator) running(ctx context.Context) error {
	// TODO make configurable
	//  Should we make the collect interval configurable per tenant? Tenants might want to choose between 1 and 4 DPM
	collectMetricsInterval := 15 * time.Second

	collectMetricsTicker := time.NewTicker(collectMetricsInterval)
	defer collectMetricsTicker.Stop()

	for {
		select {
		case <-collectMetricsTicker.C:
			g.collectMetrics()

		case <-ctx.Done():
			return nil

		case err := <-g.subservicesWatcher.Chan():
			return fmt.Errorf("metrics-generator subservice failed %w", err)
		}
	}
}

func (g *Generator) stopping(_ error) error {
	// TODO remove tokens from the ring?

	err := services.StopAndAwaitTerminated(context.Background(), g.lifecycler)
	if err != nil {
		level.Warn(log.Logger).Log("msg", "failed to stop generator lifecycler", "err", err)
	}

	for id, instance := range g.instances {
		err := instance.Shutdown(context.Background())
		if err != nil {
			level.Warn(log.Logger).Log("msg", "failed to shutdown instance", "instanceID", id, "err", err)
		}
	}

	return nil
}

func (g *Generator) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) (*tempopb.PushResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "generator.PushSpans")
	defer span.Finish()

	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	span.SetTag("instanceID", instanceID)

	instance, err := g.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	err = instance.PushSpans(ctx, req)
	if err != nil {
		return nil, err
	}

	return &tempopb.PushResponse{}, nil
}

func (g *Generator) getOrCreateInstance(instanceID string) (*instance, error) {
	inst, ok := g.getInstanceByID(instanceID)
	if ok {
		return inst, nil
	}

	// TODO: Take a RLock before Lock? 🔐
	g.instancesMtx.Lock()
	defer g.instancesMtx.Unlock()
	inst, ok = g.instances[instanceID]
	if !ok {
		var err error
		inst, err = newInstance(instanceID, g.overrides, g.userMetricsRegisterer, g.appendableFactory(instanceID))
		if err != nil {
			return nil, err
		}
		g.instances[instanceID] = inst
	}
	return inst, nil
}

func (g *Generator) getInstanceByID(id string) (*instance, bool) {
	g.instancesMtx.RLock()
	defer g.instancesMtx.RUnlock()

	inst, ok := g.instances[id]
	return inst, ok
}

func (g *Generator) collectMetrics() {
	span := opentracing.StartSpan("generator.collectMetrics")
	defer span.Finish()

	ctx := opentracing.ContextWithSpan(context.Background(), span)

	for _, instance := range g.instances {
		err := instance.collectAndPushMetrics(ctx)
		if err != nil {
			level.Error(log.Logger).Log("msg", "collecting and pushing metrics failed", "tenant", instance.instanceID, "err", err)
		}
	}
}

// Flush is called by the lifecycler on shutdown.
func (g *Generator) Flush() {
}

func (g *Generator) TransferOut(ctx context.Context) error {
	return ring.ErrTransferDisabled
}

func (g *Generator) CheckReady(ctx context.Context) error {
	if err := g.lifecycler.CheckReady(ctx); err != nil {
		return fmt.Errorf("metrics-generator check ready failed %w", err)
	}

	return nil
}
