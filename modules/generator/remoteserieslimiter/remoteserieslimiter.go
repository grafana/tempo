package remoteserieslimiter

import (
	"context"
	"flag"
	"os"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/remoteserieslimiter/usagetrackerclient"
	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	UsageTrackerRing          usagetrackerclient.InstanceRingConfig  `yaml:"usage_tracker_ring"`
	UsageTrackerPartitionRing usagetrackerclient.PartitionRingConfig `yaml:"usage_tracker_partition_ring"`
	UsageTrackerClientCfg     usagetrackerclient.Config              `yaml:"usage_tracker_client"`
}

func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.UsageTrackerRing.RegisterFlags(f, log.NewLogfmtLogger(os.Stderr))
	cfg.UsageTrackerPartitionRing.RegisterFlags(f)
	cfg.UsageTrackerClientCfg.RegisterFlags(f)
}

type RemoteSeriesLimiter struct {
	UsageTrackerClient *usagetrackerclient.UsageTrackerClient
}

func NewRemoteSeriesLimiter(cfg *Config, partitionRing *ring.MultiPartitionInstanceRing, instanceRing ring.ReadRing, logger log.Logger, registerer prometheus.Registerer) *RemoteSeriesLimiter {
	client := usagetrackerclient.NewUsageTrackerClient("usage-tracker", cfg.UsageTrackerClientCfg, partitionRing, instanceRing, logger, registerer)

	return &RemoteSeriesLimiter{
		UsageTrackerClient: client,
	}
}

func (r *RemoteSeriesLimiter) ForTenant(tenant string) registry.SeriesLimiter {
	return &tenantSeriesLimiter{
		UsageTrackerClient: r.UsageTrackerClient,
		tenant:             tenant,
	}
}

type tenantSeriesLimiter struct {
	UsageTrackerClient *usagetrackerclient.UsageTrackerClient
	tenant             string
}

// Allow implements registry.SeriesLimiter.
func (r *tenantSeriesLimiter) Allow(hashes []uint64) bool {
	// TODO: this will waste some of the limit, since we may track partially
	// and not allow series to be created. As of now, the only time this can
	// happen is with histograms, and it's better to avoid partially limiting
	// the histogram.
	rejected, err := r.UsageTrackerClient.TrackSeries(context.Background(), r.tenant, hashes)
	if err != nil {
		return false
	}
	return len(rejected) == 0
}

// Remove implements registry.SeriesLimiter.
func (r *tenantSeriesLimiter) Remove(count uint32) {
	// no-op, we rely on the usage tracker expiring series automatically
}
