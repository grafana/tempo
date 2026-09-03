package spanmetrics

import (
	"context"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/sharedconfig"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// newTestCounter stands in for the per-tenant discard counters the generator
// passes the processor; these tests only care that they are non-nil.
func newTestCounter() prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{})
}

// testSendInterval matches registry.Config's default CollectionInterval, which
// is the unit the sampler's budget is counted in.
const testSendInterval = 15 * time.Second

// newTestSampler builds a sampler whose budget is stated per send interval, as
// production states it, with reproducible block picks.
func newTestSampler(budgetPerInterval int, seed uint64, share func() float64) *seriesSampler {
	s := newSeriesSampler(budgetPerInterval, testSendInterval, share)
	seedSampler(s, seed)
	return s
}

// seedSampler makes the block pick reproducible. Every shard gets the same
// stream, which is fine because these tests exercise one key at a time.
func seedSampler(s *seriesSampler, seed uint64) {
	for i := range s.shards {
		s.shards[i].rng = rand.New(rand.NewPCG(seed, seed+1))
	}
}

// samplerRun is the outcome of feeding one key a steady span rate.
type samplerRun struct {
	spans    int
	kept     int
	estimate float64
	// positions[i] counts how often the span at offset i of a block was the one
	// kept, over the blocks observed while the block size held steady.
	positions       []int
	steadyBlockSize uint64
	endMs           int64
}

// feedSampler pushes spans for one key at spansPerSecond for the given
// duration, advancing the clock in step so sampler windows roll the way they do
// in production, and reports what came back.
func feedSampler(s *seriesSampler, key uint64, startMs int64, spansPerSecond int, d time.Duration) samplerRun {
	spans := int(d.Seconds() * float64(spansPerSecond))
	run := samplerRun{spans: spans, endMs: startMs}

	series := func() *samplerSeries {
		sh := &s.shards[key&samplerShardMsk]
		sh.mtx.Lock()
		defer sh.mtx.Unlock()
		return sh.series[key]
	}

	for i := 0; i < spans; i++ {
		// Spread the spans evenly across the run in whole milliseconds.
		run.endMs = startMs + int64(float64(i)/float64(spansPerSecond)*1000)

		var pos, blockSize uint64
		if existing := series(); existing != nil {
			pos, blockSize = existing.pos, existing.blockSize
		}

		multiplier := s.sample(key, run.endMs)
		if multiplier != 0 {
			run.kept++
			run.estimate += multiplier
			// Only record positions once the block size has settled, so a
			// warming-up run does not look like a biased pick.
			if blockSize == run.steadyBlockSize && blockSize > 0 {
				run.positions[pos]++
			}
		}
		if blockSize != run.steadyBlockSize && blockSize > 0 {
			run.steadyBlockSize = blockSize
			run.positions = make([]int, blockSize)
		}
	}
	return run
}

func TestSeriesSamplerKeepsSeriesUnderBudget(t *testing.T) {
	s := newTestSampler(1_500, 1, nil)

	// 50 spans/s against a 1500-per-15s (100/s) budget: never worth sampling, so every
	// span must come back with a multiplier of exactly 1.
	start := time.Unix(1_700_000_000, 0).UnixMilli()
	for i := 0; i < 5_000; i++ {
		require.Equal(t, 1.0, s.sample(1, start+int64(i)*20), "span %d was sampled", i)
	}

	seen, kept := s.counts()
	require.Equal(t, uint64(5_000), seen)
	require.Equal(t, uint64(5_000), kept)
}

