package generator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/atomic"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/validation"
)

const (
	// NoGenerateMetricsContextKey is used in request contexts/headers to signal to
	// the metrics generator that it should not generate metrics for the spans
	// contained in the requests. This is intended to be used by clients that send
	// requests for which span-derived metrics have already been generated elsewhere.
	NoGenerateMetricsContextKey = "no-generate-metrics"

	// failureBackoff is the duration to wait before retrying failed tenant instance creation.
	failureBackoff = 1 * time.Minute
)

var tracer = otel.Tracer("modules/generator")

var (
	ErrUnconfigured            = errors.New("no metrics_generator.storage.path configured, metrics generator will be disabled")
	ErrReadOnly                = errors.New("metrics-generator is shutting down")
	errInstanceCreationBackoff = errors.New("instance creation in backoff")
)

type Generator struct {
	services.Service

	cfg       *Config
	overrides metricsGeneratorOverrides

	instancesMtx    sync.RWMutex
	instances       map[string]*instance
	failedInstances map[string]time.Time // instance -> when creation last failed

	// When set to true, the generator will refuse incoming pushes
	// and will flush any remaining metrics.
	readOnly atomic.Bool

	reg    prometheus.Registerer
	logger log.Logger

	kafkaCh            chan *kgo.Record
	kafkaWG            sync.WaitGroup
	kafkaStop          func()
	kafkaClient        *ingest.Client
	kafkaAdm           *kadm.Client
	partitionClient    *ingest.PartitionOffsetClient
	partitionRing      ring.PartitionRingReader
	partitionMtx       sync.RWMutex
	assignedPartitions []int32

	// leaveGroupFn is called by stopKafka when LeaveConsumerGroupOnShutdown is
	// true. It is initialized in New to call ingest.LeaveConsumerGroupByInstanceID
	// and can be overridden in tests.
	leaveGroupFn func(ctx context.Context) error
}

// New makes a new Generator.
func New(cfg *Config, overrides metricsGeneratorOverrides, reg prometheus.Registerer, partitionRing ring.PartitionRingReader, logger log.Logger) (*Generator, error) {
	if cfg.Storage.Path == "" {
		return nil, ErrUnconfigured
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	err := os.MkdirAll(cfg.Storage.Path, 0o700)
	if err != nil {
		return nil, fmt.Errorf("failed to mkdir on %s: %w", cfg.Storage.Path, err)
	}

	g := &Generator{
		cfg:       cfg,
		overrides: overrides,

		instances:       map[string]*instance{},
		failedInstances: map[string]time.Time{},

		partitionRing: partitionRing,
		reg:           reg,
		logger:        logger,
	}
	g.leaveGroupFn = func(ctx context.Context) error {
		return ingest.LeaveConsumerGroupByInstanceID(ctx, g.kafkaClient.Client,
			g.cfg.Ingest.Kafka.ConsumerGroup, g.cfg.InstanceID, g.logger)
	}

	g.Service = services.NewBasicService(g.starting, g.running, g.stopping)
	return g, nil
}

func (g *Generator) starting(ctx context.Context) error {
	if g.cfg.ConsumeFromKafka {
		kafkaClient, err := ingest.NewGroupReaderClient(
			g.cfg.Ingest.Kafka,
			g.partitionRing,
			ingest.NewReaderClientMetrics("generator", prometheus.DefaultRegisterer),
			g.logger,
			kgo.InstanceID(g.cfg.InstanceID),
			kgo.OnPartitionsAssigned(func(_ context.Context, _ *kgo.Client, m map[string][]int32) {
				g.handlePartitionsAssigned(m)
			}),
			kgo.OnPartitionsRevoked(func(_ context.Context, _ *kgo.Client, m map[string][]int32) {
				g.handlePartitionsRevoked(m)
			}),
		)
		if err != nil {
			return fmt.Errorf("failed to create kafka reader client: %w", err)
		}

		g.kafkaClient = kafkaClient
		if err := ingest.WaitForKafkaBroker(ctx, g.kafkaClient.Client, g.logger); err != nil {
			return fmt.Errorf("failed to start metrics generator: %w", err)
		}

		g.kafkaAdm = kadm.NewClient(g.kafkaClient.Client)
		g.partitionClient = ingest.NewPartitionOffsetClient(g.kafkaClient.Client, g.cfg.Ingest.Kafka.Topic)
	}

	return nil
}

func (g *Generator) running(ctx context.Context) error {
	if g.cfg.ConsumeFromKafka {
		g.startKafka()
	}

	<-ctx.Done()
	return nil
}

func (g *Generator) stopping(_ error) error {
	g.stopIncomingRequests()

	// Stop reading from queue and wait for outstanding data to be processed and committed.
	if g.cfg.ConsumeFromKafka {
		g.stopKafka()
	}

	var wg sync.WaitGroup
	wg.Add(len(g.instances))

	for _, inst := range g.instances {
		go func(inst *instance) {
			inst.shutdown()
			wg.Done()
		}(inst)
	}

	wg.Wait()

	return nil
}

// stopIncomingRequests marks the generator as read-only, refusing push requests
func (g *Generator) stopIncomingRequests() {
	g.readOnly.Store(true)
}

func (g *Generator) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) (*tempopb.PushResponse, error) {
	if g.readOnly.Load() {
		return nil, ErrReadOnly
	}

	ctx, span := tracer.Start(ctx, "generator.PushSpans")
	defer span.End()

	instanceID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.String("instanceID", instanceID))

	instance, err := g.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	instance.pushSpans(ctx, req)

	return &tempopb.PushResponse{}, nil
}

