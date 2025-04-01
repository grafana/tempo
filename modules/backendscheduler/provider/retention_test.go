package provider

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestRetentionProvider(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cfg := RetentionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.Interval = 10 * time.Millisecond

	workCfg := work.Config{}
	workCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	w := work.New(workCfg)

	logger := log.NewLogfmtLogger(os.Stderr)

	p := NewRetentionProvider(
		cfg,
		w,
		logger,
	)

	jobChan := p.Start(ctx)

	var receivedJobs []*work.Job

	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case job := <-jobChan:
				err := w.AddJob(job)
				if job == nil {
					require.Error(t, err)
					require.Equal(t, work.ErrJobNil, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, tempopb.JobType_JOB_TYPE_RETENTION, job.Type)
					job.Start() // mark the job as started so that we avoid new retention jobs
				}

				receivedJobs = append(receivedJobs, job)
			}
		}
	}()

	<-ctx.Done()

	for _, job := range receivedJobs {
		require.NotNil(t, job)
		require.Equal(t, tempopb.JobType_JOB_TYPE_RETENTION, job.Type)
		// Retention jobs should not have a tenant
		require.Equal(t, "", job.Tenant())
		require.NotEqual(t, "", job.ID)
	}

	// Since we started the job, no other jobs should be in the queue
	require.Len(t, receivedJobs, 1)
}
