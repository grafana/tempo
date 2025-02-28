package backendscheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
)

const (
	ringKey = "backend-scheduler"
)

// BackendScheduler manages scheduling and execution of backend jobs
type BackendScheduler struct {
	services.Service

	cfg   Config
	store storage.Store

	// Ring lifecycle management
	// ringLifecycler *ring.BasicLifecycler
	// ring           *ring.Ring

	// Subservices management
	// subservices        *services.Manager
	// subservicesWatcher *services.FailureWatcher

	// track jobs, keyed by job ID
	jobs    map[string]*Job
	jobsMtx sync.RWMutex
}

// Interface that compactor implements
// TODO: is this interface needed?  Currently proto is enforcing the interface at the API and we likely don't need this or its methods.
type JobProcessor interface {
	GetJob(ctx context.Context) (*Job, error)
	StartJob(ctx context.Context, jobID string) error
	CompleteJob(ctx context.Context, jobID string) error
	FailJob(ctx context.Context, jobID string, err error) error
}

// New creates a new BackendScheduler
func New(cfg Config, store storage.Store) (*BackendScheduler, error) {
	s := &BackendScheduler{
		cfg:   cfg,
		jobs:  make(map[string]*Job),
		store: store,
	}

	// lifecyclerStore, err := kv.NewClient(
	// 	cfg.Ring.KVStore,
	// 	ring.GetCodec(),
	// 	kv.RegistererWithKVName(reg, "backend-scheduler"),
	// 	log.Logger,
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create lifecycler store: %w", err)
	// }
	//
	// lifecyclerCfg, err := cfg.Ring.toLifecyclerConfig()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create lifecycler config: %w", err)
	// }
	//
	// s.ringLifecycler, err = ring.NewBasicLifecycler(lifecyclerCfg, ringKey, ringKey, lifecyclerStore, nil, log.Logger, reg)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create ring lifecycler: %w", err)
	// }
	//
	// ringCfg := cfg.Ring.ToRingConfig()
	// s.ring, err = ring.New(ringCfg, ringKey, ringKey, log.Logger, reg)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create ring: %w", err)
	// }

	s.Service = services.NewBasicService(s.starting, s.running, s.stopping)
	return s, nil
}

func (s *BackendScheduler) starting(ctx context.Context) error {
	// TODO: the jobs are not populated on startup, either from what was
	// persisted, or invented new.

	// var err error
	// s.subservices, err = services.NewManager(s.ringLifecycler, s.ring)
	// if err != nil {
	// 	return fmt.Errorf("failed to create subservices: %w", err)
	// }

	// s.subservicesWatcher = services.NewFailureWatcher()
	// s.subservicesWatcher.WatchManager(s.subservices)

	// if err := services.StartManagerAndAwaitHealthy(ctx, s.subservices); err != nil {
	// 	return fmt.Errorf("failed to start subservices: %w", err)
	// }

	// Wait until this instance is ACTIVE in the ring
	// level.Info(log.Logger).Log("msg", "waiting until backend scheduler is ACTIVE in the ring")
	// if err := ring.WaitInstanceState(ctx, s.ring, s.ringLifecycler.GetInstanceID(), ring.ACTIVE); err != nil {
	// 	return err
	// }
	// level.Info(log.Logger).Log("msg", "backend scheduler is ACTIVE in the ring")

	return nil
}

func (s *BackendScheduler) running(ctx context.Context) error {
	level.Info(log.Logger).Log("msg", "backend scheduler running")

	ticker := time.NewTicker(s.cfg.ScheduleInterval)
	defer ticker.Stop()

	if len(s.jobs) == 0 {
		if err := s.ScheduleOnce(ctx); err != nil {
			return fmt.Errorf("failed to schedule initial jobs: %w", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		// case err := <-s.subservicesWatcher.Chan():
		// 	return fmt.Errorf("backend scheduler subservice failed: %w", err)
		case <-ticker.C:
			// if !s.isLeader() {
			// 	s.metrics.schedulerIsLeader.Set(0)
			// 	continue
			// }
			// s.metrics.schedulerIsLeader.Set(1)

			if err := s.ScheduleOnce(ctx); err != nil {
				level.Error(log.Logger).Log("msg", "scheduling cycle failed", "err", err)
				schedulingCycles.WithLabelValues("failed").Inc()
			} else {
				schedulingCycles.WithLabelValues("success").Inc()
			}

			// TOOO: Impelment creating jobs.  Start with compaction jobs.
		}
	}
}

func (s *BackendScheduler) stopping(_ error) error {
	// TODO: consider flushing job state
	return nil
}

// ScheduleOnce schedules jobs for compaction and performs cleanup.
func (s *BackendScheduler) ScheduleOnce(ctx context.Context) error {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	// Clean up completed/failed jobs
	for id, job := range s.jobs {
		if job.IsComplete() || job.IsFailed() {
			delete(s.jobs, id)
		}
	}

	if len(s.jobs) > 0 {
		return nil
	}

	compactions := s.store.Compactions(ctx)

	for _, job := range compactions {
		detail := job.Detail.(*tempopb.JobDetail_Compaction).Compaction

		if err := s.createCompactionJob(ctx, job.Tenant, detail.Input); err != nil {
			return fmt.Errorf("failed to create compaction job: %w", err)
		}
	}

	return nil
}

// CreateJob creates a new job
func (s *BackendScheduler) CreateJob(ctx context.Context, job *Job) error {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	// Validate max concurrent jobs per tenant
	activeCount := 0
	for _, j := range s.jobs {
		if j.JobDetail.Tenant == job.JobDetail.Tenant && !j.IsComplete() && !j.IsFailed() {
			activeCount++
		}
	}

	// if activeCount >= s.cfg.MaxConcurrentJobs {
	// 	return fmt.Errorf("max concurrent jobs (%d) reached for tenant %s", s.cfg.MaxConcurrentJobs, job.Job.Tenant)
	// }

	s.jobs[job.ID] = job
	jobsCreated.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Inc()
	jobsActive.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Inc()

	level.Info(log.Logger).Log("msg", "created new job", "job_id", job.ID, "tenant", job.JobDetail.Tenant, "type", job.Type.String())
	return nil
}

// GetJob returns a job by ID
func (s *BackendScheduler) GetJob(ctx context.Context, id string) (*Job, error) {
	s.jobsMtx.RLock()
	defer s.jobsMtx.RUnlock()

	job, exists := s.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	return job, nil
}

// CompleteJob marks a job as completed
func (s *BackendScheduler) CompleteJob(ctx context.Context, id string) error {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Complete()
	jobsCompleted.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Inc()
	jobsActive.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "completed job", "job_id", id, "tenant", job.JobDetail.Tenant, "type", job.Type.String())
	return nil
}

