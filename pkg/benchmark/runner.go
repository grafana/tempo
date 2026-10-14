package benchmark

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// Run executes the benchmark cases against the block at blockPath, using the
// trace IDs the profile measured.
func Run(ctx context.Context, blockPath string, prof *profile.BlockProfile, opts RunOptions) (*Result, error) {
	opts.applyDefaults()

	meta, raw, err := openLocalBlock(ctx, blockPath)
	if err != nil {
		return nil, err
	}
	if err := checkProfileMatchesBlock(prof, meta); err != nil {
		return nil, err
	}
	if prof.TraceIDs.Mode == profile.TraceIDModeAll {
		return nil, errors.New(`profile was built with --trace-ids=all, which embeds no IDs; rebuild it with a count`)
	}
	if err := warmBlock(ctx, raw, meta); err != nil {
		return nil, err
	}

	simulated := metrics.NewSimulatedReader(raw, opts.BackendLatency, opts.BackendBandwidth)
	counter := &metrics.CountingReader{RawReader: simulated}
	block, err := encoding.OpenBlock(meta, backend.NewReader(counter))
	if err != nil {
		return nil, fmt.Errorf("opening block: %w", err)
	}

	shards, err := shardsForBlock(meta, opts.TargetBytesPerRequest)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	all := phase1Cases()
	cases := make([]CaseResult, 0, len(all))
	for _, queryCase := range all {
		cases = append(cases, runCase(ctx, block, prof, shards, queryCase, opts, counter, prometheus.DefaultGatherer))
	}

	return &Result{
		SchemaVersion: ResultSchemaVersion,
		StartedAt:     start.UTC(),
		DurationNs:    int64(time.Since(start)),
		RunEnv:        runEnv(),
		Options:       opts,
		Shards:        len(shards),
		Cases:         cases,
	}, nil
}

// checkProfileMatchesBlock refuses a profile built from a different block. Its
// trace IDs would not be in this one, and the result would look like a
// wall-to-wall miss.
func checkProfileMatchesBlock(prof *profile.BlockProfile, meta *backend.BlockMeta) error {
	if prof == nil || prof.Block == nil {
		return errors.New("profile has no block metadata")
	}
	if prof.Block.BlockID != meta.BlockID || prof.Block.TenantID != meta.TenantID {
		return fmt.Errorf("profile is for block %s in tenant %s, but the block here is %s in tenant %s",
			prof.Block.BlockID, prof.Block.TenantID, meta.BlockID, meta.TenantID)
	}
	if prof.Block.Version != meta.Version {
		return fmt.Errorf("profile is for a %s block but this one is %s", prof.Block.Version, meta.Version)
	}
	return nil
}

// runCase measures one query shape. Latency is per execution; everything else
// is a total over the case, because the counters are process-wide.
func runCase(ctx context.Context, block common.BackendBlock, prof *profile.BlockProfile, shards []Shard, queryCase benchCase, opts RunOptions, counter *metrics.CountingReader, gatherer prometheus.Gatherer) CaseResult {
	res := CaseResult{ID: queryCase.id, API: queryCase.api, Query: queryCase.query}

	executions, err := queryCase.executions(prof, shards, opts)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if len(executions) == 0 {
		res.Error = "nothing to execute"
		return res
	}

	collector := metrics.NewCollector(counter, gatherer)
	collector.BeginCase()

	count := 0
	for range opts.Repeat {
		for _, execute := range executions {
			collector.BeginExecution()
			started := time.Now()
			out, err := execute(ctx, block, opts)
			wallNs := int64(time.Since(started))
			if err != nil {
				res.Error = err.Error()
				return res
			}
			collector.EndExecution(wallNs, out.metrics)
			res.Matched += out.matched
			count++
		}
	}

	res.Executions = count
	res.Metrics = collector.EndCase()
	return res
}

func runEnv() RunEnv {
	host, _ := os.Hostname()
	info := profile.CurrentBuildInfo()
	return RunEnv{
		TempoVersion: info.TempoVersion,
		GitSHA:       info.GitSHA,
		GoVersion:    runtime.Version(),
		GoMaxProcs:   runtime.GOMAXPROCS(0),
		Hostname:     host,
	}
}
