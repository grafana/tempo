package v1

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/tempo/modules/frontend/queue"
	"github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
	"github.com/grafana/tempo/pkg/util/grpcutil"
	"github.com/prometheus/client_golang/prometheus"
)

type outstandingSlotRequest struct {
	request    *request
	stopCancel func() bool
}

type slotDequeue struct {
	requests []queue.Request
	err      error
}

// frontendSlotStream holds the mutable state of one slot scheduling stream.
type frontendSlotStream struct {
	ctx context.Context

	outstanding map[uint64]*outstandingSlotRequest
	nextID      uint64

	// pendingFrames are queued for the writer in the order they must be sent.
	pendingFrames []*frontendv1pb.FrontendToClient

	// cancelled carries job IDs from context cancellation callbacks back to the loop,
	// so the loop stays the only writer of outstanding.
	// The buffer is bounded by frontend configuration rather than peer-supplied capacity;
	// callbacks can wait for the loop or stream cancellation when it fills.
	cancelled chan uint64

	actualBatchSize prometheus.Histogram
}

func (f *Frontend) processSlots(server frontendv1pb.Frontend_ProcessServer, slots int) (returnErr error) {
	ctx, cancel := context.WithCancel(server.Context())
	s := &frontendSlotStream{
		ctx:             ctx,
		outstanding:     make(map[uint64]*outstandingSlotRequest),
		cancelled:       make(chan uint64, min(slots, f.cfg.MaxBatchSize)),
		actualBatchSize: f.actualBatchSize,
	}

	pulls := make(chan int)
	dequeued := f.dequeueSlotRequests(ctx, pulls)

	defer func() {
		cancel()

		// Never hand a nil error to a waiting RoundTrip: it would return a nil response and no error.
		// Every exit path below is an error today, so this only guards ones added later.
		err := returnErr
		if err == nil {
			err = errors.New("slot scheduling stream closed")
		}

		// Receive any batch still owned by the dequeuer.
		// It closes dequeued when it exits, so draining also waits for dequeue cleanup to finish.
		for batch := range dequeued {
			failRequests(batch.requests, err)
		}
		for _, pending := range s.outstanding {
			pending.stopCancel()
			pending.request.err <- err
		}
	}()

	incoming, outgoing, errs := grpcutil.StartStreamIO[*frontendv1pb.ClientToFrontend, *frontendv1pb.FrontendToClient](ctx, server)

	pulling := false
	queueStopped := false
	for {
		if queueStopped && len(s.outstanding) == 0 {
			return queue.ErrStopped
		}

		// Every accepted pull has one reply.
		// Listen for that reply only while pulling;
		// after a terminal reply the dequeuer closes its output.
		// Nil channels disable inactive operations without blocking the loop.
		var pull chan int
		var receive <-chan slotDequeue
		count := min(slots-len(s.outstanding), f.cfg.MaxBatchSize)
		if pulling {
			receive = dequeued
		} else if !queueStopped && count > 0 {
			pull = pulls
		}
		var send chan<- *frontendv1pb.FrontendToClient
		var frame *frontendv1pb.FrontendToClient
		if len(s.pendingFrames) > 0 {
			send, frame = outgoing, s.pendingFrames[0]
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errs:
			return err
		case pull <- count:
			// Reserve capacity even while the queue is empty.
			// Completions and cancellations continue to be processed during this blocking dequeue.
			pulling = true
		case batch := <-receive:
			pulling = false
			if batch.err != nil {
				// The queue returns no requests alongside an error,
				// but fail any it did hand over rather than dropping them.
				failRequests(batch.requests, batch.err)
				if errors.Is(batch.err, queue.ErrStopped) {
					// Queue shutdown only guarantees dispatch, not completion.
					// Keep processing results and cancellations until jobs drain.
					queueStopped = true
					continue
				}
				return batch.err
			}
			s.assign(batch.requests)
		case send <- frame:
			s.pendingFrames[0] = nil
			s.pendingFrames = s.pendingFrames[1:]
		case result := <-incoming:
			if err := s.complete(result); err != nil {
				return err
			}
		case id := <-s.cancelled:
			s.cancelJob(id)
		}
	}
}

