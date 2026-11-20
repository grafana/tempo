package backendscheduler

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/backendscheduler/work"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
)

// TestLastSuccessfulJobTimestamp verifies that the per-job-type timestamp of the last successful job
// is exported as 0 from startup and set only when a job succeeds. Dashboards show 0 as "Never", and
// derive "time since last successful run" from any other value, so a failed job must not refresh it,
// or a scheduler whose jobs all fail would look healthy.
func TestLastSuccessfulJobTimestamp(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.LocalWorkPath = t.TempDir()

	var (
		ctx, cancel   = context.WithCancel(context.Background())
		store, rr, ww = newStore(ctx, t, cfg.LocalWorkPath)
	)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	tests := []struct {
		name      string
		status    tempopb.JobStatus
		wantIsSet bool
	}{
		{"failed job leaves the timestamp at 0", tempopb.JobStatus_JOB_STATUS_FAILED, false},
		{"running update leaves the timestamp at 0", tempopb.JobStatus_JOB_STATUS_RUNNING, false},
		{"succeeded job sets the timestamp", tempopb.JobStatus_JOB_STATUS_SUCCEEDED, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricLastSuccessfulJobTimestamp.Reset() // isolate from other tests in this package

			s, err := New(cfg, store, limits, rr, ww)
			require.NoError(t, err)

			// Every job type is exported at 0 before any job has run.
			require.Equal(t, len(tempopb.JobType_name)-1, testutil.CollectAndCount(metricLastSuccessfulJobTimestamp))

			j := &work.Job{
				ID:        uuid.New().String(),
				Type:      tempopb.JobType_JOB_TYPE_RETENTION,
				JobDetail: tempopb.JobDetail{Tenant: "tenant-last-success"},
			}
			s.work.RegisterJob(j)
			require.NoError(t, s.work.AddJob(j))
			s.work.StartJob(j.ID)

			before := time.Now()
			_, err = s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{JobId: j.ID, Status: tt.status})
			require.NoError(t, err)

			got := testutil.ToFloat64(metricLastSuccessfulJobTimestamp.WithLabelValues(j.Type.String()))
			if !tt.wantIsSet {
				require.Zero(t, got)
				return
			}

			require.GreaterOrEqual(t, got, float64(before.Unix()))
			require.LessOrEqual(t, got, float64(time.Now().Unix()+1))
		})
	}
}
