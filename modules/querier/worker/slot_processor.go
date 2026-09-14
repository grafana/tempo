package worker

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/tempo/pkg/util/grpcutil"

	"github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
)

const (
	maxResultsPerFrame  = 32
	maxResultFrameDelay = 50 * time.Millisecond
)

type slotProcessor struct {
	processor *frontendProcessor
	ctx       context.Context
	slots     int
	lastJobID uint64

	// Jobs retain dispatch credit until their result frame is handed to the writer.
	// Handler goroutines release their contexts on exit; late cancellation is harmless.
	jobs      map[uint64]context.CancelFunc
	completed chan *frontendv1pb.JobResult
	wg        sync.WaitGroup
}

func (fp *frontendProcessor) processSlotRequests(ctx context.Context, stream frontendv1pb.Frontend_ProcessClient, first *frontendv1pb.FrontendToClient, slots int, cancelStream context.CancelFunc) error {
	ctx, cancel := context.WithCancel(ctx)
	s := slotProcessor{
		processor: fp,
		ctx:       ctx,
		slots:     slots,
		jobs:      make(map[uint64]context.CancelFunc),
		completed: make(chan *frontendv1pb.JobResult, slots),
	}
	defer func() {
		cancel()
		cancelStream()
		// Capacity cannot be reused by a reconnect until handlers actually exit.
		s.wg.Wait()
	}()

	incoming, outgoing, errs := grpcutil.StartStreamIO[*frontendv1pb.FrontendToClient, *frontendv1pb.ClientToFrontend](ctx, stream)

	if err := s.handleFrame(first); err != nil {
		return err
	}

	// A periodic tick bounds coalescing delay without allocating a timer per job.
	ticker := time.NewTicker(maxResultFrameDelay)
	defer ticker.Stop()

	// Only ticks move completed jobs into pending. Keep jobs queued until handoff,
	// with one candidate frame containing a copy of the queue's first results.
	var pending []*frontendv1pb.JobResult
	var frame *frontendv1pb.ClientToFrontend
	for {
		var send chan<- *frontendv1pb.ClientToFrontend
		if frame != nil {
			send = outgoing
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errs:
			return err
		case frame := <-incoming:
			if err := s.handleFrame(frame); err != nil {
				return err
			}
		case <-ticker.C:
			// Snapshot the completion queue.
			// Jobs finishing after this point wait for the next tick, even while this tick's results are being sent.
			for range len(s.completed) {
				result := <-s.completed
				if err := fp.fitSlotResult(result); err != nil {
					return err
				}
				pending = append(pending, result)
			}
			frame = nextSlotResultFrame(pending, fp.maxMessageSize)
		case send <- frame:
			// The frontend can only reuse these credits after receiving the frame.
			// Removing ownership at handoff also handles a replacement arriving before the writer's Send call returns.
			for _, result := range frame.Results {
				delete(s.jobs, result.JobID)
			}
			count := len(frame.Results)
			clear(pending[:count])
			pending = pending[count:]
			frame = nextSlotResultFrame(pending, fp.maxMessageSize)
		}
	}
}

func (s *slotProcessor) handleFrame(frame *frontendv1pb.FrontendToClient) error {
	if frame.Type != frontendv1pb.Type_JOB_FRAME {
		return fmt.Errorf("unexpected request type %v on slot scheduling stream", frame.Type)
	}
	if len(frame.Jobs) > s.slots-len(s.jobs) {
		return fmt.Errorf("assignments exceed stream capacity (%d jobs, %d free slots)", len(frame.Jobs), s.slots-len(s.jobs))
	}
	for _, job := range frame.Jobs {
		if job == nil || job.Request == nil || job.JobID <= s.lastJobID {
			return fmt.Errorf("invalid or reused job ID on slot scheduling stream")
		}
		if err := s.ctx.Err(); err != nil {
			return err
		}
		s.lastJobID = job.JobID
		ctx, cancel := context.WithCancel(s.ctx)
		s.jobs[job.JobID] = cancel
		s.wg.Go(func() {
			defer cancel()
			response := s.processor.runRequest(ctx, job.Request)
			result := &frontendv1pb.JobResult{JobID: job.JobID, Response: response}
			if ctx.Err() != nil {
				result.Cancelled = true
				result.Response = nil
			}
			select {
			case s.completed <- result:
			case <-s.ctx.Done():
			}
		})
	}
	for _, id := range frame.CancelJobIDs {
		// Late cancellation of a completed job is a no-op, not another result.
		if cancel := s.jobs[id]; cancel != nil {
			cancel()
		}
	}
	return nil
}

// nextSlotResultFrame packs a prefix of pending without changing the queue.
// fitSlotResult has already ensured that each result fits on its own.
func nextSlotResultFrame(pending []*frontendv1pb.JobResult, byteLimit int) *frontendv1pb.ClientToFrontend {
	if len(pending) == 0 {
		return nil
	}
	count, size := 0, 0
	for _, result := range pending {
		n := slotResultSize(result)
		if count == maxResultsPerFrame || size+n > byteLimit {
			break
		}
		count++
		size += n
	}
	return &frontendv1pb.ClientToFrontend{
		// The writer owns its slice; removing sent jobs from pending must not
		// mutate a frame while Send is still marshaling it.
		Results: append([]*frontendv1pb.JobResult(nil), pending[:count]...),
	}
}

func slotResultSize(result *frontendv1pb.JobResult) int {
	return (&frontendv1pb.ClientToFrontend{Results: []*frontendv1pb.JobResult{result}}).Size()
}

func (fp *frontendProcessor) fitSlotResult(result *frontendv1pb.JobResult) error {
	if slotResultSize(result) <= fp.maxMessageSize {
		return nil
	}
	// runRequest checks body size for legacy responses; slot frames must also
	// account for headers, job IDs, and protobuf envelopes. Keep 413 non-retryable.
	result.Response = &httpgrpc.HTTPResponse{
		Code: http.StatusRequestEntityTooLarge,
		Body: []byte(fmt.Sprintf("response larger than slot frame limit (%d bytes)", fp.maxMessageSize)),
	}
	if slotResultSize(result) > fp.maxMessageSize {
		return fmt.Errorf("gRPC send limit %d is too small for a job result", fp.maxMessageSize)
	}
	return nil
}
