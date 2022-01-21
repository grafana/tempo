package generator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/storage"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	// ringAutoForgetUnhealthyPeriods is how many consecutive timeout periods an unhealthy instance
	// in the ring will be automatically removed.
	ringAutoForgetUnhealthyPeriods = 2

	// We use a safe default instead of exposing to config option to the user
	// in order to simplify the config.
	ringNumTokens = 256
)

type AppendableFactory func(userID string) storage.Appendable

type Generator struct {
	services.Service

	cfg       *Config
	overrides *overrides.Overrides

	ringLifecycler *ring.BasicLifecycler
	ring           *ring.Ring

	instancesMtx sync.RWMutex
	instances    map[string]*instance

	appendableFactory AppendableFactory

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// New makes a new Generator.
func New(cfg Config, overrides *overrides.Overrides, reg prometheus.Registerer) (*Generator, error) {
	if cfg.RemoteWrite.Enabled && cfg.RemoteWrite.Client.URL == nil {
		return nil, errors.New("remote-write enabled but client URL is not configured")
	}

	g := &Generator{
		cfg:       &cfg,
		overrides: overrides,

		instances: map[string]*instance{},
	}

	// Lifecycler and ring
	ringStore, err := kv.NewClient(
		cfg.Ring.KVStore,
		ring.GetCodec(),
		kv.RegistererWithKVName(prometheus.WrapRegistererWithPrefix("cortex_", reg), "metrics-generator"),
		log.Logger,
	)
	if err != nil {
		return nil, fmt.Errorf("create KV store client: %w", err)
	}

	lifecyclerCfg, err := cfg.Ring.toLifecyclerConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid ring lifecycler config: %w", err)
	}

	// Define lifecycler delegates in reverse order (last to be called defined first because they're
	// chained via "next delegate").
	delegate := ring.BasicLifecyclerDelegate(g)
	delegate = ring.NewLeaveOnStoppingDelegate(delegate, log.Logger)
	delegate = ring.NewAutoForgetDelegate(ringAutoForgetUnhealthyPeriods*cfg.Ring.HeartbeatTimeout, delegate, log.Logger)

	g.ringLifecycler, err = ring.NewBasicLifecycler(lifecyclerCfg, ringNameForServer, RingKey, ringStore, delegate, log.Logger, prometheus.WrapRegistererWithPrefix("cortex_", reg))
	if err != nil {
		return nil, fmt.Errorf("create ring lifecycler: %w", err)
	}

	ringCfg := cfg.Ring.ToRingConfig()
	g.ring, err = ring.NewWithStoreClientAndStrategy(ringCfg, ringNameForServer, RingKey, ringStore, ring.NewIgnoreUnhealthyInstancesReplicationStrategy(), prometheus.WrapRegistererWithPrefix("cortex_", reg), log.Logger)
	if err != nil {
		return nil, fmt.Errorf("create ring client: %w", err)
	}

	// Remote write
	remoteWriteMetrics := newRemoteWriteMetrics(reg)
	g.appendableFactory = func(userID string) storage.Appendable {
		return newRemoteWriteAppendable(cfg, log.Logger, userID, remoteWriteMetrics)
	}

	g.Service = services.NewBasicService(g.starting, g.running, g.stopping)
	return g, nil
}

func (g *Generator) starting(ctx context.Context) (err error) {
	// In case this function will return error we want to unregister the instance
	// from the ring. We do it ensuring dependencies are gracefully stopped if they
	// were already started.
	defer func() {
		if err == nil || g.subservices == nil {
			return
		}

		if stopErr := services.StopManagerAndAwaitStopped(context.Background(), g.subservices); stopErr != nil {
			level.Error(log.Logger).Log("msg", "failed to gracefully stop metrics-generator dependencies", "err", stopErr)
		}
	}()

	g.subservices, err = services.NewManager(g.ringLifecycler, g.ring)
	if err != nil {
		return fmt.Errorf("unable to start metrics-generator dependencies: %w", err)
	}
	g.subservicesWatcher = services.NewFailureWatcher()
	g.subservicesWatcher.WatchManager(g.subservices)

	err = services.StartManagerAndAwaitHealthy(ctx, g.subservices)
	if err != nil {
		return fmt.Errorf("unable to start mertics-generator dependencies: %w", err)
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

	if g.subservices != nil {
		err := services.StopManagerAndAwaitStopped(context.Background(), g.subservices)
		if err != nil {
			level.Error(log.Logger).Log("msg", "failed to stop metrics-generator dependencies", "err", err)
		}
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

	err = instance.pushSpans(ctx, req)
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
	// TODO do we need a check ready?
	//if err := g.ringLifecycler.CheckReady(ctx); err != nil {
	//	return fmt.Errorf("metrics-generator check ready failed %w", err)
	//}

	return nil
}

// OnRingInstanceRegister implements ring.BasicLifecyclerDelegate
func (g *Generator) OnRingInstanceRegister(lifecycler *ring.BasicLifecycler, ringDesc ring.Desc, instanceExists bool, instanceID string, instanceDesc ring.InstanceDesc) (ring.InstanceState, ring.Tokens) {
	// When we initialize the metrics-generator instance in the ring we want to start from
	// a clean situation, so whatever is the state we set it ACTIVE, while we keep existing
	// tokens (if any) or the ones loaded from file.
	var tokens []uint32
	if instanceExists {
		tokens = instanceDesc.GetTokens()
	}

	takenTokens := ringDesc.GetTokens()
	newTokens := ring.GenerateTokens(ringNumTokens-len(tokens), takenTokens)

	// Tokens sorting will be enforced by the parent caller.
	tokens = append(tokens, newTokens...)

	return ring.ACTIVE, tokens
}

// OnRingInstanceTokens implements ring.BasicLifecyclerDelegate
func (g *Generator) OnRingInstanceTokens(lifecycler *ring.BasicLifecycler, tokens ring.Tokens) {
}

// OnRingInstanceStopping implements ring.BasicLifecyclerDelegate
func (g *Generator) OnRingInstanceStopping(lifecycler *ring.BasicLifecycler) {
}

// OnRingInstanceHeartbeat implements ring.BasicLifecyclerDelegate
func (g *Generator) OnRingInstanceHeartbeat(lifecycler *ring.BasicLifecycler, ringDesc *ring.Desc, instanceDesc *ring.InstanceDesc) {
}