func TestSeriesSamplerCountIsExactWithinOneBlock(t *testing.T) {
	// The scheme's central guarantee: every completed block consumes blockSize
	// spans and contributes exactly blockSize to the estimate, so the only error
	// is the block still in flight.
	for _, tc := range []struct {
		name              string
		budgetPerInterval int
		spansPerSecond    int
	}{
		{"under budget", 1_500, 50},
		{"10x over budget", 1_500, 1_000},
		{"100x over budget", 1_500, 10_000},
		{"1000x over budget", 150, 10_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestSampler(tc.budgetPerInterval, 7, nil)

			const duration = 2 * time.Minute
			run := feedSampler(s, 1, time.Unix(1_700_000_000, 0).UnixMilli(), tc.spansPerSecond, duration)

			blockSize := currentBlockSize(t, s, 1)
			require.LessOrEqual(t, math.Abs(run.estimate-float64(run.spans)), float64(blockSize),
				"count estimate %v is more than one block (%d) away from %d spans", run.estimate, blockSize, run.spans)

			// The budget has to actually bite, otherwise the bound above is
			// trivially satisfied by keeping everything. Allow generous slack for
			// the logarithmic overshoot of the first window.
			wantKept := float64(tc.budgetPerInterval) * duration.Seconds() / testSendInterval.Seconds()
			if float64(tc.spansPerSecond) > float64(tc.budgetPerInterval)/testSendInterval.Seconds() {
				require.Less(t, float64(run.kept), 3*wantKept,
					"kept %d spans, budget over this run is %v", run.kept, wantKept)
			}
		})
	}
}

func TestSeriesSamplerKeepsExactlyOnePerBlock(t *testing.T) {
	s := newTestSampler(1_500, 3, nil)

	// Warm up to a steady block size, then walk whole blocks and check each one
	// yields a single span scaled by exactly that block's size.
	start := time.Unix(1_700_000_000, 0).UnixMilli()
	run := feedSampler(s, 1, start, 10_000, 2*time.Minute)
	nowMs := run.endMs

	series := s.shards[1&samplerShardMsk].series[1]
	require.NotNil(t, series)
	require.Greater(t, series.blockSize, uint64(1), "rate never pushed the block size above 1")

	// Finish the block in flight so the walk starts on a boundary.
	for series.pos != 0 {
		s.sample(1, nowMs)
	}

	for block := 0; block < 50; block++ {
		blockSize := series.blockSize
		keptInBlock := 0
		multiplierSum := 0.0
		for i := uint64(0); i < blockSize; i++ {
			if multiplier := s.sample(1, nowMs); multiplier != 0 {
				keptInBlock++
				multiplierSum += multiplier
			}
		}
		require.Equal(t, 1, keptInBlock, "block %d kept %d spans", block, keptInBlock)
		require.Equal(t, float64(blockSize), multiplierSum, "block %d scaled to the wrong total", block)
	}
}

func TestSeriesSamplerPickIsUniformWithinBlock(t *testing.T) {
	// Always picking a fixed position inside the block would make the estimates
	// wrong whenever arrival order correlates with latency, so check the picks
	// spread across the block.
	s := newTestSampler(1_500, 11, nil)

	run := feedSampler(s, 1, time.Unix(1_700_000_000, 0).UnixMilli(), 10_000, 30*time.Minute)
	require.Greater(t, run.steadyBlockSize, uint64(4), "block too small to say anything about the spread")

	total := 0
	for _, count := range run.positions {
		total += count
	}
	require.Greater(t, total, 20*len(run.positions), "too few blocks observed at a steady size")

	expected := float64(total) / float64(len(run.positions))
	for i, count := range run.positions {
		require.InDelta(t, expected, float64(count), 0.5*expected,
			"position %d of %d was picked %d times, expected about %v", i, len(run.positions), count, expected)
	}
}

func TestSeriesSamplerAdaptsWhenRateDrops(t *testing.T) {
	s := newTestSampler(1_500, 5, nil)

	run := feedSampler(s, 1, time.Unix(1_700_000_000, 0).UnixMilli(), 10_000, time.Minute)
	require.Greater(t, currentBlockSize(t, s, 1), uint64(1))

	// The series goes quiet, then comes back well under budget. A couple of
	// windows of low-rate traffic bring the block size back to 1.
	quietMs := run.endMs + 10*testSendInterval.Milliseconds()
	back := feedSampler(s, 1, quietMs, 10, 30*time.Second)

	require.Equal(t, uint64(1), currentBlockSize(t, s, 1))
	require.Equal(t, 1.0, s.sample(1, back.endMs+100))
}

