package v1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/frontend/queue"
	"github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
)

// Channels synchronize the mock with the test and keep frames immutable after
// handoff. Cancelling ctx unblocks Send and Recv, as closing a real stream would.
type mockSlotProcessServer struct {
	grpc.ServerStream
	ctx        context.Context
	incoming   chan *frontendv1pb.ClientToFrontend
	outgoing   chan *frontendv1pb.FrontendToClient
	sendErrors chan error
	recvErrors chan error
}

func (m *mockSlotProcessServer) Context() context.Context { return m.ctx }

func (m *mockSlotProcessServer) Send(frame *frontendv1pb.FrontendToClient) error {
	select {
	case m.outgoing <- frame:
		return nil
	case err := <-m.sendErrors:
		return err
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

func (m *mockSlotProcessServer) Recv() (*frontendv1pb.ClientToFrontend, error) {
	select {
	case frame := <-m.incoming:
		return frame, nil
	case err := <-m.recvErrors:
		return nil, err
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func TestProcessSlotsBoundsStartupAllocation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f, err := New(Config{MaxOutstandingPerTenant: 16, MaxBatchSize: 2}, log.NewNopLogger(), prometheus.NewRegistry())
		require.NoError(t, err)
		defer f.subservices.StopAsync()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		server := &mockSlotProcessServer{ctx: ctx}
		done := make(chan error, 1)

		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		go func() {
			// Large enough to catch a slot-sized channel (8 MiB), without
			// risking the 32 GiB allocation allowed by the wire field.
			done <- f.processSlots(server, 1<<20)
		}()
		synctest.Wait()
		runtime.ReadMemStats(&after)

		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
		synctest.Wait()
		// Leave ample room for stream and dequeue goroutines, while keeping
		// startup memory independent of the peer's advertised capacity.
		require.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(1<<20))
	})
}

func TestProcessSlotsDrainsStoppedQueue(t *testing.T) {
	for _, outcome := range []string{"completed", "job_cancelled", "stream_cancelled"} {
		t.Run(outcome, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f, err := New(Config{MaxOutstandingPerTenant: 16, MaxBatchSize: 2}, log.NewNopLogger(), prometheus.NewRegistry())
				require.NoError(t, err)
				require.NoError(t, services.StartAndAwaitRunning(context.Background(), f.requestQueue))
				defer f.subservices.StopAsync()

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				jobCtx, cancelJob := context.WithCancel(context.Background())
				defer cancelJob()
				requests := []*request{newSlotTestRequest(jobCtx), newSlotTestRequest(context.Background())}
				for _, r := range requests {
					require.NoError(t, f.requestQueue.EnqueueRequest("tenant", r))
				}

				server := &mockSlotProcessServer{
					ctx:      ctx,
					incoming: make(chan *frontendv1pb.ClientToFrontend),
					outgoing: make(chan *frontendv1pb.FrontendToClient),
				}
				done := make(chan error, 1)
				go func() {
					// The unused third slot keeps a dequeue waiting while both
					// assigned jobs are still outstanding.
					done <- f.processSlots(server, 3)
				}()
				assignment := <-server.outgoing
				require.Equal(t, frontendv1pb.Type_JOB_FRAME, assignment.Type)
				require.Len(t, assignment.Jobs, 2)
				synctest.Wait()

				// Advance fake time past the queue's periodic empty-tenant
				// cleanup so it can stop without waiting for another cleanup.
				time.Sleep(time.Minute)
				synctest.Wait()
				require.NoError(t, services.StopAndAwaitTerminated(context.Background(), f.requestQueue))
				synctest.Wait()
				require.Empty(t, done, "queue shutdown must not close a stream with outstanding jobs")
				for _, r := range requests {
					require.Empty(t, r.err)
					require.Empty(t, r.response)
				}

				if outcome == "stream_cancelled" {
					cancel()
					require.ErrorIs(t, <-done, context.Canceled)
					for _, r := range requests {
						require.ErrorIs(t, <-r.err, context.Canceled)
						require.Empty(t, r.response)
					}
					return
				}

				for i, job := range assignment.Jobs {
					result := &frontendv1pb.JobResult{JobID: job.JobID, Response: &httpgrpc.HTTPResponse{Code: http.StatusOK}}
					if outcome == "job_cancelled" && i == 0 {
						cancelJob()
						require.Equal(t, &frontendv1pb.FrontendToClient{
							Type:         frontendv1pb.Type_JOB_FRAME,
							CancelJobIDs: []uint64{job.JobID},
						}, <-server.outgoing)
						result.Response = nil
						result.Cancelled = true
					}
					server.incoming <- &frontendv1pb.ClientToFrontend{Results: []*frontendv1pb.JobResult{result}}
					synctest.Wait()
					if result.Cancelled {
						require.ErrorIs(t, <-requests[i].err, context.Canceled)
						require.Empty(t, requests[i].response)
					} else {
						response := <-requests[i].response
						require.Equal(t, http.StatusOK, response.StatusCode)
						require.NoError(t, response.Body.Close())
						require.Empty(t, requests[i].err)
					}
					if i == 0 {
						require.Empty(t, done, "the remaining job must keep the stream open")
					}
				}
				require.ErrorIs(t, <-done, queue.ErrStopped)
			})
		})
	}
}

