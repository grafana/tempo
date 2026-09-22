package benchmark

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/pkg/util"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// DefaultTargetBytesPerRequest matches the query frontend's default, so the
// shards a search is split into match the jobs production would issue.
const DefaultTargetBytesPerRequest = 100 * 1024 * 1024

// DefaultSearchLimit matches the search API's default, so an unfiltered search
// stops where a real one would.
const DefaultSearchLimit = 20

// DefaultMaxSeries caps a metrics query's series the way a request would.
const DefaultMaxSeries = 1000

// minMetricsStep and metricsStepDivisor set the step of a range query:
// max(60s, window/divisor), which targets about 30 points over the block and
// falls back to 60s for a short window.
const (
	minMetricsStep     = time.Minute
	metricsStepDivisor = 30
)

// RunOptions are the knobs a run holds fixed. They are recorded in the result
// because they are what an experiment varies between its groups.
type RunOptions struct {
	Repeat     int `json:"repeat"`
	Warmup     int `json:"warmup"`
	MaxSamples int `json:"maxSamples"`

	TargetBytesPerRequest int `json:"targetBytesPerRequest"`
	SearchLimit           int `json:"searchLimit"`
	MaxSeries             int `json:"maxSeries"`
	Exemplars             int `json:"exemplars"`

	ReadBufferSize     int    `json:"readBufferSize,omitempty"`
	ReadBufferCount    int    `json:"readBufferCount,omitempty"`
	ChunkSizeBytes     uint32 `json:"chunkSizeBytes,omitempty"`
	PrefetchTraceCount int    `json:"prefetchTraceCount,omitempty"`
}

func (o *RunOptions) applyDefaults() {
	o.Repeat = max(o.Repeat, 1)
	if o.MaxSamples <= 0 {
		o.MaxSamples = 10000
	}
	if o.TargetBytesPerRequest <= 0 {
		o.TargetBytesPerRequest = DefaultTargetBytesPerRequest
	}
	if o.SearchLimit <= 0 {
		o.SearchLimit = DefaultSearchLimit
	}
	if o.MaxSeries <= 0 {
		o.MaxSeries = DefaultMaxSeries
	}
}

func (o RunOptions) searchOptions() common.SearchOptions {
	opts := common.DefaultSearchOptions()
	if o.ReadBufferSize > 0 {
		opts.ReadBufferSize = o.ReadBufferSize
	}
	if o.ReadBufferCount > 0 {
		opts.ReadBufferCount = o.ReadBufferCount
	}
	if o.ChunkSizeBytes > 0 {
		opts.ChunkSizeBytes = o.ChunkSizeBytes
	}
	if o.PrefetchTraceCount > 0 {
		opts.PrefetchTraceCount = o.PrefetchTraceCount
	}
	return opts
}

// Run executes the benchmark cases against the block at blockPath, using the
// trace IDs the profile measured.
func Run(ctx context.Context, blockPath string, profile *BlockProfile, o RunOptions) (*Result, error) {
	o.applyDefaults()

	meta, raw, err := openLocalBlock(ctx, blockPath)
	if err != nil {
		return nil, err
	}
	if err := checkProfileMatchesBlock(profile, meta); err != nil {
		return nil, err
	}
	if profile.TraceIDs.Mode == TraceIDModeAll {
		return nil, errors.New(`profile was built with --trace-ids=all, which embeds no IDs; rebuild it with a count`)
	}

	// Every promauto metric in Tempo registers into the default registry at
	// package init, so an in-process run can read them without a scrape
	// endpoint.
	gatherer := prometheus.Gatherer(prometheus.DefaultGatherer)

	counter := &countingReader{RawReader: raw}
	blk, err := encoding.OpenBlock(meta, backend.NewReader(counter))
	if err != nil {
		return nil, fmt.Errorf("opening block: %w", err)
	}

	shards, err := shardsForBlock(meta, profile.RowGroups, o.TargetBytesPerRequest)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	cases := make([]CaseResult, 0, len(phase1Cases()))
	for _, c := range phase1Cases() {
		cases = append(cases, runCase(ctx, blk, profile, shards, c, o, counter, gatherer))
	}

	return &Result{
		SchemaVersion: ResultSchemaVersion,
		StartedAt:     start.UTC(),
		DurationNs:    int64(time.Since(start)),
		RunEnv:        runEnv(),
		Options:       o,
		Cases:         cases,
	}, nil
}

