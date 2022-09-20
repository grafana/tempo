package frontend

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
)

// searchResponse is a thread safe struct used to aggregate the responses from all downstream
// queriers
type searchResponse struct {
	err        error
	statusCode int
	statusMsg  string
	ctx        context.Context

	resultsMap     map[string]*tempopb.TraceSearchMetadata
	resultsMetrics *tempopb.SearchMetrics
	cancelFunc     context.CancelFunc

	limit int
	mtx   sync.Mutex
}

func newSearchResponse(ctx context.Context, limit int, cancelFunc context.CancelFunc) *searchResponse {
	return &searchResponse{
		ctx:            ctx,
		statusCode:     http.StatusOK,
		limit:          limit,
		cancelFunc:     cancelFunc,
		resultsMetrics: &tempopb.SearchMetrics{},
		resultsMap:     map[string]*tempopb.TraceSearchMetadata{},
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
		fmt.Println("---- setStatus cancel", time.Now())
	}
}

func (r *searchResponse) setError(err error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.err = err

	if r.internalShouldQuit() {
		// cancel currently running requests, and bail
		r.cancelFunc()
		fmt.Println("---- setError cancel", time.Now())
	}
}

func (r *searchResponse) addResponse(res *tempopb.SearchResponse) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	for _, t := range res.Traces {
		// todo: determine a better way to combine?
		if _, ok := r.resultsMap[t.TraceID]; !ok {
			r.resultsMap[t.TraceID] = t
		}
	}

	// purposefully ignoring InspectedBlocks as that value is set by the sharder
	r.resultsMetrics.InspectedBytes += res.Metrics.InspectedBytes
	r.resultsMetrics.InspectedTraces += res.Metrics.InspectedTraces
	r.resultsMetrics.SkippedBlocks += res.Metrics.SkippedBlocks
	r.resultsMetrics.SkippedTraces += res.Metrics.SkippedTraces

	fmt.Println("---- add resp", time.Now())

	if r.internalShouldQuit() {
		// cancel currently running requests, and bail
		r.cancelFunc()
		fmt.Println("---- add resp cancel", time.Now())
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
		fmt.Println("---- shouldQuit cancel", time.Now())
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

// make a internalShouldQuit
// call it in each state change (setError, setStatus, addResponse)
// cancel if we need to cancel??
// handle context cancelled errors in setError
//

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
