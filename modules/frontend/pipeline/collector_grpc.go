package pipeline

import (
	"context"
	"net/http"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"google.golang.org/grpc/codes"
)

type GRPCCollector[T combiner.TResponse] struct {
	next     AsyncRoundTripper[combiner.PipelineResponse]
	combiner combiner.GRPCCombiner[T]

	send func(T) error
}

func NewGRPCCollector[T combiner.TResponse](next AsyncRoundTripper[combiner.PipelineResponse], combiner combiner.GRPCCombiner[T], send func(T) error) *GRPCCollector[T] {
	return &GRPCCollector[T]{
		next:     next,
		combiner: combiner,
		send:     send,
	}
}

// Handle
func (c GRPCCollector[T]) RoundTrip(req *http.Request) error {
	ctx := req.Context()
	ctx, cancel := context.WithCancel(ctx) // create a new context with a cancel function
	defer cancel()

	req = req.WithContext(ctx)
	resps, err := c.next.RoundTrip(req)
	if err != nil {
		return grpcError(err)
	}

	lastUpdate := time.Now()

	err = addNextAsync(ctx, resps, c.next, c.combiner, func() error {
		// check if we should send an update
		if time.Since(lastUpdate) > 500*time.Millisecond {
			lastUpdate = time.Now()

			// send a diff only during streaming
			resp, err := c.combiner.GRPCDiff()
			if err != nil {
				return err
			}
			err = c.send(resp)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return grpcError(err)
	}

	// send a final, complete response
	resp, err := c.combiner.GRPCFinal()
	if err != nil {
		return grpcError(err)
	}
	err = c.send(resp)
	if err != nil {
		return grpcError(err)
	}

	return nil
}

func grpcError(err error) error {
	// if is already a grpc err then just return. something with more context has already created a
	// standard grpc error and we will honor that
	if _, ok := status.FromError(err); ok {
		return err
	}

	return status.Error(codes.Internal, err.Error())
}
