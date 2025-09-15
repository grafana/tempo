package backendworker

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"time"

	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	backendscheduler_client "github.com/grafana/tempo/modules/backendscheduler/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	tempo_util "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
)

const (
	// ringAutoForgetUnhealthyPeriods is how many consecutive timeout periods an unhealthy instance
	// in the ring will be automatically removed.
	ringAutoForgetUnhealthyPeriods = 2

	// We use a safe default instead of exposing to config option to the user
	// in order to simplify the config.
	ringNumTokens = 512

	backendWorkerRingKey = "backend-worker"
)

var ringOp = ring.NewOp([]ring.InstanceState{ring.ACTIVE}, nil)

type BackendWorker struct {
	services.Service

	cfg              Config
	store            storage.Store
	overrides        overrides.Interface
	backendScheduler tempopb.BackendSchedulerClient

	workerID string

	// Ring used for sharding tenant index writing.
	ringLifecycler *ring.BasicLifecycler
	Ring           *ring.Ring

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// var tracer = otel.Tracer("modules/backendworker")

func New(cfg Config, schedulerClientCfg backendscheduler_client.Config, store storage.Store, overrides overrides.Interface, reg prometheus.Registerer) (*BackendWorker, error) {
	err := ValidateConfig(&cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	w := &BackendWorker{
		cfg:       cfg,
		store:     store,
		overrides: overrides,
	}

	workerID, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	w.workerID = workerID

	level.Info(log.Logger).Log("msg", "backend worker starting", "worker_id", w.workerID)

	schedulerClient, err := backendscheduler_client.New(cfg.BackendSchedulerAddr, schedulerClientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend scheduler client: %w", err)
	}
	w.backendScheduler = schedulerClient

	if w.isSharded() {
		reg = prometheus.WrapRegistererWithPrefix("tempo_", reg)

		lifecyclerStore, err := kv.NewClient(
			cfg.Ring.KVStore,
			ring.GetCodec(),
			kv.RegistererWithKVName(reg, backendWorkerRingKey+"-lifecycler"),
			log.Logger,
		)
		if err != nil {
			return nil, err
		}

		// Define lifecycler delegates in reverse order (last to be called defined first because they're
		// chained via "next delegate").
		delegate := ring.BasicLifecyclerDelegate(w)
		delegate = ring.NewLeaveOnStoppingDelegate(delegate, log.Logger)
		delegate = ring.NewAutoForgetDelegate(ringAutoForgetUnhealthyPeriods*cfg.Ring.HeartbeatTimeout, delegate, log.Logger)

		lifecyclerCfg, err := toBasicLifecyclerConfig(cfg.Ring, log.Logger)
		if err != nil {
			return nil, fmt.Errorf("invalid ring lifecycler config: %w", err)
		}

		w.ringLifecycler, err = ring.NewBasicLifecycler(lifecyclerCfg, backendWorkerRingKey, cfg.OverrideRingKey, lifecyclerStore, delegate, log.Logger, prometheus.WrapRegistererWithPrefix("tempo_", reg))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize backend-worker ring lifecycler: %w", err)
		}

		w.Ring, err = ring.New(cfg.Ring.ToLifecyclerConfig().RingConfig, backendWorkerRingKey, cfg.OverrideRingKey, log.Logger, reg)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize compactor ring: %w", err)
		}
	}

	w.Service = services.NewBasicService(w.starting, w.running, w.stopping)

	return w, nil
}

