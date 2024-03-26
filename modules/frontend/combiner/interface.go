package combiner

import (
	"net/http"
)

// Combiner is used to merge multiple responses into a single response.
//
// Implementations must be thread-safe.
// TODO: StatusCode() and the tenant parameter on AddRequest are only used in for multi-tenant support. Can we remove them?
type Combiner interface {
	AddResponse(r *http.Response) error
	StatusCode() int
	ShouldQuit() bool

	// returns the final/complete results
	HTTPFinal() (*http.Response, error)
}

type GRPCCombiner[T TResponse] interface {
	Combiner

	GRPCFinal() (T, error)
	GRPCDiff() (T, error)
}