// dequeueSlotRequests starts a goroutine that answers each pull with a batch of at most that many requests.
// The returned channel is unbuffered and closes when the goroutine exits.
// After cancelling ctx,
// the caller must drain the channel to unblock any pending handoff and account for all dequeued requests.
func (f *Frontend) dequeueSlotRequests(ctx context.Context, pulls <-chan int) <-chan slotDequeue {
	out := make(chan slotDequeue)
	go func() {
		defer close(out)
		last := queue.FirstUser()
		for {
			var count int
			select {
			case <-ctx.Done():
				return
			case count = <-pulls:
			}

			requests, idx, err := f.requestQueue.GetNextRequestForQuerier(ctx, last, make([]queue.Request, count))
			last = idx

			active := requests[:0]
			for _, wrapper := range requests {
				r := wrapper.(*request)
				f.queueDuration.Observe(time.Since(r.enqueueTime).Seconds())
				r.queueSpan.End()
				if r.OriginalContext().Err() == nil {
					active = append(active, wrapper)
				}
			}
			// If every request was expired, come back to the same user instead of
			// rotating past it. This drains a large expired query for a tenant and
			// allows them to execute a real query.
			if len(active) == 0 {
				last = last.ReuseLastUser()
			}

			// Transfer ownership to the loop, even during cancellation: its
			// cleanup receives any remaining batch and reports the stream error.
			out <- slotDequeue{requests: active, err: err}
			if err != nil {
				return
			}
		}
	}()
	return out
}

// assign turns dequeued requests into job frames and records them as outstanding.
func (s *frontendSlotStream) assign(requests []queue.Request) {
	frame := newJobFrame()

	for _, wrapper := range requests {
		r := wrapper.(*request)
		if r.OriginalContext().Err() != nil {
			continue
		}
		wire, err := requestToHTTPGRPC(r)
		if err != nil {
			r.err <- fmt.Errorf("encode query job: %w", err)
			continue
		}

		s.nextID++
		id := s.nextID
		job := &frontendv1pb.Job{JobID: id, Request: wire}

		frame.Jobs = append(frame.Jobs, job)

		s.outstanding[id] = &outstandingSlotRequest{
			request: r,
			stopCancel: context.AfterFunc(r.OriginalContext(), func() {
				select {
				case s.cancelled <- id:
				case <-s.ctx.Done():
				}
			}),
		}
	}

	s.queueFrame(frame)
}

// cancelJob asks the querier to abandon a job whose caller has gone away. The
// slot stays occupied until the querier reports the result, so outstanding
// keeps the entry.
func (s *frontendSlotStream) cancelJob(id uint64) {
	// The completion may have arrived first and removed the entry. Job IDs are
	// never reused and each cancellation fires at most once, so there is no
	// repeat delivery to suppress.
	if s.outstanding[id] == nil {
		return
	}
	s.queueFrame(&frontendv1pb.FrontendToClient{
		Type: frontendv1pb.Type_JOB_FRAME, CancelJobIDs: []uint64{id},
	})
}

func (s *frontendSlotStream) complete(frame *frontendv1pb.ClientToFrontend) error {
	if len(frame.Results) == 0 {
		return fmt.Errorf("empty completion frame on slot scheduling stream")
	}
	for _, result := range frame.Results {
		if result == nil || (result.Response == nil && !result.Cancelled) {
			return fmt.Errorf("invalid job result on slot scheduling stream")
		}
		pending := s.outstanding[result.JobID]
		if pending == nil {
			return fmt.Errorf("completion for unknown job %d", result.JobID)
		}
		pending.stopCancel()
		delete(s.outstanding, result.JobID)
		if result.Cancelled {
			pending.request.err <- context.Canceled
		} else {
			pending.request.response <- httpGRPCResponseToHTTPResponse(result.Response)
		}
	}
	return nil
}

// queueFrame hands a frame to the writer, dropping it if it carries nothing.
func (s *frontendSlotStream) queueFrame(frame *frontendv1pb.FrontendToClient) {
	if len(frame.Jobs) == 0 && len(frame.CancelJobIDs) == 0 {
		return
	}
	if len(frame.Jobs) > 0 {
		s.actualBatchSize.Observe(float64(len(frame.Jobs)))
	}
	s.pendingFrames = append(s.pendingFrames, frame)
}

func newJobFrame() *frontendv1pb.FrontendToClient {
	return &frontendv1pb.FrontendToClient{Type: frontendv1pb.Type_JOB_FRAME}
}

func failRequests(requests []queue.Request, err error) {
	for _, wrapper := range requests {
		wrapper.(*request).err <- err
	}
}