func TestSeriesSamplerBurstGuard(t *testing.T) {
	// A brand new series has no rate estimate, so it starts at block size 1. A
	// burst inside that first window must neither cost a full-price aggregation
	// per span nor blow up the block size past what the volume implies.
	s := newTestSampler(150, 13, nil)

	const spans = 1_000_000
	nowMs := time.Unix(1_700_000_000, 0).UnixMilli()
	estimate := 0.0
	for i := 0; i < spans; i++ {
		estimate += s.sample(1, nowMs)
	}

	budget := uint64(150)
	_, kept := s.counts()

	// budget*(1+ln(spans/budget)) is what the seen/budget block sizing lets
	// through; check the same order of magnitude rather than the exact figure.
	wantKept := float64(budget) * (1 + math.Log(float64(spans)/float64(budget)))
	require.Less(t, float64(kept), 2*wantKept, "burst guard let %d of %d spans through", kept, spans)

	// The block, and so the count error, stayed bounded by spans/budget.
	blockSize := currentBlockSize(t, s, 1)
	require.LessOrEqual(t, blockSize, uint64(spans)/budget)
	require.LessOrEqual(t, math.Abs(estimate-float64(spans)), float64(blockSize))
}

func currentBlockSize(t *testing.T, s *seriesSampler, key uint64) uint64 {
	t.Helper()
	sh := &s.shards[key&samplerShardMsk]
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	series := sh.series[key]
	require.NotNil(t, series)
	return series.blockSize
}

func TestSeriesSamplerSplitsBudgetAcrossReplicas(t *testing.T) {
	// The budget is fleet-wide. Every generator emits its own copy of a series
	// and a query sums them, so N instances each keeping the whole budget would
	// hand the query N budgets. Each instance takes the share of the budget
	// matching the share of the spans it sees, and those shares sum to 1.
	const (
		budgetPerInterval = 1_500
		replicas          = 10
		fleetSpansPerSec  = 20_000
		// Long enough that each replica's first window, which overshoots
		// logarithmically because no rate estimate exists yet, amortizes away.
		duration = 10 * time.Minute
	)

	fleetKept := 0.0
	fleetEstimate := 0.0
	for replica := 0; replica < replicas; replica++ {
		s := newTestSampler(budgetPerInterval, uint64(replica)+1, func() float64 { return 1.0 / replicas })
		run := feedSampler(s, 1, time.Unix(1_700_000_000, 0).UnixMilli(), fleetSpansPerSec/replicas, duration)
		fleetKept += float64(run.kept)
		fleetEstimate += run.estimate
	}

	intervals := duration.Seconds() / testSendInterval.Seconds()
	wantKept := budgetPerInterval * intervals
	require.InDelta(t, wantKept, fleetKept, 0.5*wantKept,
		"fleet kept %v spans, the fleet-wide budget over this run is %v", fleetKept, wantKept)
	// The point of the split: without it each replica would keep the whole
	// budget and the fleet would keep replicas times too many.
	require.Less(t, fleetKept, float64(replicas)*wantKept/2)

	// Summing the replicas' series still recovers the fleet's span count, which
	// is what a query aggregating over __metrics_gen_instance computes.
	wantSpans := fleetSpansPerSec * duration.Seconds()
	require.InDelta(t, wantSpans, fleetEstimate, 0.001*wantSpans)
}

func TestSeriesSamplerIgnoresUnusableShare(t *testing.T) {
	// A share the deployment could not work out must leave the budget whole:
	// under-sampling costs CPU, over-sampling would quietly cost accuracy.
	for _, share := range []float64{0, -1, 1.5, math.NaN()} {
		s := newTestSampler(1_500, 3, func() float64 { return share })
		s.shards[1&samplerShardMsk].mtx.Lock()
		budget := s.shards[1&samplerShardMsk].budgetFor(s, 0)
		s.shards[1&samplerShardMsk].mtx.Unlock()
		require.Equal(t, uint64(1_500), budget, "share %v", share)
	}
}

