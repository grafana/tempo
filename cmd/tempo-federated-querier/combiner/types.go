package combiner

import (
	"net/http"

	"github.com/go-kit/log"
)

// QueryResult holds the result from a single Tempo instance
type QueryResult struct {
	Instance string
	Response *http.Response
	Body     []byte
	Error    error
}

// TraceMetrics contains metrics about the trace query
type TraceMetrics struct {
	InstancesQueried   int  `json:"instancesQueried"`
	InstancesResponded int  `json:"instancesResponded"`
	InstancesWithTrace int  `json:"instancesWithTrace"`
	InstancesNotFound  int  `json:"instancesNotFound"`
	InstancesFailed    int  `json:"instancesFailed"`
	TotalSpans         int  `json:"totalSpans"`
	PartialResponse    bool `json:"partialResponse"`
}

// CombineMetadata contains metadata about the combine operation
type CombineMetadata struct {
	InstancesQueried   int
	InstancesResponded int
	InstancesWithTrace int
	InstancesNotFound  int
	InstancesFailed    int
	TotalSpans         int
	PartialResponse    bool
	Errors             []string
}

// SearchMetadata contains metadata about the search combine operation
type SearchMetadata struct {
	InstancesQueried   int
	InstancesResponded int
	InstancesFailed    int
	Errors             []string
}

// Combiner combines results from multiple Tempo instances
type Combiner struct {
	maxSizeBytes int
	logger       log.Logger
}

// New creates a new Combiner
func New(maxSizeBytes int, logger log.Logger) *Combiner {
	return &Combiner{
		maxSizeBytes: maxSizeBytes,
		logger:       logger,
	}
}
