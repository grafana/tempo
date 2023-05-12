package api

import (
	"net/http"
	"strconv"

	"github.com/grafana/tempo/pkg/tempopb"
)

// IsBackendSearch returns true if the request has a start, end and tags parameter and is the /api/search path
func IsBackendSearch(r *http.Request) bool {
	q := r.URL.Query()
	return q.Get(urlParamStart) != "" && q.Get(urlParamEnd) != ""
}

func IsMissingStartOrEnd(r *http.Request) bool {
	q := r.URL.Query()
	return q.Get(urlParamStart) == "" || q.Get(urlParamEnd) == ""
}

func SetStartAndEnd(r *http.Request, startTime, endTime uint64) {
	q := r.URL.Query()
	q.Set(urlParamStart, strconv.FormatUint(startTime, 10))
	q.Set(urlParamEnd, strconv.FormatUint(endTime, 10))
}

// IsSearchBlock returns true if the request appears to be for backend blocks. It is not exhaustive
// and only looks for blockID
func IsSearchBlock(r *http.Request) bool {
	q := r.URL.Query()

	return q.Get(urlParamBlockID) != ""
}

// IsTraceQLQuery returns true if the request contains a traceQL query.
func IsTraceQLQuery(r *tempopb.SearchRequest) bool {
	return len(r.Query) > 0
}
