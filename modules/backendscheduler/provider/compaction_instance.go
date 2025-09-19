package provider

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/blockselector"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// tenantDrainer represents a single tenant compaction drainer.
type tenantDrainer interface {
	run(ctx context.Context) <-chan *work.Job
}

var _ tenantDrainer = (*compactionInstance)(nil)

// compactionInstance operates on a single tenant to push compaction jobs to the compaction provider.
type compactionInstance struct {
	tenant   string
	selector blockselector.CompactionBlockSelector
	cfg      CompactionConfig
	logger   log.Logger
	tracker  jobTracker
}

func newCompactionInstance(
	tenantID string,
	selector blockselector.CompactionBlockSelector,
	cfg CompactionConfig,
	logger log.Logger,
	tracker jobTracker,
) tenantDrainer {
	i := &compactionInstance{
		tenant:   tenantID,
		selector: selector,
		cfg:      cfg,
		logger:   log.With(logger, "tenant_id", tenantID),
		tracker:  tracker,
	}

	return i
}

func (i *compactionInstance) run(ctx context.Context) <-chan *work.Job {
	_, span := tracer.Start(ctx, "compactionInstance.run")
	defer span.End()

	instanceJobs := make(chan *work.Job)

	go func() {
		defer close(instanceJobs)

		level.Debug(i.logger).Log("msg", "compaction instance started")

		jobCount := 0
		var job *work.Job
		var added bool

		for {
			select {
			case <-ctx.Done():
				level.Debug(i.logger).Log("msg", "compaction instance stopping")
				span.AddEvent("context done")
				return
			default:
				job = i.createJob(ctx)
				if job == nil {
					span.AddEvent("tenant exhausted", trace.WithAttributes(
						attribute.String("tenant_id", i.tenant),
					))
					return
				}

				added = i.tracker.addToRecentJobs(ctx, job)
				if !added {
					// Job was a duplicate, try again.
					continue
				}

				if jobCount >= i.cfg.MaxJobsPerTenant {
					level.Info(i.logger).Log("msg", "max jobs per tenant reached, stopping instance")
					span.AddEvent("max jobs per tenant reached", trace.WithAttributes(
						attribute.String("tenant_id", i.tenant),
						attribute.Int("job_count", jobCount),
					))
					return
				}

				select {
				case <-ctx.Done():
					level.Debug(i.logger).Log("msg", "compaction instance stopping")
					span.AddEvent("context done")
					return
				case instanceJobs <- job:
					span.AddEvent("job created", trace.WithAttributes(
						attribute.String("job_id", job.ID),
						attribute.String("tenant_id", i.tenant),
					))

					metricJobsCreated.WithLabelValues(i.tenant).Inc()

					jobCount++
				}
			}
		}
	}()

	return instanceJobs
}

func (i *compactionInstance) createJob(ctx context.Context) *work.Job {
	_, span := tracer.Start(ctx, "compactionInstance.createJob")
	defer span.End()

	span.SetAttributes(attribute.String("tenant_id", i.tenant))

	input, ok := i.getNextBlockIDs()
	if !ok {
		span.AddEvent("not-enough-input-blocks", trace.WithAttributes(
			attribute.Int("input_blocks", len(input)),
		))

		span.SetStatus(codes.Error, "not enough input blocks for compaction")
		return nil
	}

	span.AddEvent("input blocks selected", trace.WithAttributes(
		attribute.Int("input_blocks", len(input)),
		attribute.StringSlice("input_block_ids", input),
	))
	span.SetStatus(codes.Ok, "compaction job created")

	return &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:     i.tenant,
			Compaction: &tempopb.CompactionDetail{Input: input},
		},
	}
}

func (i *compactionInstance) getNextBlockIDs() ([]string, bool) {
	ids := make([]string, 0, i.cfg.MaxInputBlocks)

	toBeCompacted, _ := i.selector.BlocksToCompact()

	if len(toBeCompacted) == 0 {
		return nil, false
	}

	for _, b := range toBeCompacted {
		ids = append(ids, b.BlockID.String())
	}

	return ids, len(ids) >= i.cfg.MinInputBlocks
}
