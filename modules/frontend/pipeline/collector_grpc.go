package pipeline

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"google.golang.org/grpc/codes"
)

type GRPCCollector[T combiner.TResponse] struct {
	next     AsyncRoundTripper[*http.Response]
	combiner combiner.GRPCCombiner[T]

	send func(T) error
}

func NewGRPCCollector[T combiner.TResponse](next AsyncRoundTripper[*http.Response], combiner combiner.GRPCCombiner[T], send func(T) error) *GRPCCollector[T] {
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

	// stores any error that occurs during the streaming
	//  the wg protects the store from concurrent writes/reads
	var overallErr error

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		lastUpdate := time.Now()

		for {
			done, err := contextDone(ctx)
			if done {
				if err != nil {
					overallErr = err
				}
				break
			}

			resp, done, err := resps.Next(ctx)
			if err != nil {
				overallErr = err
				break
			}

			if resp != nil {
				err = c.combiner.AddRequest(resp, "")
				if err != nil {
					overallErr = err
					break
				}
			}

			// limit reached or http errors
			if c.combiner.ShouldQuit() {
				break
			}

			// pipeline exhausted
			if done {
				break
			}

			// check if we should send an update
			if time.Since(lastUpdate) > 500*time.Millisecond {
				lastUpdate = time.Now()

				// send a diff only during streaming
				resp, err := c.combiner.GRPCDiff()
				if err != nil {
					overallErr = err
					break
				}
				err = c.send(resp)
				if err != nil {
					overallErr = err
					break
				}
			}
		}
	}()

	// goroutine to close the streamingResps and send the final message
	wg.Wait()

	// if we have an error then no need to send a final message
	if overallErr != nil {
		return grpcError(overallErr)
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

func contextDone(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		err := ctx.Err()
		if err != nil && !errors.Is(err, context.Canceled) {
			return true, err
		}
		return true, nil
	default:
		return false, nil
	}
}