func (w *BackendWorker) starting(ctx context.Context) (err error) {
	defer func() {
		if err == nil || w.subservices == nil {
			return
		}

		if stopErr := services.StopManagerAndAwaitStopped(context.Background(), w.subservices); stopErr != nil {
			level.Error(log.Logger).Log("msg", "failed to gracefully stop backend-worker dependencies", "err", stopErr)
		}
	}()

	if w.isSharded() {
		w.subservices, err = services.NewManager(w.ringLifecycler, w.Ring)
		if err != nil {
			return fmt.Errorf("failed to create subservices: %w", err)
		}
		w.subservicesWatcher = services.NewFailureWatcher()
		w.subservicesWatcher.WatchManager(w.subservices)

		err := services.StartManagerAndAwaitHealthy(ctx, w.subservices)
		if err != nil {
			return fmt.Errorf("failed to start subservices: %w", err)
		}

		// Wait until the ring client detected this instance in the ACTIVE state.
		level.Info(log.Logger).Log("msg", "waiting until compactor is ACTIVE in the ring")
		ctxWithTimeout, cancel := context.WithTimeout(ctx, w.cfg.Ring.WaitActiveInstanceTimeout)
		defer cancel()
		if err := ring.WaitInstanceState(ctxWithTimeout, w.Ring, w.ringLifecycler.GetInstanceID(), ring.ACTIVE); err != nil {
			return err
		}
		level.Info(log.Logger).Log("msg", "compactor is ACTIVE in the ring")

		// In the event of a cluster cold start we may end up in a situation where each new compactor
		// instance starts at a slightly different time and thus each one starts with a different state
		// of the ring. It's better to just wait the ring stability for a short time.
		if w.cfg.Ring.WaitStabilityMinDuration > 0 {
			minWaiting := w.cfg.Ring.WaitStabilityMinDuration
			maxWaiting := w.cfg.Ring.WaitStabilityMaxDuration

			level.Info(log.Logger).Log("msg", "waiting until compactor ring topology is stable", "min_waiting", minWaiting.String(), "max_waiting", maxWaiting.String())
			if err := ring.WaitRingStability(ctx, w.Ring, ringOp, minWaiting, maxWaiting); err != nil {
				level.Warn(log.Logger).Log("msg", "compactor ring topology is not stable after the max waiting time, proceeding anyway")
			} else {
				level.Info(log.Logger).Log("msg", "compactor ring topology is stable")
			}
		}
	}

	if w.cfg.Poll {
		w.store.EnablePolling(ctx, w, false)
	}

	return nil
}

func (w *BackendWorker) running(ctx context.Context) error {
	level.Info(log.Logger).Log("msg", "backend worker running")

	b := backoff.New(ctx, w.cfg.Backoff)

	jobCtx := ctx
	if w.cfg.FinishOnShutdownTimeout > 0 {
		var jobsCancel context.CancelFunc
		jobCtx, jobsCancel = createShutdownContext(ctx, w.cfg.FinishOnShutdownTimeout)
		defer jobsCancel()
	}

	if w.subservices != nil {
		for {
			select {
			case <-ctx.Done():
				return nil
			case err := <-w.subservicesWatcher.Chan():
				return fmt.Errorf("worker subservices failed: %w", err)
			default:
				if err := w.processJobs(jobCtx); err != nil {
					level.Error(log.Logger).Log("msg", "error processing jobs", "err", err, "backoff", b.NextDelay())
					b.Wait()
					continue
				}

				b.Reset()
			}
		}
	} else {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				if err := w.processJobs(jobCtx); err != nil {
					level.Error(log.Logger).Log("msg", "error processing jobs", "err", err, "backoff", b.NextDelay())
					b.Wait()
					continue
				}

				b.Reset()
			}
		}
	}
}