func TestDequeueSlotRequestsTransfersOwnership(t *testing.T) {
	for _, tc := range []struct {
		name                string
		receiveBeforeCancel bool
	}{
		{name: "received_by_loop", receiveBeforeCancel: true},
		{name: "received_by_cleanup"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f, err := New(Config{MaxOutstandingPerTenant: 16, MaxBatchSize: 2}, log.NewNopLogger(), prometheus.NewRegistry())
				require.NoError(t, err)
				defer f.subservices.StopAsync()

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				r := newSlotTestRequest(context.Background())
				require.NoError(t, f.requestQueue.EnqueueRequest("tenant", r))

				pulls := make(chan int)
				dequeued := f.dequeueSlotRequests(ctx, pulls)
				pulls <- 1
				synctest.Wait() // The dequeuer owns the batch and is waiting to hand it over.

				streamErr := errors.New("stream failed")
				deliveries := 0
				receive := func(batch slotDequeue) {
					require.NoError(t, batch.err)
					require.Equal(t, []queue.Request{r}, batch.requests)
					failRequests(batch.requests, streamErr)
					deliveries++
				}
				if tc.receiveBeforeCancel {
					receive(<-dequeued)
				}

				// Match processSlots cleanup: cancellation must neither drop the
				// owned batch nor replace the terminal error chosen by the loop.
				cancel()
				for batch := range dequeued {
					receive(batch)
				}

				require.Equal(t, 1, deliveries)
				require.ErrorIs(t, <-r.err, streamErr)
				require.Empty(t, r.err)
				require.Empty(t, r.response)
			})
		})
	}
}

func TestProcessSlotsRefillsOnlyCompletedSlots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f, err := New(Config{MaxOutstandingPerTenant: 16, MaxBatchSize: 2}, log.NewNopLogger(), prometheus.NewRegistry())
		require.NoError(t, err)
		defer f.subservices.StopAsync()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		jobCtx, cancelJob := context.WithCancel(context.Background())
		defer cancelJob()
		requests := []*request{
			newSlotTestRequest(jobCtx),
			newSlotTestRequest(context.Background()),
			newSlotTestRequest(context.Background()),
		}
		for _, r := range requests {
			require.NoError(t, f.requestQueue.EnqueueRequest("tenant", r))
		}

		server := &mockSlotProcessServer{
			ctx:      ctx,
			incoming: make(chan *frontendv1pb.ClientToFrontend),
			outgoing: make(chan *frontendv1pb.FrontendToClient),
		}
		done := make(chan error, 1)
		go func() { done <- f.processSlots(server, 2) }()
		assignment := <-server.outgoing
		require.Equal(t, frontendv1pb.Type_JOB_FRAME, assignment.Type)
		require.Len(t, assignment.Jobs, 2)
		require.Equal(t, uint64(1), assignment.Jobs[0].JobID)
		require.Equal(t, uint64(2), assignment.Jobs[1].JobID)

		cancelJob()
		require.Equal(t, &frontendv1pb.FrontendToClient{
			Type:         frontendv1pb.Type_JOB_FRAME,
			CancelJobIDs: []uint64{1},
		}, <-server.outgoing)
		synctest.Wait()
		select {
		case frame := <-server.outgoing:
			t.Fatalf("cancellation alone must not free a slot; got frame %v", frame)
		default:
		}
		for _, r := range requests {
			require.Empty(t, r.err)
			require.Empty(t, r.response)
		}

		// Complete the second job first. Its replacement must not wait for
		// the cancelled sibling's acknowledgement.
		server.incoming <- &frontendv1pb.ClientToFrontend{Results: []*frontendv1pb.JobResult{{
			JobID: 2, Response: &httpgrpc.HTTPResponse{Code: http.StatusCreated},
		}}}
		response := <-requests[1].response
		require.Equal(t, http.StatusCreated, response.StatusCode)
		require.NoError(t, response.Body.Close())
		replacement := <-server.outgoing
		require.Equal(t, frontendv1pb.Type_JOB_FRAME, replacement.Type)
		require.Len(t, replacement.Jobs, 1)
		require.Equal(t, uint64(3), replacement.Jobs[0].JobID)
		require.Empty(t, requests[0].err)

		server.incoming <- &frontendv1pb.ClientToFrontend{Results: []*frontendv1pb.JobResult{{JobID: 1, Cancelled: true}}}
		require.ErrorIs(t, <-requests[0].err, context.Canceled)
		synctest.Wait() // The freed slot now has a dequeue waiting on an empty queue.
		require.Empty(t, requests[2].response)
		require.Empty(t, done)

		server.incoming <- &frontendv1pb.ClientToFrontend{Results: []*frontendv1pb.JobResult{{
			JobID: 3, Response: &httpgrpc.HTTPResponse{Code: http.StatusAccepted},
		}}}
		response = <-requests[2].response
		require.Equal(t, http.StatusAccepted, response.StatusCode)
		require.NoError(t, response.Body.Close())

		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
		for _, r := range requests {
			require.Empty(t, r.err, "teardown must not notify completed requests again")
			require.Empty(t, r.response)
		}
	})
}