func TestSeriesSamplerDisabled(t *testing.T) {
	require.Nil(t, newSeriesSampler(0, testSendInterval, nil))
	require.Nil(t, newSeriesSampler(-1, testSendInterval, nil))
}

// TestSpanMetricsSamplingPreservesTotals drives the whole processor and checks
// that the series the sampler cut still reports the totals it would have
// without sampling. Every span carries the same latency and size, which makes
// _sum and size_total scale with the count exactly, so any scaling that the
// implementation forgot to apply shows up as a hard mismatch rather than as
// statistical noise.
func TestSpanMetricsSamplingPreservesTotals(t *testing.T) {
	const (
		// 100 spans per push at one push per millisecond is 100k spans/s for the
		// single series this fixture maps to. Running for 40 simulated seconds
		// rolls several sampler windows, so the totals below are checked against
		// the steady state rather than the first window's warm-up.
		spansPerPush      = 100
		pushes            = 40_000
		spans             = spansPerPush * pushes
		latencySecs       = 0.75
		budgetPerInterval = 1_500
	)

	testRegistry := registry.NewTestRegistry()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.MaxSpansPerSeriesPerInterval = budgetPerInterval
	cfg.SendInterval = testSendInterval

	p, err := New(cfg, testRegistry, newTestCounter(), newTestCounter())
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	processor := p.(*Processor)
	seedSampler(processor.sampler, 17)

	now := time.Unix(1_700_000_000, 0)
	processor.now = func() time.Time { return now }

	request := samplingTestRequest(spansPerPush, latencySecs)
	spanSize := float64(request.Batches[0].ScopeSpans[0].Spans[0].Size())

	for push := 0; push < pushes; push++ {
		now = now.Add(time.Millisecond)
		p.PushSpans(context.Background(), request)
	}

	lbls := labels.FromMap(map[string]string{
		"service":     "svc",
		"span_name":   "GET /api",
		"span_kind":   "SPAN_KIND_SERVER",
		"status_code": "STATUS_CODE_OK",
	})

	seen, kept := processor.sampler.counts()
	require.Equal(t, uint64(spans), seen)
	require.Less(t, kept, uint64(spans/100), "sampling barely bit: kept %d of %d spans", kept, spans)

	blockSize, seriesCount := onlySampledSeries(t, processor.sampler)
	require.Equal(t, 1, seriesCount, "the fixture should map to exactly one series")
	require.Greater(t, blockSize, 1.0)

	// One block of slack, the bound the scheme guarantees.
	assert.InDelta(t, float64(spans), testRegistry.Query("traces_spanmetrics_calls_total", lbls), blockSize)
	assert.InDelta(t, float64(spans), testRegistry.Query("traces_spanmetrics_latency_count", lbls), blockSize)
	assert.InDelta(t, float64(spans)*latencySecs, testRegistry.Query("traces_spanmetrics_latency_sum", lbls), blockSize*latencySecs)
	assert.InDelta(t, float64(spans)*spanSize, testRegistry.Query("traces_spanmetrics_size_total", lbls), blockSize*spanSize)

	// The latency lands in the (0.5, 1] bucket, so the cumulative counts are 0,
	// spans, spans.
	assert.Equal(t, 0.0, testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 0.5)))
	assert.InDelta(t, float64(spans), testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, 1)), blockSize)
	assert.InDelta(t, float64(spans), testRegistry.Query("traces_spanmetrics_latency_bucket", withLe(lbls, math.Inf(1))), blockSize)
}