func (w *BackendWorker) processJobs(ctx context.Context) error {
	var (
		resp *tempopb.NextJobResponse
		err  error
	)

	// Request next job
	err = w.callSchedulerWithBackoff(ctx, func(ctx context.Context) error {
		var funcErr error
		resp, funcErr = w.backendScheduler.Next(ctx, &tempopb.NextJobRequest{
			WorkerId: w.workerID,
		})
		if funcErr != nil {
			if errStatus, ok := status.FromError(funcErr); ok {
				if errStatus.Code() == codes.NotFound {
					return errStatus.Err()
				}
			}

			return fmt.Errorf("error getting next job: %w", funcErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed processing jobs: %w", err)
	}

	if resp == nil || resp.JobId == "" {
		return fmt.Errorf("no jobs available")
	}

	metricWorkerJobsTotal.WithLabelValues().Inc()

	switch resp.Type {
	case tempopb.JobType_JOB_TYPE_COMPACTION:
		return w.processCompactionJob(ctx, resp)
	case tempopb.JobType_JOB_TYPE_RETENTION:
		return w.processRetentionJob(ctx, resp)
	default:
		return fmt.Errorf("unknown job type: %s", resp.Type.String())
	}
}

func (w *BackendWorker) processCompactionJob(ctx context.Context, resp *tempopb.NextJobResponse) error {
	if resp.Detail.Tenant == "" {
		metricWorkerBadJobsReceived.WithLabelValues("no_tenant").Inc()
		return w.failJob(ctx, resp.JobId, "received compaction job with empty tenant")
	}

	level.Debug(log.Logger).Log("msg", "received job", "job_id", resp.JobId, "tenant", resp.Detail.Tenant)

	blockMetas := w.store.BlockMetas(resp.Detail.Tenant)

	// Collect the metas which match the IDs in the job
	var sourceMetas []*backend.BlockMeta
	for _, blockMeta := range blockMetas {
		for _, blockID := range resp.Detail.Compaction.Input {
			if blockMeta.BlockID.String() == blockID {
				sourceMetas = append(sourceMetas, blockMeta)
			}
		}
	}

	// Execute compaction using existing logic
	newCompacted, err := w.compact(ctx, sourceMetas, resp.Detail.Tenant)
	if err != nil {
		return w.failJob(ctx, resp.JobId, fmt.Sprintf("error compacting blocks: %v", err))
	}

	var newIDs []string
	for _, blockMeta := range newCompacted {
		newIDs = append(newIDs, blockMeta.BlockID.String())
	}

	// Mark job as complete
	err = w.callSchedulerWithBackoff(ctx, func(ctx context.Context) error {
		_, err = w.backendScheduler.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  resp.JobId,
			Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
			Compaction: &tempopb.CompactionDetail{
				Output: newIDs,
			},
		})
		if err != nil {
			return fmt.Errorf("failed marking job %q as complete: %w", resp.JobId, err)
		}

		return nil
	})
	if err != nil {
		return w.failJob(ctx, resp.JobId, fmt.Sprintf("error marking job as complete: %v", err))
	}

	return nil
}

func (w *BackendWorker) processRetentionJob(ctx context.Context, resp *tempopb.NextJobResponse) error {
	level.Debug(log.Logger).Log("msg", "received retention job", "job_id", resp.JobId, "tenant", resp.Detail.Tenant)
	w.store.RetainWithConfig(ctx, &w.cfg.Compactor, ownsEverythingSharder{}, w)

	err := w.callSchedulerWithBackoff(ctx, func(ctx context.Context) error {
		_, err := w.backendScheduler.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  resp.JobId,
			Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
		})
		if err != nil {
			return fmt.Errorf("failed marking job %q as complete: %w", resp.JobId, err)
		}

		return nil
	})
	if err != nil {
		return w.failJob(ctx, resp.JobId, fmt.Sprintf("error marking job as complete: %v", err))
	}

	return nil
}

func (w *BackendWorker) stopping(_ error) error {
	if w.subservices != nil {
		return services.StopManagerAndAwaitStopped(context.Background(), w.subservices)
	}

	level.Info(log.Logger).Log("msg", "backend worker stopped")

	return nil
}