// checkProfileMatchesBlock refuses a profile built from a different block: its
// trace IDs would not be in this one, and the result would look like a
// wall-to-wall miss.
func checkProfileMatchesBlock(profile *BlockProfile, meta *backend.BlockMeta) error {
	if profile == nil || profile.Block == nil {
		return errors.New("profile has no block metadata")
	}
	if profile.Block.BlockID != meta.BlockID || profile.Block.TenantID != meta.TenantID {
		return fmt.Errorf("profile is for block %s in tenant %s, but the block here is %s in tenant %s",
			profile.Block.BlockID, profile.Block.TenantID, meta.BlockID, meta.TenantID)
	}
	if profile.Block.Version != meta.Version {
		return fmt.Errorf("profile is for a %s block but this one is %s", profile.Block.Version, meta.Version)
	}
	return nil
}

// runCase measures one query shape. Latency is per execution; the counters are
// process-wide or too coarse to attribute, so they are totals over the case.
func runCase(ctx context.Context, blk common.BackendBlock, profile *BlockProfile, shards []Shard, c benchCase, o RunOptions, counter *countingReader, gatherer prometheus.Gatherer) CaseResult {
	res := CaseResult{ID: c.id, API: c.api, Query: c.query}

	exec, err := c.executions(profile, shards, o)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if len(exec) == 0 {
		res.Error = "nothing to execute"
		return res
	}

	for range o.Warmup {
		for _, e := range exec {
			if _, err := e(ctx, blk, o); err != nil {
				res.Error = err.Error()
				return res
			}
		}
	}

	var (
		samples    = make([]int64, 0, len(exec)*o.Repeat)
		memBefore  runtime.MemStats
		memAfter   runtime.MemStats
		readBefore = counter.snapshot()
	)
	// A gather failure loses the process metrics for this case but says nothing
	// about the query, so it is not allowed to fail the case.
	promBefore, _ := gatherMetrics(gatherer)
	runtime.ReadMemStats(&memBefore)
	cpuBefore := cpuTime()

	for range o.Repeat {
		for _, e := range exec {
			started := time.Now()
			out, err := e(ctx, blk, o)
			samples = append(samples, int64(time.Since(started)))
			if err != nil {
				res.Error = err.Error()
				return res
			}
			res.Matched += out.matched
			res.Response = addMetrics(res.Response, out.metrics)
		}
	}

	res.CPUNs = int64(cpuTime() - cpuBefore)
	runtime.ReadMemStats(&memAfter)
	res.AllocBytes = int64(memAfter.TotalAlloc - memBefore.TotalAlloc)
	res.AllocCount = int64(memAfter.Mallocs - memBefore.Mallocs)
	res.Backend = counter.since(readBefore)
	if promAfter, err := gatherMetrics(gatherer); err == nil {
		res.Process = promAfter.since(promBefore)
	}

	res.Executions = len(samples)
	res.WallNs = summarize(samples)
	res.Samples = thin(samples, o.MaxSamples)
	return res
}

// thin keeps at most limit samples, at an even stride so the shape of the
// distribution survives.
func thin(samples []int64, limit int) []int64 {
	if len(samples) <= limit {
		return samples
	}
	stride := float64(len(samples)) / float64(limit)
	out := make([]int64, 0, limit)
	for i := range limit {
		out = append(out, samples[int(float64(i)*stride)])
	}
	return out
}

func runEnv() RunEnv {
	host, _ := os.Hostname()
	info := buildInfo()
	return RunEnv{
		TempoVersion: info.TempoVersion,
		GitSHA:       info.GitSHA,
		GoVersion:    runtime.Version(),
		GoMaxProcs:   runtime.GOMAXPROCS(0),
		Hostname:     host,
	}
}

