package api

import (
	"net/http"
)

// IsBackendSearch returns true if the request has a start, end and tags parameter and is the /api/search path
func IsBackendSearch(r *http.Request) bool {
	q := r.URL.Query()
	return q.Get(urlParamStart) != "" && q.Get(urlParamEnd) != ""
}

// IsSearchBlock returns true if the request appears to be for backend blocks. It is not exhaustive
// and only looks for blockID
func IsSearchBlock(r *http.Request) bool {
	q := r.URL.Query()

	return q.Get(urlParamBlockID) != ""
}
