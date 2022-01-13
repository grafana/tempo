package generator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/storage"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

type AppendableFactory func(userID string) storage.Appendable

type Generator struct {
	services.Service

	cfg *Config

	instancesMtx sync.RWMutex
	instances    map[string]*instance

	lifecycler *ring.Lifecycler
	overrides  *overrides.Overrides

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

	// TODO should we use the BasicLifecycler so we can auto-forget unhealthy instances?
	lc, err := ring.NewLifecycler(cfg.LifecyclerConfig, nil, "metrics-generator", cfg.OverrideRingKey, false, log.Logger, prometheus.WrapRegistererWithPrefix("cortex_", reg))
	if err != nil {
		return nil, fmt.Errorf("NewLifecycler failed %w", err)
	}
	lc.SetUnregisterOnShutdown(true)
	g.lifecycler = lc

	g.subservicesWatcher = services.NewFailureWatcher()
	g.subservicesWatcher.WatchService(g.lifecycler)

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

	// TODO should we make this a separate service? This could simplify stop logic a little bit
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
	// TODO at shutdown we should:
	//   - refuse new writes
	//   - collect and push metrics a final time
	//  Shutting down the instance/processors isn't necessary I think

	err := services.StopAndAwaitTerminated(context.Background(), g.lifecycler)
	if err != nil {
		level.Warn(log.Logger).Log("msg", "failed to stop generator lifecycler", "err", err)
	}

	for id, instance := range g.instances {
		err := instance.shutdown(context.Background())
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

	g.instancesMtx.Lock()
	defer g.instancesMtx.Unlock()
	inst, ok = g.instances[instanceID]
	if !ok {
		var err error
		inst, err = newInstance(g.cfg, instanceID, g.overrides, g.appendableFactory(instanceID))
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

func (g *Generator) CheckReady(ctx context.Context) error {
	if err := g.lifecycler.CheckReady(ctx); err != nil {
		return fmt.Errorf("metrics-generator check ready failed %w", err)
	}

	return nil
}
