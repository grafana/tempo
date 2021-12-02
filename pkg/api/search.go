package api

import (
	"net/http"
	"path"
)

// IsBackendSearch returns true if the request has a start, end and tags parameter and is the /api/search path
func IsBackendSearch(r *http.Request) bool {
	q := r.URL.Query()
	return path.Clean(r.URL.Path) == PathSearch &&
		q.Get(URLParamStart) != "" &&
		q.Get(URLParamEnd) != "" &&
		q.Get(urlParamTags) != ""
}
