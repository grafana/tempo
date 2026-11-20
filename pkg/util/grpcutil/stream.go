package grpcutil

import (
	"context"

	"github.com/gogo/protobuf/proto"
)

type StreamingClient[I, O proto.Message] interface {
	Recv() (I, error)
	Send(O) error
}

// StartStreamIO moves frames between the blocking gRPC stream and the scheduling loop.
// Send and Recv can both block indefinitely, so neither may be called from the loop goroutine.
// Returning from Process cancels the server-side context and unblocks any Send or Recv still in progress,
// so the loop must not join these goroutines before returning.
func StartStreamIO[I, O proto.Message](ctx context.Context, server StreamingClient[I, O]) (<-chan I, chan<- O, <-chan error) {
	incoming := make(chan I)
	outgoing := make(chan O)
	errs := make(chan error, 2) // one slot per stream, so neither block on the way out

	go func() {
		for {
			frame, err := server.Recv()
			if err != nil {
				errs <- err
				return
			}
			select {
			case incoming <- frame:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case frame := <-outgoing:
				if err := server.Send(frame); err != nil {
					errs <- err
					return
				}
			}
		}
	}()

	return incoming, outgoing, errs
}
