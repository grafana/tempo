package combiner

import (
	"net/http"
)

// Combiner is used to merge multiple responses into a single response.
//
// Implementations must be thread-safe.
type Combiner interface {
	AddRequest(r *http.Response, tenant string) error
	Complete() (*http.Response, error)
	StatusCode() int

	ShouldQuit() bool
}