// Shard is a contiguous row-group range, in the units
// common.SearchOptions.StartPage and TotalPages use.
type Shard struct {
	Index      int `json:"index"`
	StartPage  int `json:"startPage"`
	TotalPages int `json:"totalPages"`
}

// shardsForBlock splits the block the way the query frontend splits a search:
// the unit is row groups, but how many go in a shard is decided by a byte
// target. Mirrors modules/frontend.pagesPerRequest.
func shardsForBlock(meta *backend.BlockMeta, rowGroups, targetBytesPerRequest int) ([]Shard, error) {
	pages := pagesPerShard(meta, rowGroups, targetBytesPerRequest)
	if pages <= 0 {
		return nil, fmt.Errorf("block %s: cannot size a shard from %d bytes over %d row groups", meta.BlockID, meta.Size_, rowGroups)
	}

	shards := make([]Shard, 0, (rowGroups+pages-1)/pages)
	for start := 0; start < rowGroups; start += pages {
		shards = append(shards, Shard{
			Index:      len(shards),
			StartPage:  start,
			TotalPages: min(pages, rowGroups-start),
		})
	}
	return shards, nil
}

func pagesPerShard(meta *backend.BlockMeta, rowGroups, targetBytesPerRequest int) int {
	if meta.Size_ == 0 || rowGroups == 0 {
		return 0
	}
	// A block smaller than the target is one job.
	if meta.Size_ < uint64(targetBytesPerRequest) {
		return rowGroups
	}

	bytesPerPage := meta.Size_ / uint64(rowGroups)
	if bytesPerPage == 0 {
		return 0
	}
	return max(targetBytesPerRequest/int(bytesPerPage), 1)
}

// execOutput is what one execution produced, for the totals and the validity
// check.
type execOutput struct {
	matched int64
	// metrics is whatever the response reported, flattened. Nil when the API
	// returned no metrics, which the trace-by-ID miss path does.
	metrics map[string]int64
}

type execution func(context.Context, common.BackendBlock, RunOptions) (execOutput, error)

func traceByIDExecutions(hexIDs []string, opts common.SearchOptions) ([]execution, error) {
	exec := make([]execution, 0, len(hexIDs))
	for _, hexID := range hexIDs {
		id, err := util.HexStringToTraceID(hexID)
		if err != nil {
			return nil, fmt.Errorf("decoding trace ID %q: %w", hexID, err)
		}

		exec = append(exec, func(ctx context.Context, blk common.BackendBlock, _ RunOptions) (execOutput, error) {
			resp, err := blk.FindTraceByID(ctx, id, opts)
			if err != nil {
				return execOutput{}, err
			}
			out := execOutput{}
			if resp == nil {
				// A bloom miss short-circuits before a response is built, so
				// there is nothing to report but the read the counter saw.
				return out, nil
			}
			if resp.Trace != nil {
				out.matched = 1
			}
			if resp.Metrics != nil {
				if out.metrics, err = responseMetrics(resp.Metrics); err != nil {
					return execOutput{}, err
				}
			}
			return out, nil
		})
	}
	return exec, nil
}

// metricsStep is the step of a range query over the block: about 30 points,
// never finer than a minute.
func metricsStep(meta *backend.BlockMeta) time.Duration {
	return max(meta.EndTime.Sub(meta.StartTime)/metricsStepDivisor, minMetricsStep)
}

