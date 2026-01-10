package combiner

import (
	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
)

// InstanceResult is a generic result from a single Tempo instance
type InstanceResult[T any] struct {
	Instance string
	Response T
	Error    error
	NotFound bool // True if instance returned 404
}

// TraceResult holds a trace result from a single instance
type TraceResult = InstanceResult[*tempopb.Trace]

// TraceByIDResult holds a TraceByID v2 API result from a single instance
type TraceByIDResult = InstanceResult[*tempopb.TraceByIDResponse]

// SearchResult holds a search result from a single instance
type SearchResult = InstanceResult[*tempopb.SearchResponse]

// SearchTagsResult holds a tags result from a single instance
type SearchTagsResult = InstanceResult[*tempopb.SearchTagsResponse]

// SearchTagsV2Result holds a tags v2 result from a single instance
type SearchTagsV2Result = InstanceResult[*tempopb.SearchTagsV2Response]

// SearchTagValuesResult holds a tag values result from a single instance
type SearchTagValuesResult = InstanceResult[*tempopb.SearchTagValuesResponse]

// SearchTagValuesV2Result holds a tag values v2 result from a single instance
type SearchTagValuesV2Result = InstanceResult[*tempopb.SearchTagValuesV2Response]

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