func (w *BackendWorker) failJob(ctx context.Context, jobID string, errMsg string) error {
	level.Error(log.Logger).Log("msg", "job failed", "job_id", jobID, "error", errMsg)

	err := w.callSchedulerWithBackoff(ctx, func(ctx context.Context) error {
		_, err := w.backendScheduler.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
			JobId:  jobID,
			Status: tempopb.JobStatus_JOB_STATUS_FAILED,
			Error:  errMsg,
		})
		if err != nil {
			return fmt.Errorf("failed marking job %q as failed: %w", jobID, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error marking job %q as failed: %w", jobID, err)
	}

	return fmt.Errorf("%s", errMsg)
}

func (w *BackendWorker) compact(ctx context.Context, blockMetas []*backend.BlockMeta, tenantID string) ([]*backend.BlockMeta, error) {
	return w.store.CompactWithConfig(ctx, blockMetas, tenantID, &w.cfg.Compactor, w, w)
}

// Combine implements tempodb.CompactorSharder
func (w *BackendWorker) Combine(dataEncoding string, tenantID string, objs ...[]byte) ([]byte, bool, error) {
	combinedObj, wasCombined, err := model.StaticCombiner.Combine(dataEncoding, objs...)
	if err != nil {
		return nil, false, err
	}

	maxBytes := w.overrides.MaxBytesPerTrace(tenantID)
	if maxBytes == 0 || len(combinedObj) < maxBytes {
		return combinedObj, wasCombined, nil
	}

	// technically neither of these conditions should ever be true, we are adding them as guard code
	// for the following logic
	if len(objs) == 0 {
		return []byte{}, wasCombined, nil
	}
	if len(objs) == 1 {
		return objs[0], wasCombined, nil
	}

	totalDiscarded := countSpans(dataEncoding, objs[1:]...)
	overrides.RecordDiscardedSpans(totalDiscarded, overrides.ReasonCompactorDiscardedSpans, tenantID)
	return objs[0], wasCombined, nil
}

// Copied from compactor module.  Centralize?
func countSpans(dataEncoding string, objs ...[]byte) (total int) {
	var traceID string
	decoder, err := model.NewObjectDecoder(dataEncoding)
	if err != nil {
		return 0
	}

	for _, o := range objs {
		t, err := decoder.PrepareForRead(o)
		if err != nil {
			continue
		}

		for _, b := range t.ResourceSpans {
			for _, ilm := range b.ScopeSpans {
				if len(ilm.Spans) > 0 && traceID == "" {
					traceID = tempo_util.TraceIDToHexString(ilm.Spans[0].TraceId)
				}
				total += len(ilm.Spans)
			}
		}
	}

	level.Debug(log.Logger).Log("msg", "max size of trace exceeded", "traceId", traceID, "discarded_span_count", total)

	return
}

func (w *BackendWorker) Owns(hash string) bool {
	if !w.isSharded() {
		return true
	}

	level.Debug(log.Logger).Log("msg", "checking hash", "hash", hash)

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(hash))
	hash32 := hasher.Sum32()

	rs, err := w.Ring.Get(hash32, ringOp, []ring.InstanceDesc{}, nil, nil)
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to get ring", "err", err)
		return false
	}

	if len(rs.Instances) != 1 {
		level.Error(log.Logger).Log("msg", "unexpected number of compactors in the shard (expected 1, got %d)", len(rs.Instances))
		return false
	}

	ringAddr := w.ringLifecycler.GetInstanceAddr()

	level.Debug(log.Logger).Log("msg", "checking addresses", "owning_addr", rs.Instances[0].Addr, "this_addr", ringAddr)

	return rs.Instances[0].Addr == ringAddr
}

func (w *BackendWorker) RecordDiscardedSpans(count int, tenantID string, traceID string, rootSpanName string, rootServiceName string) {
	level.Warn(log.Logger).Log("msg", "max size of trace exceeded", "tenant", tenantID, "traceId", traceID,
		"rootSpanName", rootSpanName, "rootServiceName", rootServiceName, "discarded_span_count", count)
	overrides.RecordDiscardedSpans(count, overrides.ReasonCompactorDiscardedSpans, tenantID)
}

// BlockRetentionForTenant implements CompactorOverrides
func (w *BackendWorker) BlockRetentionForTenant(tenantID string) time.Duration {
	return w.overrides.BlockRetention(tenantID)
}

// CompactionDisabledForTenant implements CompactorOverrides
func (w *BackendWorker) CompactionDisabledForTenant(tenantID string) bool {
	return w.overrides.CompactionDisabled(tenantID)
}

func (w *BackendWorker) MaxBytesPerTraceForTenant(tenantID string) int {
	return w.overrides.MaxBytesPerTrace(tenantID)
}

func (w *BackendWorker) MaxCompactionRangeForTenant(tenantID string) time.Duration {
	return w.overrides.MaxCompactionRange(tenantID)
}

