// Package jobcost estimates the bytes a query job reads, and calibrates the estimate with the bytes
// that finished jobs actually read.
package jobcost

// Op is the type of query a job belongs to. The values match the query types of the query-frontend.
type Op uint8

const (
	// OpTraceByID is trace by ID, v1 and v2.
	OpTraceByID Op = iota
	// OpSearch is TraceQL search.
	OpSearch
	// OpMetadata is tag and tag value search.
	OpMetadata
	// OpMetrics is TraceQL metrics.
	OpMetrics

	// numOps count the number of Op values
	numOps
)

// String returns the label of the op used on query-frontend metrics.
func (o Op) String() string {
	switch o {
	case OpTraceByID:
		return "traces"
	case OpSearch:
		return "search"
	case OpMetadata:
		return "metadata"
	case OpMetrics:
		return "metrics"
	default:
		return "unknown"
	}
}

// Kind tells how the estimate of a job was made.
type Kind uint8

const (
	// KindNone is the zero value, the job has no estimate.
	KindNone Kind = iota
	// KindBlock jobs read blocks in the backend. Their estimate is Base multiplied by a factor that is
	// calibrated with the bytes finished jobs actually read.
	KindBlock
	// KindLiveStore jobs read recent data from live-stores or generators. Their estimate is a fixed cost.
	KindLiveStore
)

// String returns the label of the kind used on query-frontend metrics.
func (k Kind) String() string {
	switch k {
	case KindBlock:
		return "block"
	case KindLiveStore:
		return "live_store"
	default:
		return "none"
	}
}

// Cost is the estimated bytes a job reads.
type Cost struct {
	Kind Kind

	// Base is what the estimate of a KindBlock job is derived from: the bytes of the block the job covers,
	// or for trace-by-ID the number of blocks it looks into. It is 0 for other kinds.
	Base uint64

	// Estimate is the estimated number of bytes the job reads.
	Estimate uint64
}
