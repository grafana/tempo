package combiner

import (
	"context"
	"net/http"
)

// Combiner is used to merge multiple responses into a single response.
//
// Implementations must be thread-safe.
// TODO: StatusCode() is only used for multi-tenant support. Can we remove it?
type Combiner interface {
	AddResponse(r PipelineResponse) error
	StatusCode() int
	ShouldQuit() bool

	// returns the final/complete results
	HTTPFinal(ctx context.Context) (*http.Response, error)
}

type TypedCombiner[T TResponse] interface {
	AddTypedResponse(r T) error
}

type GRPCCombiner[T TResponse] interface {
	Combiner
	TypedCombiner[T]

	GRPCFinal(ctx context.Context) (T, error)
	GRPCDiff() (T, error)
	GRPCSegment(response T, maxSize int) ([]T, error)
}
