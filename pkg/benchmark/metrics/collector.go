package metrics

import (
	"runtime"
	rtmetrics "runtime/metrics"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// Keys for what the harness measures itself. Response and process metrics are
// keyed by Tempo's own names under their own prefix.
const (
	KeyWallNs     = "harness.wallNs"
	KeyCPUNs      = "harness.cpuNs"
	KeyAllocBytes = "harness.allocBytes"
	KeyAllocCount = "harness.allocCount"

	KeyBackendReads  = "backend.reads"
	KeyBackendBytes  = "backend.bytes"
	KeyBackendTimeNs = "backend.timeNs"

	// PrefixResponse marks what a query API reported about itself.
	PrefixResponse = "response."
	// PrefixProcess marks what Tempo emitted to the process registry.
	PrefixProcess = "process."
)

// allocMetrics are cumulative allocation counters read through runtime/metrics
// rather than runtime.ReadMemStats, which stops the world: an STW pause per
// execution would inflate the wall clock measured beside it.
var allocMetrics = []string{
	"/gc/heap/allocs:bytes",
	"/gc/heap/allocs:objects",
}

// Collector gathers one case's measurements.
//
// Every source is read once per execution, so every measurement gets a
// distribution. Reads happen in a fixed order: the execution is timed, then the
// cheap counters are read, then the registry is gathered. A gather costs about
// 90us and 87KB, and it runs after the other deltas are captured and before the
// next execution re-baselines, so it distorts none of them. It does add garbage
// per execution, which in a very long run could pull an extra GC cycle into a
// timed window.
type Collector struct {
	counter  *CountingReader
	gatherer prometheus.Gatherer

	samples map[string][]float64
	kinds   map[string]Kind

	// per-execution baselines
	cpuBefore     int64
	allocBefore   [2]uint64
	readBefore    ReadStats
	processBefore Snapshot

	// processBase is the case baseline, for dropping gauges nothing touched.
	processBase Snapshot

	alloc []rtmetrics.Sample
}

// NewCollector starts collecting for one case. counter and gatherer may each be
// nil, in which case those measurements are skipped.
func NewCollector(counter *CountingReader, gatherer prometheus.Gatherer) *Collector {
	alloc := make([]rtmetrics.Sample, len(allocMetrics))
	for i, name := range allocMetrics {
		alloc[i].Name = name
	}

	return &Collector{
		counter:  counter,
		gatherer: gatherer,
		samples:  map[string][]float64{},
		kinds:    map[string]Kind{},
		alloc:    alloc,
	}
}

// BeginCase clears the heap and takes the case baseline.
//
// The GC is so each case is measured against a comparable environment instead
// of inheriting the warmup's and the previous case's garbage. testing.(*B).runN
// does the same.
func (c *Collector) BeginCase() {
	runtime.GC()
	c.processBase, _ = Gather(c.gatherer)
	c.processBefore = c.processBase
}

// BeginExecution takes the per-execution baselines.
//
// These belong to one execution only because executions run one at a time and
// nothing else in the process is working. Anything concurrent, such as a
// write-back cache's goroutines, would land in whichever execution is in
// flight.
func (c *Collector) BeginExecution() {
	c.cpuBefore = int64(CPUTime())
	if c.counter != nil {
		c.readBefore = c.counter.Snapshot()
	}
	c.allocBefore = c.readAlloc()
}

// EndExecution records what the execution cost, plus what its response and the
// registry reported. wallNs is measured by the caller, outside this call, so
// the cost of collecting is never inside it.
func (c *Collector) EndExecution(wallNs int64, response map[string]int64) {
	// Cheapest first, so the later reads cannot charge themselves to this
	// execution's CPU or allocations.
	allocAfter := c.readAlloc()
	cpuNs := int64(CPUTime()) - c.cpuBefore

	c.observe(Counter, KeyWallNs, float64(wallNs))
	c.observe(Counter, KeyCPUNs, float64(cpuNs))
	c.observe(Counter, KeyAllocBytes, float64(allocAfter[0]-c.allocBefore[0]))
	c.observe(Counter, KeyAllocCount, float64(allocAfter[1]-c.allocBefore[1]))

	if c.counter != nil {
		read := c.counter.Since(c.readBefore)
		c.observe(Counter, KeyBackendReads, float64(read.Reads))
		c.observe(Counter, KeyBackendBytes, float64(read.Bytes))
		c.observe(Counter, KeyBackendTimeNs, float64(read.TimeNs))
	}

	// A metric the API did not report is left out rather than recorded as zero,
	// so a summary's count says how many executions reported it.
	for name, v := range response {
		c.observe(Counter, PrefixResponse+name, float64(v))
	}

	if after, err := Gather(c.gatherer); err == nil {
		after.observeInto(c.processBefore, c)
		c.processBefore = after
	}
}

// EndCase returns every measurement, dropping the ones nothing touched.
func (c *Collector) EndCase() Set {
	out := make(Set, len(c.samples))
	for key, samples := range c.samples {
		kind := c.kinds[key]
		summary := summarize(samples)

		if kind == Gauge {
			// A gauge that sat at its case baseline the whole time was not
			// touched, and the registry holds plenty of those.
			if summary.Min == summary.Max && summary.Max == c.processBase.gauges[strings.TrimPrefix(key, PrefixProcess)] {
				continue
			}
			// A gauge's total is the value it was left at.
			out[key] = Measurement{Kind: Gauge, Total: samples[len(samples)-1], Summary: summary}
			continue
		}

		var total float64
		for _, v := range samples {
			total += v
		}
		// The registry holds hundreds of counters a case never touches, so
		// those are dropped. Everything else was measured, and a measured zero
		// is a result: keeping it is what stops it reading as "not reported".
		if total == 0 && strings.HasPrefix(key, PrefixProcess) {
			continue
		}
		out[key] = Measurement{Kind: Counter, Total: total, Summary: summary}
	}
	return out
}

func (c *Collector) observe(kind Kind, key string, v float64) {
	c.samples[key] = append(c.samples[key], v)
	c.kinds[key] = kind
}

func (c *Collector) readAlloc() [2]uint64 {
	rtmetrics.Read(c.alloc)
	return [2]uint64{c.alloc[0].Value.Uint64(), c.alloc[1].Value.Uint64()}
}