func (w *BackendWorker) callSchedulerWithBackoff(ctx context.Context, f func(context.Context) error) error {
	var (
		b   = backoff.New(ctx, w.cfg.Backoff)
		err error
	)

	for b.Ongoing() {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err = f(ctx); err != nil {
				if ctx.Err() != nil {
					// Parent was canceled while executing, return
					return nil
				}

				level.Error(log.Logger).Log("msg", "error calling scheduler", "err", err, "backoff", b.NextDelay())
				metricWorkerCallRetries.WithLabelValues().Inc()
				// Add jitter so all workers don't all retry at once and cause a thundering herd.
				time.Sleep(time.Duration(rand.Float32() * float32(1*time.Second)))
				b.Wait()
				continue
			}

			b.Reset()
			return nil
		}
	}

	return fmt.Errorf("backoff terminated: %w, %w", b.Err(), err)
}

func (w *BackendWorker) isSharded() bool {
	return w.cfg.Ring.KVStore.Store != ""
}

// OnRingInstanceRegister is called while the lifecycler is registering the
// instance within the ring and should return the state and set of tokens to
// use for the instance itself.
func (w *BackendWorker) OnRingInstanceRegister(_ *ring.BasicLifecycler, ringDesc ring.Desc, instanceExists bool, _ string, instanceDesc ring.InstanceDesc) (ring.InstanceState, ring.Tokens) {
	// When we initialize the compactor instance in the ring we want to start from
	// a clean situation, so whatever is the state we set it ACTIVE, while we keep existing
	// tokens (if any) or the ones loaded from file.
	var tokens []uint32
	if instanceExists {
		tokens = instanceDesc.GetTokens()
	}

	takenTokens := ringDesc.GetTokens()
	gen := ring.NewRandomTokenGenerator()
	newTokens := gen.GenerateTokens(ringNumTokens-len(tokens), takenTokens)

	// Tokens sorting will be enforced by the parent caller.
	tokens = append(tokens, newTokens...)

	return ring.ACTIVE, tokens
}

// OnRingInstanceTokens is called once the instance tokens are set and are
// stable within the ring (honoring the observe period, if set).
func (w *BackendWorker) OnRingInstanceTokens(*ring.BasicLifecycler, ring.Tokens) {}

// OnRingInstanceStopping is called while the lifecycler is stopping. The lifecycler
// will continue to heartbeat the ring the this function is executing and will proceed
// to unregister the instance from the ring only after this function has returned.
func (w *BackendWorker) OnRingInstanceStopping(*ring.BasicLifecycler) {}

// OnRingInstanceHeartbeat is called while the instance is updating its heartbeat
// in the ring.
func (w *BackendWorker) OnRingInstanceHeartbeat(*ring.BasicLifecycler, *ring.Desc, *ring.InstanceDesc) {
}

type ownsEverythingSharder struct {
	w *BackendWorker
}

var _ tempodb.CompactorSharder = ownsEverythingSharder{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}

func (s ownsEverythingSharder) RecordDiscardedSpans(count int, tenantID string, traceID string, rootSpanName string, rootServiceName string) {
	s.w.RecordDiscardedSpans(count, tenantID, traceID, rootSpanName, rootServiceName)
}

func (s ownsEverythingSharder) Combine(dataEncoding string, tenantID string, objs ...[]byte) ([]byte, bool, error) {
	return s.w.Combine(dataEncoding, tenantID, objs...)
}

// createShutdownContext creates a context that starts a timeout only after parentCtx is cancelled
func createShutdownContext(parentCtx context.Context, shutdownTimeout time.Duration) (context.Context, context.CancelFunc) {
	jobsCtx, jobsCancel := context.WithCancel(context.Background())

	go func() {
		<-parentCtx.Done() // Wait for parent cancellation

		// Now start the shutdown timeout
		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer timeoutCancel()

		select {
		case <-timeoutCtx.Done():
			// Timeout expired, force cancel jobs
			level.Warn(log.Logger).Log("msg", "job timeout expired")
			jobsCancel()
		case <-jobsCtx.Done():
			// Jobs completed gracefully before timeout
			return
		}
	}()

	return jobsCtx, jobsCancel
}
