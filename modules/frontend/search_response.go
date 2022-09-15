package frontend

import (
	"context"
	"net/http"
	"sort"
	"sync"

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

	limit int
	mtx   sync.Mutex
}

func newSearchResponse(ctx context.Context, limit int) *searchResponse {
	return &searchResponse{
		ctx:            ctx,
		statusCode:     http.StatusOK,
		limit:          limit,
		resultsMetrics: &tempopb.SearchMetrics{},
		resultsMap:     map[string]*tempopb.TraceSearchMetadata{},
	}
}

func (r *searchResponse) setStatus(statusCode int, statusMsg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.statusCode = statusCode
	r.statusMsg = statusMsg
}

func (r *searchResponse) setError(err error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.err = err
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
}

func (r *searchResponse) shouldQuit() bool {
	r.mtx.Lock()
	defer r.mtx.Unlock()

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