// metricsExecutions runs a TraceQL metrics query, one execution per shard, over
// the block's whole time range.
//
// instant collapses the window to a single point, which is the only difference
// between an instant query and a range query.
func metricsExecutions(query string, instant bool, shards []Shard, meta *backend.BlockMeta, base common.SearchOptions) []execution {
	var (
		start = uint64(meta.StartTime.UnixNano())
		end   = uint64(meta.EndTime.UnixNano())
	)

	exec := make([]execution, 0, len(shards))
	for _, shard := range shards {
		opts := base
		opts.StartPage, opts.TotalPages = shard.StartPage, shard.TotalPages

		exec = append(exec, func(ctx context.Context, blk common.BackendBlock, o RunOptions) (execOutput, error) {
			req := &tempopb.QueryRangeRequest{
				Query:     query,
				Start:     start,
				End:       end,
				Step:      uint64(metricsStep(meta)),
				MaxSeries: uint32(o.MaxSeries),
				Exemplars: uint32(o.Exemplars),
			}
			if instant {
				req.Step = end - start
				req.XInstant = &tempopb.QueryRangeRequest_Instant{Instant: true}
			}

			eval, err := traceql.NewEngine().CompileMetricsQueryRange(req,
				traceql.WithUnsafeHints(true),
				traceql.WithEngineBytesTracking(true),
			)
			if err != nil {
				return execOutput{}, fmt.Errorf("compiling %q: %w", query, err)
			}

			fetcher := traceql.NewSpansetFetcherWrapperBoth(
				func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
					return blk.Fetch(ctx, req, opts)
				},
				func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
					return blk.FetchSpans(ctx, req, opts)
				},
			)

			if err := eval.Do(ctx, fetcher, start, end, o.MaxSeries); err != nil {
				return execOutput{}, err
			}

			// Results does the final series processing, so it is part of what
			// the query costs and must run inside the measured window.
			results := eval.Results()

			out := execOutput{matched: int64(len(results))}
			if out.metrics, err = evaluatorResponseMetrics(eval.Metrics()); err != nil {
				return execOutput{}, err
			}
			return out, nil
		})
	}
	return exec
}

// tagNamesExecutions lists tag names in one scope, one execution per shard.
//
// This API reports through a callback rather than a metrics message, so bytes
// read is all it can give. The backend counter and the process metrics are what
// cover the rest.
func tagNamesExecutions(scope traceql.AttributeScope, shards []Shard, base common.SearchOptions) []execution {
	exec := make([]execution, 0, len(shards))
	for _, shard := range shards {
		opts := base
		opts.StartPage, opts.TotalPages = shard.StartPage, shard.TotalPages

		exec = append(exec, func(ctx context.Context, blk common.BackendBlock, _ RunOptions) (execOutput, error) {
			var names, bytesRead int64
			err := blk.SearchTags(ctx, scope,
				func(string, traceql.AttributeScope) { names++ },
				func(b uint64) { bytesRead += int64(b) },
				opts,
			)
			if err != nil {
				return execOutput{}, err
			}

			out := execOutput{matched: names}
			if bytesRead > 0 {
				// Keyed as the responses key it, so a reader does not have to
				// know which API produced the row.
				out.metrics = map[string]int64{"inspectedBytes": bytesRead}
			}
			return out, nil
		})
	}
	return exec
}

func searchExecutions(query string, shards []Shard, meta *backend.BlockMeta, base common.SearchOptions) []execution {
	exec := make([]execution, 0, len(shards))
	for _, shard := range shards {
		opts := base
		opts.StartPage, opts.TotalPages = shard.StartPage, shard.TotalPages

		exec = append(exec, func(ctx context.Context, blk common.BackendBlock, o RunOptions) (execOutput, error) {
			fetcher := traceql.NewSpansetFetcherWrapperBoth(
				func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
					return blk.Fetch(ctx, req, opts)
				},
				func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
					return blk.FetchSpans(ctx, req, opts)
				},
			)

			resp, err := traceql.NewEngine().ExecuteSearch(ctx, &tempopb.SearchRequest{
				Query: query,
				Limit: uint32(o.SearchLimit),
				Start: uint32(meta.StartTime.Unix()),
				End:   uint32(meta.EndTime.Unix()),
			}, fetcher)
			if err != nil {
				return execOutput{}, err
			}

			out := execOutput{}
			if resp == nil {
				return out, nil
			}
			out.matched = int64(len(resp.Traces))
			if resp.Metrics != nil {
				if out.metrics, err = responseMetrics(resp.Metrics); err != nil {
					return execOutput{}, err
				}
			}
			return out, nil
		})
	}
	return exec
}