func (g *Generator) getOrCreateInstance(instanceID string) (*instance, error) {
	// Fast path: check with read lock first
	inst, ok := g.getInstanceByID(instanceID)
	if ok {
		return inst, nil
	}

	g.instancesMtx.Lock()
	defer g.instancesMtx.Unlock()

	// Double-check after acquiring write lock
	if inst, ok := g.instances[instanceID]; ok {
		return inst, nil
	}

	// Check if this instance creation failed previously
	if failedAt, ok := g.failedInstances[instanceID]; ok {
		if time.Since(failedAt) < failureBackoff {
			return nil, errInstanceCreationBackoff
		}
		// Backoff expired, clear the failure and retry
		delete(g.failedInstances, instanceID)
	}

	inst, err := g.createInstance(instanceID)
	if err != nil {
		g.failedInstances[instanceID] = time.Now()
		level.Error(g.logger).Log("msg", "instance creation failed, will retry after backoff",
			"backoff", failureBackoff, "tenant", instanceID, "err", err)
		return nil, err
	}

	g.instances[instanceID] = inst
	return inst, nil
}

func (g *Generator) getInstanceByID(id string) (*instance, bool) {
	g.instancesMtx.RLock()
	defer g.instancesMtx.RUnlock()

	inst, ok := g.instances[id]
	return inst, ok
}

func (g *Generator) createInstance(id string) (*instance, error) {
	// Duplicate metrics generation errors occur when creating
	// the wal for a tenant twice. This happens if the wal is
	// create successfully, but the instance is not. On the
	// next push it will panic.
	// We prevent the panic by using a temporary registry
	// for wal and instance creation, and merge it with the
	// main registry only if successful.
	reg := prometheus.NewRegistry()

	wal, err := storage.New(&g.cfg.Storage, g.overrides, id, reg, g.logger)
	if err != nil {
		return nil, err
	}

	inst, err := newInstance(g.cfg, id, g.overrides, wal, g.logger)
	if err != nil {
		_ = wal.Close()
		return nil, err
	}

	err = g.reg.Register(reg)
	if err != nil {
		inst.shutdown()
		return nil, err
	}

	return inst, nil
}

func (g *Generator) CheckReady(_ context.Context) error {
	if g.cfg.ConsumeFromKafka && g.kafkaClient == nil {
		return fmt.Errorf("metrics-generator check ready failed: kafka client not initialized")
	}

	return nil
}

// ExtractNoGenerateMetrics checks for presence of context keys that indicate no
// span-derived metrics should be generated for the request. If any such context
// key is present, this will return true, otherwise it will return false.
func ExtractNoGenerateMetrics(ctx context.Context) bool {
	// check gRPC context
	if len(metadata.ValueFromIncomingContext(ctx, NoGenerateMetricsContextKey)) > 0 {
		return true
	}

	// check http context
	if len(client.FromContext(ctx).Metadata.Get(NoGenerateMetricsContextKey)) > 0 {
		return true
	}

	return false
}
