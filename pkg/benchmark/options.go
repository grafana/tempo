package benchmark

import (
	"time"

	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// DefaultTargetBytesPerRequest matches the query frontend's default, so shards
// match the jobs production would issue.
const DefaultTargetBytesPerRequest = 100 * 1024 * 1024

// DefaultSearchLimit matches the search API's default.
const DefaultSearchLimit = 20

// DefaultMaxSeries caps a metrics query's series the way a request would.
const DefaultMaxSeries = 1000

// A range query steps at max(60s, window/30), which lands about 30 points over
// the block and falls back to 60s for a short window.
const (
	minMetricsStep     = time.Minute
	metricsStepDivisor = 30
)

// RunOptions are the knobs a run holds fixed. They are recorded in the result
// because they are what an experiment varies between its groups.
type RunOptions struct {
	Repeat int `json:"repeat"`
	// Warmup passes run and are discarded before a case is measured. It has no
	// default so 0 stays expressible, but without it the first case pays the
	// block's cold-read cost and every later case runs warm, a bias repetition
	// does not average out.
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

func (opts *RunOptions) applyDefaults() {
	opts.Repeat = max(opts.Repeat, 1)
	if opts.MaxSamples <= 0 {
		opts.MaxSamples = 10000
	}
	if opts.TargetBytesPerRequest <= 0 {
		opts.TargetBytesPerRequest = DefaultTargetBytesPerRequest
	}
	if opts.SearchLimit <= 0 {
		opts.SearchLimit = DefaultSearchLimit
	}
	if opts.MaxSeries <= 0 {
		opts.MaxSeries = DefaultMaxSeries
	}
}

// searchOptions turns the zero-means-default knobs into storage read options.
func (opts RunOptions) searchOptions() common.SearchOptions {
	readOpts := common.DefaultSearchOptions()
	if opts.ReadBufferSize > 0 {
		readOpts.ReadBufferSize = opts.ReadBufferSize
	}
	if opts.ReadBufferCount > 0 {
		readOpts.ReadBufferCount = opts.ReadBufferCount
	}
	if opts.ChunkSizeBytes > 0 {
		readOpts.ChunkSizeBytes = opts.ChunkSizeBytes
	}
	if opts.PrefetchTraceCount > 0 {
		readOpts.PrefetchTraceCount = opts.PrefetchTraceCount
	}
	return readOpts
}
