package pipeline

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"google.golang.org/grpc/codes"
)

type GRPCCollector[T combiner.TResponse] struct {
	next      AsyncRoundTripper[combiner.PipelineResponse]
	combiner  combiner.GRPCCombiner[T]
	consumers int

	send func(T) error
}

func NewGRPCCollector[T combiner.TResponse](next AsyncRoundTripper[combiner.PipelineResponse], consumers int, combiner combiner.GRPCCombiner[T], send func(T) error) *GRPCCollector[T] {
	return &GRPCCollector[T]{
		next:      next,
		combiner:  combiner,
		consumers: consumers,
		send:      send,
	}
}

// RoundTrip implements the http.RoundTripper interface
func (c GRPCCollector[T]) RoundTrip(req *http.Request) error {
	ctx := req.Context()
	ctx, cancel := context.WithCancel(ctx) // create a new context with a cancel function
	defer cancel()

	ctx, span := tracer.Start(ctx, "GRPCCollector.RoundTrip")
	defer span.End()

	req = req.WithContext(ctx)
	resps, err := c.next.RoundTrip(NewHTTPRequest(req))
	if err != nil {
		return grpcError(err)
	}
	span.AddEvent("next.RoundTrip done")

	lastUpdate := time.Now()
	// sendDiffCb should return an error if the context is cancelled,
	// callback's error is used to exit early from the loop and return the error to the caller
	sendDiffCb := func() error {
		// check if we should send an update
		if time.Since(lastUpdate) > 500*time.Millisecond {
			lastUpdate = time.Now()
			// check and return the context errors, like ctx cancelled, etc
			if req.Context().Err() != nil {
				return req.Context().Err()
			}

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
	}

	err = consumeAndCombineResponses(ctx, c.consumers, resps, c.combiner, sendDiffCb)
	if err != nil {
		return grpcError(err)
	}
	span.AddEvent("consumeAndCombineResponses done")

	// send the final diff if there is anything left
	resp, err := c.combiner.GRPCDiff()
	if err != nil {
		return grpcError(err)
	}
	span.AddEvent("final combiner.GRPCDiff() done")
	// check and return the context errors, like ctx cancelled, etc
	if req.Context().Err() != nil {
		return grpcError(req.Context().Err())
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

	// if this is context cancelled, we return a grpc cancelled error
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, err.Error())
	}

	// rest all fall into internal server error
	return status.Error(codes.Internal, err.Error())
}
