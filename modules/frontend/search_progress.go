package frontend

import (
	"context"
	"net/http"
	"sync"

	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

// searchProgressFactory is used to provide a way to construct a shardedSearchProgress to the searchSharder. It exists
// so that streaming search can inject and track it's own special progress object
type searchProgressFactory func(ctx context.Context, limit, totalJobs, totalBlocks int, totalBlockBytes uint64) shardedSearchProgress

// shardedSearchProgress is an interface that allows us to get progress
// events from the search sharding handler.
type shardedSearchProgress interface {
	setStatus(statusCode int, statusMsg string)
	setError(err error)
	addResponse(res *tempopb.SearchResponse)
	shouldQuit() bool
	result() *shardedSearchResults
}

// shardedSearchResults is the overall response from the shardedSearchProgress
type shardedSearchResults struct {
	response         *tempopb.SearchResponse
	statusCode       int
	statusMsg        string
	err              error
	finishedRequests int
}

var _ shardedSearchProgress = (*searchProgress)(nil)

// searchProgress is a thread safe struct used to aggregate the responses from all downstream
// queriers
type searchProgress struct {
	err        error
	statusCode int
	statusMsg  string
	ctx        context.Context

	resultsCombiner  *traceql.MetadataCombiner
	resultsMetrics   *tempopb.SearchMetrics
	finishedRequests int

	limit int
	mtx   sync.Mutex
}

func newSearchProgress(ctx context.Context, limit, totalJobs, totalBlocks int, totalBlockBytes uint64) shardedSearchProgress {
	return &searchProgress{
		ctx:              ctx,
		statusCode:       http.StatusOK,
		limit:            limit,
		finishedRequests: 0,
		resultsMetrics: &tempopb.SearchMetrics{
			TotalBlocks:     uint32(totalBlocks),
			TotalBlockBytes: totalBlockBytes,
			TotalJobs:       uint32(totalJobs),
		},
		resultsCombiner: traceql.NewMetadataCombiner(),
	}
}

func (r *searchProgress) setStatus(statusCode int, statusMsg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.statusCode = statusCode
	r.statusMsg = statusMsg
}

func (r *searchProgress) setError(err error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.err = err
}

func (r *searchProgress) addResponse(res *tempopb.SearchResponse) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	for _, t := range res.Traces {
		r.resultsCombiner.AddMetadata(t)
	}

	// purposefully ignoring TotalBlocks as that value is set by the sharder
	r.resultsMetrics.InspectedBytes += res.Metrics.InspectedBytes
	r.resultsMetrics.InspectedTraces += res.Metrics.InspectedTraces
	r.resultsMetrics.CompletedJobs++

	// count this request as finished
	r.finishedRequests++
}

// shouldQuit locks and checks if we should quit from current execution or not
func (r *searchProgress) shouldQuit() bool {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	return r.internalShouldQuit()
}

// internalShouldQuit check if we should quit but without locking,
// NOTE: only use internally where we already hold lock on searchResponse
func (r *searchProgress) internalShouldQuit() bool {
	if r.err != nil {
		return true
	}
	if r.ctx.Err() != nil {
		return true
	}
	if r.statusCode/100 != 2 {
		return true
	}
	if r.resultsCombiner.Count() >= r.limit {
		return true
	}

	return false
}

func (r *searchProgress) result() *shardedSearchResults {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	res := &shardedSearchResults{
		statusCode:       r.statusCode,
		statusMsg:        r.statusMsg,
		err:              r.err,
		finishedRequests: r.finishedRequests,
	}

	// copy metadata b/c the resultsCombiner holds a pointer to the data and continues
	// to modify it. this may race with anything getting results
	md := r.resultsCombiner.Metadata()
	mdCopy := make([]*tempopb.TraceSearchMetadata, 0, len(md))
	for _, m := range md {
		mCopy := &tempopb.TraceSearchMetadata{
			TraceID:           m.TraceID,
			RootServiceName:   m.RootServiceName,
			RootTraceName:     m.RootTraceName,
			StartTimeUnixNano: m.StartTimeUnixNano,
			DurationMs:        m.DurationMs,
			SpanSet:           copySpanset(m.SpanSet),
		}

		// copy service stats
		if len(m.ServiceStats) > 0 {
			mCopy.ServiceStats = make(map[string]*tempopb.ServiceStats, len(m.ServiceStats))
			for service, stat := range m.ServiceStats {
				mCopy.ServiceStats[service] = &tempopb.ServiceStats{
					SpanCount:  stat.SpanCount,
					ErrorCount: stat.ErrorCount,
				}
			}
		}

		// now copy spansets
		if len(m.SpanSets) > 0 {
			mCopy.SpanSets = make([]*tempopb.SpanSet, 0, len(m.SpanSets))
			for _, ss := range m.SpanSets {
				mCopy.SpanSets = append(mCopy.SpanSets, copySpanset(ss))
			}
		}

		// substitute empty root span
		if mCopy.RootServiceName == "" {
			mCopy.RootServiceName = search.RootSpanNotYetReceivedText
		}

		mdCopy = append(mdCopy, mCopy)
	}

	searchRes := &tempopb.SearchResponse{
		// clone search metrics to avoid race conditions on the pointer
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: r.resultsMetrics.InspectedTraces,
			InspectedBytes:  r.resultsMetrics.InspectedBytes,
			TotalBlocks:     r.resultsMetrics.TotalBlocks,
			CompletedJobs:   r.resultsMetrics.CompletedJobs,
			TotalJobs:       r.resultsMetrics.TotalJobs,
			TotalBlockBytes: r.resultsMetrics.TotalBlockBytes,
		},
		Traces: mdCopy,
	}

	res.response = searchRes

	return res
}

func copySpanset(ss *tempopb.SpanSet) *tempopb.SpanSet {
	if ss == nil {
		return nil
	}

	// the metadata results combiner considers the spans and attributes immutable. it does not attempt to change them
	// so just copying the slices should be safe
	return &tempopb.SpanSet{
		Spans:      append([]*tempopb.Span(nil), ss.Spans...),
		Matched:    ss.Matched,
		Attributes: append([]*v1.KeyValue(nil), ss.Attributes...),
	}
}