func TestSlotCompletionIgnoresLateCancellation(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		cancelBeforeCompletion bool
	}{
		{name: "callback_already_queued", cancelBeforeCompletion: true},
		{name: "callback_stopped_by_completion"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				r := newSlotTestRequest(ctx)
				s := frontendSlotStream{
					ctx:             context.Background(),
					outstanding:     make(map[uint64]*outstandingSlotRequest),
					cancelled:       make(chan uint64, 1),
					actualBatchSize: prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_slot_batch_size"}),
				}
				s.assign([]queue.Request{r})
				require.Len(t, s.pendingFrames, 1)
				s.pendingFrames = nil // The assignment has been handed to the writer.

				if tc.cancelBeforeCompletion {
					cancel()
					synctest.Wait()
					require.Len(t, s.cancelled, 1)
				}
				require.NoError(t, s.complete(&frontendv1pb.ClientToFrontend{Results: []*frontendv1pb.JobResult{{
					JobID: 1, Response: &httpgrpc.HTTPResponse{Code: http.StatusOK},
				}}}))

				if tc.cancelBeforeCompletion {
					s.cancelJob(<-s.cancelled)
				} else {
					cancel()
				}
				synctest.Wait()
				require.Empty(t, s.outstanding)
				require.Empty(t, s.pendingFrames)
				require.Empty(t, s.cancelled)
				response := <-r.response
				require.Equal(t, http.StatusOK, response.StatusCode)
				require.NoError(t, response.Body.Close())
				require.Empty(t, r.response)
				require.Empty(t, r.err)
			})
		})
	}
}

func TestProcessSlotsFailsUnsentRequestsOnTeardown(t *testing.T) {
	for _, outcome := range []string{"send_failed", "recv_failed", "stream_cancelled"} {
		t.Run(outcome, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f, err := New(Config{MaxOutstandingPerTenant: 16, MaxBatchSize: 1}, log.NewNopLogger(), prometheus.NewRegistry())
				require.NoError(t, err)
				defer f.subservices.StopAsync()

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				requests := []*request{
					newSlotTestRequest(context.Background()),
					newSlotTestRequest(context.Background()),
					newSlotTestRequest(context.Background()),
				}
				for _, r := range requests {
					require.NoError(t, f.requestQueue.EnqueueRequest("tenant", r))
				}
				server := &mockSlotProcessServer{
					ctx:        ctx,
					sendErrors: make(chan error),
					recvErrors: make(chan error),
				}
				done := make(chan error, 1)
				go func() { done <- f.processSlots(server, 3) }()
				// Send and Recv are blocked. One assignment is held by the writer;
				// the loop must still fill the other slots with pending frames.
				synctest.Wait()

				streamErr := errors.New("stream failed")
				switch outcome {
				case "send_failed":
					server.sendErrors <- streamErr
				case "recv_failed":
					server.recvErrors <- streamErr
				case "stream_cancelled":
					streamErr = context.Canceled
					cancel()
				}
				// On transport failure, leave the server context alive until
				// Process returns: joining the other blocked pump would deadlock.
				require.ErrorIs(t, <-done, streamErr)
				for _, r := range requests {
					require.ErrorIs(t, <-r.err, streamErr)
					require.Empty(t, r.err)
					require.Empty(t, r.response)
				}
			})
		})
	}
}

func newSlotTestRequest(ctx context.Context) *request {
	return &request{
		enqueueTime: time.Now(),
		queueSpan:   trace.SpanFromContext(ctx),
		request:     pipeline.NewHTTPRequest(httptest.NewRequest(http.MethodGet, "http://example.com", nil).WithContext(ctx)),
		err:         make(chan error, 1),
		response:    make(chan *http.Response, 1),
	}
}
