package generator

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	// TODO make tenant-aware
	metricSpansReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_spans_received_total",
		Help:      "The total number of spans received.",
	})
)

type AppendableFactory func(userID string) storage.Appendable

type Generator struct {
	services.Service

	cfg       *Config
	overrides *overrides.Overrides

	lifecycler *ring.Lifecycler

	// TODO cache these per userID?
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
		overrides: overrides,
	}

	lc, err := ring.NewLifecycler(cfg.LifecyclerConfig, g, "generator", cfg.OverrideRingKey, true, log.Logger, prometheus.WrapRegistererWithPrefix("cortex_", reg))
	if err != nil {
		return nil, fmt.Errorf("NewLifecycler failed %w", err)
	}
	g.lifecycler = lc

	g.subservicesWatcher = services.NewFailureWatcher()
	g.subservicesWatcher.WatchService(g.lifecycler)

	// TODO add service to listen to changes in the overrides, if a tenant is added spin up a processor pipeline

	rwMetrics := newRemoteWriteMetrics(reg)
	g.appendableFactory = func(userID string) storage.Appendable {
		return newRemoteWriteAppendable(cfg, log.Logger, userID, rwMetrics)
	}

	g.Service = services.NewBasicService(g.starting, g.running, g.stopping)
	return g, nil
}

func (g *Generator) starting(ctx context.Context) error {
	// Now that user states have been created, we can start the lifecycler.
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
	<-ctx.Done()

	return nil
}

func (g *Generator) stopping(_ error) error {
	// TODO remove tokens from the ring?

	err := services.StopAndAwaitTerminated(context.Background(), g.lifecycler)
	if err != nil {
		level.Warn(log.Logger).Log("msg", "failed to stop generator lifecycler", "err", err)
	}

	return nil
}

func (g *Generator) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) (*tempopb.PushResponse, error) {
	metricSpansReceivedTotal.Inc()

	// YOLO: write a sample for every request we receive
	orgID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	appendable := g.appendableFactory(orgID)
	appender := appendable.Appender(ctx)

	_, err = appender.Append(
		0,
		[]labels.Label{
			{
				Name:  "__name__",
				Value: "hello_from_tempo",
			},
		},
		time.Now().UnixMilli(),
		float64(rand.Float64()),
	)
	if err != nil {
		return nil, err
	}

	err = appender.Commit()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Flush is called by the lifecycler on shutdown.
func (g *Generator) Flush() {
}

func (g *Generator) TransferOut(ctx context.Context) error {
	return ring.ErrTransferDisabled
}

func (g *Generator) CheckReady(ctx context.Context) error {
	if err := g.lifecycler.CheckReady(ctx); err != nil {
		return fmt.Errorf("generator check ready failed %w", err)
	}

	return nil
}