// TestSpanMetricsSamplingComposesWithSpanMultiplier checks the sampling
// multiplier stacks with the configured span multiplier rather than replacing
// it: a tenant sampling at the collector and again here must see both factors.
func TestSpanMetricsSamplingComposesWithSpanMultiplier(t *testing.T) {
	const (
		spansPerPush = 100
		pushes       = 20_000
		spans        = spansPerPush * pushes
		sampleRate   = 0.25
	)

	testRegistry := registry.NewTestRegistry()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.HistogramBuckets = []float64{0.5, 1}
	cfg.MaxSpansPerSeriesPerInterval = 1_500
	cfg.SendInterval = testSendInterval
	cfg.SpanMultiplierKey = "sample.rate"

	p, err := New(cfg, testRegistry, newTestCounter(), newTestCounter())
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	processor := p.(*Processor)
	seedSampler(processor.sampler, 19)
	now := time.Unix(1_700_000_000, 0)
	processor.now = func() time.Time { return now }

	request := samplingTestRequest(spansPerPush, 0.75)
	for _, span := range request.Batches[0].ScopeSpans[0].Spans {
		span.Attributes = append(span.Attributes, &common_v1.KeyValue{
			Key:   "sample.rate",
			Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_DoubleValue{DoubleValue: sampleRate}},
		})
	}

	for push := 0; push < pushes; push++ {
		now = now.Add(time.Millisecond)
		p.PushSpans(context.Background(), request)
	}

	lbls := labels.FromMap(map[string]string{
		"service":     "svc",
		"span_name":   "GET /api",
		"span_kind":   "SPAN_KIND_SERVER",
		"status_code": "STATUS_CODE_OK",
	})
	// GetSpanMultiplier reads the attribute as a rate, so each span counts for
	// 1/sampleRate spans upstream.
	want := float64(spans) / sampleRate
	assert.InDelta(t, want, testRegistry.Query("traces_spanmetrics_calls_total", lbls), 0.01*want)
}

// TestSpanMetricsSamplingKeyTracksLabelSet pins the sampling key to the same
// values the label set is built from. A key that splits one series into several
// hands it several budgets and wastes the CPU sampling was meant to save; a key
// that merges series makes them share one budget and undershoots the accuracy
// target.
func TestSpanMetricsSamplingKeyTracksLabelSet(t *testing.T) {
	newProcessor := func(t *testing.T, tune func(*Config)) *Processor {
		cfg := Config{}
		cfg.RegisterFlagsAndApplyDefaults("", nil)
		cfg.MaxSpansPerSeriesPerInterval = 1_500
		cfg.SendInterval = testSendInterval
		if tune != nil {
			tune(&cfg)
		}
		p, err := New(cfg, registry.NewTestRegistry(), newTestCounter(), newTestCounter())
		require.NoError(t, err)
		t.Cleanup(func() { p.Shutdown(context.Background()) })
		return p.(*Processor)
	}

	keyOf := func(p *Processor, rs *resource_v1.Resource, span *trace_v1.Span) uint64 {
		scratch := p.scratchPool.Get().(*spanScratch)
		defer p.scratchPool.Put(scratch)
		// Reuse sampleSpan's key construction without touching sampler state.
		p.buildSamplerKey(scratch, "svc", "", "", rs, span)
		return scratch.key.sum()
	}

	baseSpan := func() *trace_v1.Span {
		return &trace_v1.Span{
			Name:              "GET /api",
			Kind:              trace_v1.Span_SPAN_KIND_SERVER,
			StartTimeUnixNano: 0,
			EndTimeUnixNano:   uint64(time.Second),
			Status:            &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_OK},
			Attributes: []*common_v1.KeyValue{
				samplingStringAttr("http.method", "GET"),
			},
		}
	}
	resource := &resource_v1.Resource{Attributes: []*common_v1.KeyValue{
		samplingStringAttr("service.name", "svc"),
	}}

	t.Run("attributes outside the label set do not split a series", func(t *testing.T) {
		p := newProcessor(t, nil)
		a, b := baseSpan(), baseSpan()
		b.Attributes = append(b.Attributes, samplingStringAttr("http.url", "/api/12345"))
		require.Equal(t, keyOf(p, resource, a), keyOf(p, resource, b))
	})

	t.Run("span name splits a series", func(t *testing.T) {
		p := newProcessor(t, nil)
		a, b := baseSpan(), baseSpan()
		b.Name = "GET /other"
		require.NotEqual(t, keyOf(p, resource, a), keyOf(p, resource, b))
	})

	t.Run("span name does not split a series once the dimension is off", func(t *testing.T) {
		p := newProcessor(t, func(cfg *Config) { cfg.IntrinsicDimensions.SpanName = false })
		a, b := baseSpan(), baseSpan()
		b.Name = "GET /other"
		require.Equal(t, keyOf(p, resource, a), keyOf(p, resource, b))
	})

	t.Run("status code splits a series", func(t *testing.T) {
		p := newProcessor(t, nil)
		a, b := baseSpan(), baseSpan()
		b.Status = &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_ERROR}
		require.NotEqual(t, keyOf(p, resource, a), keyOf(p, resource, b))
	})

	t.Run("configured dimension splits a series", func(t *testing.T) {
		p := newProcessor(t, func(cfg *Config) { cfg.Dimensions = []string{"http.method"} })
		a, b := baseSpan(), baseSpan()
		b.Attributes = []*common_v1.KeyValue{samplingStringAttr("http.method", "POST")}
		require.NotEqual(t, keyOf(p, resource, a), keyOf(p, resource, b))
	})

	t.Run("dimension mapping source splits a series", func(t *testing.T) {
		p := newProcessor(t, func(cfg *Config) {
			cfg.DimensionMappings = []sharedconfig.DimensionMappings{
				{Name: "route_key", SourceLabel: []string{"http.method", "http.route"}, Join: ":"},
			}
		})
		a, b := baseSpan(), baseSpan()
		a.Attributes = append(a.Attributes, samplingStringAttr("http.route", "/api/:id"))
		b.Attributes = append(b.Attributes, samplingStringAttr("http.route", "/other/:id"))
		require.NotEqual(t, keyOf(p, resource, a), keyOf(p, resource, b))
	})

	t.Run("job and instance split a series only when target_info is on", func(t *testing.T) {
		withJob := func(p *Processor, job, instance string) uint64 {
			scratch := p.scratchPool.Get().(*spanScratch)
			defer p.scratchPool.Put(scratch)
			p.buildSamplerKey(scratch, "svc", job, instance, resource, baseSpan())
			return scratch.key.sum()
		}

		off := newProcessor(t, nil)
		require.Equal(t, withJob(off, "ns/svc", "instance-1"), withJob(off, "other/svc", "instance-2"))

		on := newProcessor(t, func(cfg *Config) { cfg.EnableTargetInfo = true })
		require.NotEqual(t, withJob(on, "ns/svc", "instance-1"), withJob(on, "other/svc", "instance-1"))
		require.NotEqual(t, withJob(on, "ns/svc", "instance-1"), withJob(on, "ns/svc", "instance-2"))
	})
}