// FailJob marks a job as failed
func (s *BackendScheduler) FailJob(ctx context.Context, id string) error {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Fail()
	jobsFailed.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Inc()
	jobsActive.WithLabelValues(job.JobDetail.Tenant, job.Type.String()).Dec()

	level.Info(log.Logger).Log("msg", "failed job", "job_id", id, "tenant", job.JobDetail.Tenant, "type", job.Type.String())
	return nil
}

// Next implements the BackendSchedulerServer interface
func (s *BackendScheduler) Next(ctx context.Context, req *tempopb.NextJobRequest) (*tempopb.NextJobResponse, error) {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	// Find next available job
	for id, job := range s.jobs {
		// TODO: check if we have pending jobs for this workerID and hand those out first.

		if job.Status() == JobStatusPending {
			// Honor the request job type if specified
			if req.Type != tempopb.JobType_JOB_TYPE_UNSPECIFIED && job.Type != req.Type {
				continue
			}

			// Create response with job details
			resp := &tempopb.NextJobResponse{
				JobId:  id,
				Type:   job.Type,
				Detail: job.JobDetail,
			}

			// Mark job as running
			job.Start(req.WorkerId)

			level.Info(log.Logger).Log("msg", "assigned job to worker", "job_id", id, "worker", req.WorkerId)
			return resp, nil
		}
	}

	// No jobs available
	// TODO: consider returning a status code to indicate no jobs available
	return nil, nil
}

// UpdateJob implements the BackendSchedulerServer interface
func (s *BackendScheduler) UpdateJob(ctx context.Context, req *tempopb.UpdateJobStatusRequest) (*tempopb.UpdateJobStatusResponse, error) {
	s.jobsMtx.Lock()
	defer s.jobsMtx.Unlock()

	job, exists := s.jobs[req.JobId]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", req.JobId)
	}

	switch req.Status {
	case tempopb.JobStatus_JOB_STATUS_SUCCEEDED:
		job.Complete()
		jobsCompleted.WithLabelValues(job.JobDetail.Tenant, job.JobDetail.Tenant).Inc()
		level.Info(log.Logger).Log("msg", "job completed", "job_id", req.JobId)

	case tempopb.JobStatus_JOB_STATUS_FAILED:
		job.Fail()
		jobsFailed.WithLabelValues(job.JobDetail.Tenant, job.JobDetail.Tenant).Inc()
		level.Error(log.Logger).Log("msg", "job failed", "job_id", req.JobId, "error", req.Error)

	default:
		return nil, fmt.Errorf("invalid job status: %v", req.Status)
	}

	return &tempopb.UpdateJobStatusResponse{
		Success: true,
	}, nil
}

// ListJobs returns all jobs for a given tenant
func (s *BackendScheduler) ListJobs(ctx context.Context, tenantID string) ([]*Job, error) {
	s.jobsMtx.RLock()
	defer s.jobsMtx.RUnlock()

	var jobs []*Job
	for _, job := range s.jobs {
		if job.JobDetail.Tenant == tenantID {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

// CreateCompactionJob creates a new compaction job for the given tenant and blocks.
// Must be called under jobsMtx lock.
func (s *BackendScheduler) createCompactionJob(ctx context.Context, tenantID string, input []string) error {
	// Skip blocks which already have a job
	for _, blockID := range input {
		for _, j := range s.jobs {
			if j.JobDetail.Tenant == tenantID {
				// TODO: switch on job type to skip non-compaction jobs

				for _, b := range j.JobDetail.Detail.(*tempopb.JobDetail_Compaction).Compaction.Input {
					if b == blockID {
						// TODO: consider continue when we have a stripe of like-blocks.
						return nil
					}
				}
			}
		}
	}

	jobID := uuid.New().String()

	job := &Job{
		ID: jobID,
		// Type: JobTypeCompaction,
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: tenantID,
			Detail: &tempopb.JobDetail_Compaction{
				Compaction: &tempopb.CompactionDetail{
					Input: input,
				},
			},
		},
	}

	// Store the job
	s.jobs[jobID] = job

	// Update metrics
	jobsCreated.WithLabelValues(tenantID, job.Type.String()).Inc()
	jobsActive.WithLabelValues(tenantID, job.Type.String()).Inc()

	return nil
}
