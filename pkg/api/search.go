package api

import (
	"net/http"
	"path"
	"strings"
)

// IsBackendSearch returns true if the request has a start, end and tags parameter and is the /api/search path
func IsBackendSearch(r *http.Request) bool {
	q := r.URL.Query()
	return strings.HasSuffix(path.Clean(r.URL.Path), PathSearch) &&
		q.Get(urlParamStart) != "" &&
		q.Get(urlParamEnd) != "" &&
		q.Get(urlParamTags) != ""
}

// IsSearchBlock returns true if the request appears to be for backend blocks. It is not exhaustive
// and only looks for blockID
func IsSearchBlock(r *http.Request) bool {
	q := r.URL.Query()

	return strings.HasSuffix(path.Clean(r.URL.Path), PathSearch) &&
		q.Get(urlParamBlockID) != ""
}