// onlySampledSeries reports the block size of the sampler's single tracked
// series along with how many it is tracking, so a test can assert both.
func onlySampledSeries(t *testing.T, s *seriesSampler) (blockSize float64, count int) {
	t.Helper()
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mtx.Lock()
		for _, series := range sh.series {
			blockSize = float64(series.blockSize)
			count++
		}
		sh.mtx.Unlock()
	}
	return blockSize, count
}

func samplingTestRequest(spans int, latencySecs float64) *tempopb.PushSpansRequest {
	rs := &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{Attributes: []*common_v1.KeyValue{
			samplingStringAttr("service.name", "svc"),
		}},
		ScopeSpans: []*trace_v1.ScopeSpans{{}},
	}
	for i := 0; i < spans; i++ {
		rs.ScopeSpans[0].Spans = append(rs.ScopeSpans[0].Spans, &trace_v1.Span{
			Name:              "GET /api",
			Kind:              trace_v1.Span_SPAN_KIND_SERVER,
			StartTimeUnixNano: 0,
			EndTimeUnixNano:   uint64(latencySecs * float64(time.Second)),
			Status:            &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_OK},
			Attributes: []*common_v1.KeyValue{
				samplingStringAttr("http.method", "GET"),
			},
		})
	}
	return &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{rs}}
}

func samplingStringAttr(key, value string) *common_v1.KeyValue {
	return &common_v1.KeyValue{
		Key:   key,
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: value}},
	}
}
