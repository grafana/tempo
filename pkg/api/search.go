package api

import (
	"net/http"

	"github.com/grafana/tempo/v2/pkg/tempopb"
)

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
