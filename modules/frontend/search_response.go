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

func newSearchResponse(ctx context.Context, limit, _, totalBlocks, totalBlockBytes int) shardedSearchProgress {
	return &searchProgress{
		ctx:        ctx,
		statusCode: http.StatusOK,
		limit:      limit,
		resultsMetrics: &tempopb.SearchMetrics{
			InspectedBlocks: uint32(totalBlocks),
			TotalBlockBytes: uint64(totalBlockBytes),
		},
		finishedRequests: 0,
		resultsMap:       map[string]*tempopb.TraceSearchMetadata{},
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
		if _, ok := r.resultsMap[t.TraceID]; !ok {
			r.resultsMap[t.TraceID] = t
		} else {
			search.CombineSearchResults(r.resultsMap[t.TraceID], t)
		}
	}

	// purposefully ignoring InspectedBlocks as that value is set by the sharder
	r.resultsMetrics.InspectedBytes += res.Metrics.InspectedBytes
	r.resultsMetrics.InspectedTraces += res.Metrics.InspectedTraces
	r.resultsMetrics.SkippedBlocks += res.Metrics.SkippedBlocks
	r.resultsMetrics.SkippedTraces += res.Metrics.SkippedTraces

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
		Metrics: r.resultsMetrics,
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
