package frontend

import (
	"context"
	"net/http"
	"sort"
	"sync"

	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
)

// shardedSearchProgress is an interface that allows us to get progress
// events from the search sharding handler.
type shardedSearchProgress interface {
	init(ctx context.Context, limit, totalJobs, totalBlocks, totalBlockBytes int)

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

	resultsMap       map[string]*tempopb.TraceSearchMetadata
	resultsMetrics   *tempopb.SearchMetrics
	finishedRequests int

	limit int
	mtx   sync.Mutex
}

func newSearchProgress() *searchProgress {
	return &searchProgress{
		resultsMetrics: &tempopb.SearchMetrics{},
	}
}

func (r *searchProgress) init(ctx context.Context, limit, totalJobs, totalBlocks, totalBlockBytes int) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.ctx = ctx
	r.statusCode = http.StatusOK
	r.limit = limit
	r.resultsMetrics = &tempopb.SearchMetrics{
		TotalBlocks:     uint32(totalBlocks),
		TotalBlockBytes: uint64(totalBlockBytes),
		TotalJobs:       uint32(totalJobs),
	}
	r.finishedRequests = 0
	r.resultsMap = map[string]*tempopb.TraceSearchMetadata{}
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
		if _, ok := r.resultsMap[t.TraceID]; !ok {
			r.resultsMap[t.TraceID] = t
		} else {
			// combine into the incoming trace and then set in the map. this prevents
			// race conditions on pointers to traces that we've already returned from
			// .result()
			search.CombineSearchResults(t, r.resultsMap[t.TraceID])
			r.resultsMap[t.TraceID] = t
		}
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
	if len(r.resultsMap) > r.limit {
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
	}

	for _, t := range r.resultsMap {
		searchRes.Traces = append(searchRes.Traces, t)
	}
	sort.Slice(searchRes.Traces, func(i, j int) bool {
		return searchRes.Traces[i].StartTimeUnixNano > searchRes.Traces[j].StartTimeUnixNano
	})

	res.response = searchRes

	return res
}
