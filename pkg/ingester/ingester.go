package ingester

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util"

	"github.com/grafana/frigg/friggdb"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/ingester/client"
	"github.com/grafana/frigg/pkg/storage"
	"github.com/grafana/frigg/pkg/util/validation"
)

// ErrReadOnly is returned when the ingester is shutting down and a push was
// attempted.
var ErrReadOnly = errors.New("Ingester is shutting down")

var readinessProbeSuccess = []byte("Ready")

var metricFlushQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "frigg",
	Name:      "ingester_flush_queue_length",
	Help:      "The total number of series pending in the flush queue.",
})

// Ingester builds chunks for incoming log streams.
type Ingester struct {
	cfg          Config
	clientConfig client.Config

	shutdownMtx  sync.Mutex // Allows processes to grab a lock and prevent a shutdown
	instancesMtx sync.RWMutex
	instances    map[string]*instance
	readonly     bool

	lifecycler *ring.Lifecycler
	store      storage.Store

	done     sync.WaitGroup
	quit     chan struct{}
	quitting chan struct{}

	// One queue per flush thread.  Fingerprint is used to
	// pick a queue.
	flushQueues     []*util.PriorityQueue
	flushQueueIndex int
	flushQueuesDone sync.WaitGroup

	limiter *Limiter
	wal     friggdb.WAL
}

// New makes a new Ingester.
func New(cfg Config, clientConfig client.Config, store storage.Store, limits *validation.Overrides) (*Ingester, error) {
	if cfg.ingesterClientFactory == nil {
		cfg.ingesterClientFactory = client.New
	}

	i := &Ingester{
		cfg:          cfg,
		clientConfig: clientConfig,
		instances:    map[string]*instance{},
		store:        store,
		quit:         make(chan struct{}),
		flushQueues:  make([]*util.PriorityQueue, cfg.ConcurrentFlushes),
		quitting:     make(chan struct{}),
	}

	i.flushQueuesDone.Add(cfg.ConcurrentFlushes)
	for j := 0; j < cfg.ConcurrentFlushes; j++ {
		i.flushQueues[j] = util.NewPriorityQueue(metricFlushQueueLength)
		go i.flushLoop(j)
	}

	var err error
	i.lifecycler, err = ring.NewLifecycler(cfg.LifecyclerConfig, i, "ingester", ring.IngesterRingKey)
	if err != nil {
		return nil, fmt.Errorf("NewLifecycler failed %v", err)
	}

	i.lifecycler.Start()

	// Now that the lifecycler has been created, we can create the limiter
	// which depends on it.
	i.limiter = NewLimiter(limits, i.lifecycler, cfg.LifecyclerConfig.RingConfig.ReplicationFactor)

	i.wal, err = i.store.WAL()
	if err != nil {
		return nil, err
	}
	err = i.replayWal()

	i.done.Add(1)
	go i.loop()

	return i, nil
}

func (i *Ingester) loop() {
	defer i.done.Done()

	flushTicker := time.NewTicker(i.cfg.FlushCheckPeriod)
	defer flushTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			i.sweepUsers(false)

		case <-i.quit:
			return
		}
	}
}

// Shutdown stops the ingester.
func (i *Ingester) Shutdown() {
	close(i.quit)
	i.done.Wait()

	i.lifecycler.Shutdown()
}

// Stopping helps cleaning up resources before actual shutdown
func (i *Ingester) Stopping() {
	close(i.quitting)
}

// Push implements friggpb.Pusher.
func (i *Ingester) Push(ctx context.Context, req *friggpb.PushRequest) (*friggpb.PushResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	} else if i.readonly {
		return nil, ErrReadOnly
	}

	instance, err := i.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	err = instance.Push(ctx, req)
	return &friggpb.PushResponse{}, err
}

// FindTraceByID implements friggpb.Querier.
func (i *Ingester) FindTraceByID(ctx context.Context, req *friggpb.TraceByIDRequest) (*friggpb.TraceByIDResponse, error) {
	if !validation.ValidTraceID(req.TraceID) {
		return nil, fmt.Errorf("invalid trace id")
	}

	instanceID, err := user.ExtractOrgID(ctx)
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &friggpb.TraceByIDResponse{}, nil
	}

	trace, err := inst.FindTraceByID(req.TraceID)
	if err != nil {
		return nil, err
	}

	return &friggpb.TraceByIDResponse{
		Trace: trace,
	}, nil
}

// Check implements grpc_health_v1.HealthCheck.
func (*Ingester) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

// Watch implements grpc_health_v1.HealthCheck.
func (*Ingester) Watch(*grpc_health_v1.HealthCheckRequest, grpc_health_v1.Health_WatchServer) error {
	return nil
}

// ReadinessHandler is used to indicate to k8s when the ingesters are ready for
// the addition removal of another ingester. Returns 200 when the ingester is
// ready, 500 otherwise.
func (i *Ingester) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if err := i.lifecycler.CheckReady(r.Context()); err != nil {
		http.Error(w, "Not ready: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(readinessProbeSuccess); err != nil {
		level.Error(util.Logger).Log("msg", "error writing success message", "error", err)
	}
}

func (i *Ingester) getOrCreateInstance(instanceID string) (*instance, error) {
	inst, ok := i.getInstanceByID(instanceID)
	if ok {
		return inst, nil
	}

	i.instancesMtx.Lock()
	defer i.instancesMtx.Unlock()
	inst, ok = i.instances[instanceID]
	if !ok {
		var err error
		inst, err = newInstance(instanceID, i.limiter, i.wal)
		if err != nil {
			return nil, err
		}
		i.instances[instanceID] = inst
	}
	return inst, nil
}

func (i *Ingester) getInstanceByID(id string) (*instance, bool) {
	i.instancesMtx.RLock()
	defer i.instancesMtx.RUnlock()

	inst, ok := i.instances[id]
	return inst, ok
}

func (i *Ingester) getInstances() []*instance {
	i.instancesMtx.RLock()
	defer i.instancesMtx.RUnlock()

	instances := make([]*instance, 0, len(i.instances))
	for _, instance := range i.instances {
		instances = append(instances, instance)
	}
	return instances
}

// StopIncomingRequests implements ring.Lifecycler.
func (i *Ingester) StopIncomingRequests() {
	i.shutdownMtx.Lock()
	defer i.shutdownMtx.Unlock()

	i.instancesMtx.Lock()
	defer i.instancesMtx.Unlock()

	i.readonly = true
}

// TransferOut implements ring.Lifecycler.
func (i *Ingester) TransferOut(ctx context.Context) error {
	if i.cfg.MaxTransferRetries <= 0 {
		return fmt.Errorf("transfers disabled")
	}

	// need to decide what, if any support, we're going to have here
	return nil
}

func (i *Ingester) replayWal() error {
	blocks, err := i.wal.AllBlocks()
	if err != nil {
		return nil
	}

	read := &friggpb.Trace{}
	for _, b := range blocks {
		err = b.Iterator(read, func(id friggdb.ID, r proto.Message) (bool, error) {
			req := r.(*friggpb.Trace)

			_, tenantID, _, _ := b.Identity()
			instance, err := i.getOrCreateInstance(tenantID)
			if err != nil {
				return false, err
			}

			err = instance.PushTrace(context.Background(), req)
			if err != nil {
				return false, err
			}

			return true, nil
		})
		if err != nil {
			// jpe:  this is gorpy and error prone.  change to use the wal work dir?
			// clean up any instance headblocks that were created to keep from replaying again and again
			for _, instance := range i.instances {
				instance.headBlock.Clear()
			}

			return err
		}
		b.Clear()
	}

	return nil
}
