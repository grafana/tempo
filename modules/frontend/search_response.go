package frontend

import (
	"context"
	"net/http"
	"sort"
	"sync"

	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
)

// searchResponse is a thread safe struct used to aggregate the responses from all downstream
// queriers
type searchResponse struct {
	err        error
	statusCode int
	statusMsg  string
	ctx        context.Context

	resultsMap       map[string]*tempopb.TraceSearchMetadata
	resultsMetrics   *tempopb.SearchMetrics
	cancelFunc       context.CancelFunc
	finishedRequests int

	limit int
	mtx   sync.Mutex
}

func newSearchResponse(ctx context.Context, limit int, cancelFunc context.CancelFunc) *searchResponse {
	return &searchResponse{
		ctx:              ctx,
		statusCode:       http.StatusOK,
		limit:            limit,
		cancelFunc:       cancelFunc,
		resultsMetrics:   &tempopb.SearchMetrics{},
		finishedRequests: 0,
		resultsMap:       map[string]*tempopb.TraceSearchMetadata{},
	}
}

func (r *searchResponse) setStatus(statusCode int, statusMsg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.statusCode = statusCode
	r.statusMsg = statusMsg

	if r.internalShouldQuit() {
		// cancel currently running requests, and bail
		r.cancelFunc()
	}
}

func (r *searchResponse) setError(err error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.err = err

	if r.internalShouldQuit() {
		// cancel currently running requests, and bail
		r.cancelFunc()
	}
}

func (r *searchResponse) addResponse(res *tempopb.SearchResponse) {
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

	if r.internalShouldQuit() {
		// cancel currently running requests, and bail
		r.cancelFunc()
	}
}

// shouldQuit locks and checks if we should quit from current execution or not
func (r *searchResponse) shouldQuit() bool {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	quit := r.internalShouldQuit()
	if quit {
		// cancel currently running requests, and bail
		r.cancelFunc()
	}

	return quit
}

// internalShouldQuit check if we should quit but without locking,
// NOTE: only use internally where we already hold lock on searchResponse
func (r *searchResponse) internalShouldQuit() bool {
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

func (r *searchResponse) result() *tempopb.SearchResponse {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	res := &tempopb.SearchResponse{
		Metrics: r.resultsMetrics,
	}

	for _, t := range r.resultsMap {
		res.Traces = append(res.Traces, t)
	}
	sort.Slice(res.Traces, func(i, j int) bool {
		return res.Traces[i].StartTimeUnixNano > res.Traces[j].StartTimeUnixNano
	})

	return res
}
