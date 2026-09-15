package worker

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/httpgrpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
)

// sendReturns optionally keeps Send blocked after the frontend receives a frame.
// This lets tests separate result delivery from the writer's Send returning.
type mockSlotProcessClient struct {
	grpc.ClientStream
	ctx         context.Context
	incoming    chan *frontendv1pb.FrontendToClient
	outgoing    chan *frontendv1pb.ClientToFrontend
	sendReturns chan struct{}
}

func (m *mockSlotProcessClient) Context() context.Context { return m.ctx }

func (m *mockSlotProcessClient) Recv() (*frontendv1pb.FrontendToClient, error) {
	select {
	case frame := <-m.incoming:
		return frame, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *mockSlotProcessClient) Send(frame *frontendv1pb.ClientToFrontend) error {
	select {
	case m.outgoing <- frame:
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
	if m.sendReturns != nil {
		select {
		case <-m.sendReturns:
		case <-m.ctx.Done():
			return m.ctx.Err()
		}
	}
	return nil
}

func TestSlotProcessorJobOwnsContext(t *testing.T) {
	for _, tc := range []struct {
		name               string
		cancelWhileRunning bool
	}{
		{name: "completed"},
		{name: "cancelled", cancelWhileRunning: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				handlerContext := make(chan context.Context, 1)
				fp := newSlotTestProcessor(func(ctx context.Context, req *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
					handlerContext <- ctx
					if tc.cancelWhileRunning {
						<-ctx.Done()
					}
					return &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: []byte(req.Url)}, nil
				})
				s := slotProcessor{
					processor: fp,
					ctx:       ctx,
					slots:     1,
					jobs:      make(map[uint64]context.CancelFunc),
					completed: make(chan *frontendv1pb.JobResult, 1),
				}
				require.NoError(t, s.handleFrame(slotTestAssignments(1)))
				jobCtx := <-handlerContext
				cancellation := &frontendv1pb.FrontendToClient{Type: frontendv1pb.Type_JOB_FRAME, CancelJobIDs: []uint64{1}}
				if tc.cancelWhileRunning {
					require.NoError(t, s.handleFrame(cancellation))
				}
				s.wg.Wait()

				// Context cleanup must not depend on the scheduling loop reading
				// the completion, nor turn a successful result into a cancellation.
				require.ErrorIs(t, jobCtx.Err(), context.Canceled)
				want := slotTestResults(1).Results[0]
				if tc.cancelWhileRunning {
					want = &frontendv1pb.JobResult{JobID: 1, Cancelled: true}
				}
				require.Equal(t, want, <-s.completed)
				require.Len(t, s.jobs, 1, "a completed job retains credit until result handoff")
				require.NotNil(t, s.jobs[1])
				require.EqualError(t, s.handleFrame(slotTestAssignments(2)), "assignments exceed stream capacity (1 jobs, 0 free slots)")

				require.NoError(t, s.handleFrame(cancellation))
				synctest.Wait()
				require.Empty(t, s.completed, "late cancellation must not produce another result")
				require.Len(t, s.jobs, 1)
			})
		})
	}
}

func TestSlotProcessorFlushesOnTick(t *testing.T) {
	for _, tc := range []struct {
		name string
		jobs int
	}{
		{name: "empty"},
		{name: "single", jobs: 1},
		{name: "several", jobs: 2},
		{name: "full_frame", jobs: maxResultsPerFrame},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				stream := &mockSlotProcessClient{
					ctx:      ctx,
					incoming: make(chan *frontendv1pb.FrontendToClient),
					outgoing: make(chan *frontendv1pb.ClientToFrontend),
				}
				fp := newSlotTestProcessor(nil)
				ids := make([]uint64, tc.jobs)
				for i := range ids {
					ids[i] = uint64(i + 1)
				}
				done := make(chan error, 1)
				go func() {
					done <- fp.processSlotRequests(ctx, stream, slotTestAssignments(ids...), max(1, tc.jobs), cancel)
				}()
				synctest.Wait()
				requireNoSlotTestResults(t, stream)
				time.Sleep(maxResultFrameDelay - time.Nanosecond)
				synctest.Wait()
				requireNoSlotTestResults(t, stream)
				time.Sleep(time.Nanosecond)
				synctest.Wait()

				if tc.jobs > 0 {
					frame := receiveSlotTestResults(t, stream)
					// Handler completion order is intentionally unconstrained.
					require.ElementsMatch(t, slotTestResults(ids...).Results, frame.Results)
					require.LessOrEqual(t, frame.Size(), fp.maxMessageSize)
				}
				time.Sleep(maxResultFrameDelay)
				synctest.Wait()
				requireNoSlotTestResults(t, stream)
				cancel()
				require.ErrorIs(t, <-done, context.Canceled)
			})
		})
	}
}

func TestSlotProcessorSplitsResultsByBytes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stream := &mockSlotProcessClient{
			ctx:      ctx,
			incoming: make(chan *frontendv1pb.FrontendToClient),
			outgoing: make(chan *frontendv1pb.ClientToFrontend),
		}
		body := []byte(strings.Repeat("x", 64))
		fp := newSlotTestProcessor(func(context.Context, *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
			return &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: body}, nil
		})
		fp.maxMessageSize = 100 // Either result fits alone, but not both together.
		done := make(chan error, 1)
		go func() { done <- fp.processSlotRequests(ctx, stream, slotTestAssignments(1, 2), 2, cancel) }()
		synctest.Wait()
		requireNoSlotTestResults(t, stream)
		time.Sleep(maxResultFrameDelay - time.Nanosecond)
		synctest.Wait()
		requireNoSlotTestResults(t, stream)
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		first := receiveSlotTestResults(t, stream)
		require.Len(t, first.Results, 1)
		require.LessOrEqual(t, first.Size(), fp.maxMessageSize)
		synctest.Wait()
		second := receiveSlotTestResults(t, stream) // Both split frames flush on the same tick.
		require.Len(t, second.Results, 1)
		require.LessOrEqual(t, second.Size(), fp.maxMessageSize)
		results := append([]*frontendv1pb.JobResult(nil), first.Results...)
		results = append(results, second.Results...)
		require.ElementsMatch(t, []*frontendv1pb.JobResult{
			{JobID: 1, Response: &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: body}},
			{JobID: 2, Response: &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: body}},
		}, results)
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
	})
}

func TestSlotProcessorFrameBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name           string
		jobs           int
		jobsPerFrame   int
		maxMessageSize int
	}{
		{name: "count", jobs: 2*maxResultsPerFrame + 1, jobsPerFrame: maxResultsPerFrame, maxMessageSize: 1 << 20},
		{name: "exact_bytes", jobs: 5, jobsPerFrame: 2, maxMessageSize: slotTestResults(1, 2).Size()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				stream := &mockSlotProcessClient{
					ctx:      ctx,
					incoming: make(chan *frontendv1pb.FrontendToClient),
					outgoing: make(chan *frontendv1pb.ClientToFrontend),
				}
				fp := newSlotTestProcessor(nil)
				fp.maxMessageSize = tc.maxMessageSize
				ids := make([]uint64, tc.jobs)
				for i := range ids {
					ids[i] = uint64(i + 1)
				}
				done := make(chan error, 1)
				go func() { done <- fp.processSlotRequests(ctx, stream, slotTestAssignments(ids...), tc.jobs, cancel) }()

				synctest.Wait()
				requireNoSlotTestResults(t, stream)
				time.Sleep(maxResultFrameDelay - time.Nanosecond)
				synctest.Wait()
				requireNoSlotTestResults(t, stream)
				time.Sleep(time.Nanosecond)
				flushedAt := time.Now()

				var results []*frontendv1pb.JobResult
				for i := 0; i < tc.jobs/tc.jobsPerFrame; i++ {
					synctest.Wait()
					frame := receiveSlotTestResults(t, stream)
					require.Len(t, frame.Results, tc.jobsPerFrame)
					require.LessOrEqual(t, frame.Size(), tc.maxMessageSize)
					results = append(results, frame.Results...)
				}
				synctest.Wait()
				last := receiveSlotTestResults(t, stream)
				require.Len(t, last.Results, 1)
				require.LessOrEqual(t, last.Size(), tc.maxMessageSize)
				results = append(results, last.Results...)
				require.Equal(t, flushedAt, time.Now(), "all split frames, including the partial tail, must flush on the same tick")
				require.ElementsMatch(t, slotTestResults(ids...).Results, results)
				synctest.Wait()
				requireNoSlotTestResults(t, stream)
				cancel()
				require.ErrorIs(t, <-done, context.Canceled)
			})
		})
	}
}

func TestSlotProcessorFlushesBehindBlockedWriter(t *testing.T) {
	for _, tc := range []struct {
		name           string
		maxMessageSize int
		want           []*frontendv1pb.ClientToFrontend
	}{
		{name: "coalesce_until_handoff", maxMessageSize: 1 << 20, want: []*frontendv1pb.ClientToFrontend{slotTestResults(2, 3)}},
		{name: "split_by_bytes", maxMessageSize: slotTestResults(2, 3).Size() - 1, want: []*frontendv1pb.ClientToFrontend{slotTestResults(2), slotTestResults(3)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				finishSecond, finishThird := make(chan struct{}), make(chan struct{})
				fp := newSlotTestProcessor(func(ctx context.Context, req *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
					var finish <-chan struct{}
					switch req.Url {
					case "/job/2":
						finish = finishSecond
					case "/job/3":
						finish = finishThird
					}
					if finish != nil {
						select {
						case <-finish:
						case <-ctx.Done():
							return nil, ctx.Err()
						}
					}
					return &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: []byte(req.Url)}, nil
				})
				fp.maxMessageSize = tc.maxMessageSize
				stream := &mockSlotProcessClient{
					ctx:         ctx,
					incoming:    make(chan *frontendv1pb.FrontendToClient),
					outgoing:    make(chan *frontendv1pb.ClientToFrontend),
					sendReturns: make(chan struct{}),
				}
				done := make(chan error, 1)
				go func() { done <- fp.processSlotRequests(ctx, stream, slotTestAssignments(1, 2), 2, cancel) }()
				synctest.Wait()
				time.Sleep(maxResultFrameDelay)
				synctest.Wait()
				first := receiveSlotTestResults(t, stream)
				synctest.Wait() // Send remains blocked after the first result is received.

				// Receiving job 1's result permits a replacement even before Send returns.
				stream.incoming <- slotTestAssignments(3)
				synctest.Wait()
				require.Empty(t, done, "credit must be released at handoff, not after Send returns")
				close(finishSecond)
				synctest.Wait()
				time.Sleep(maxResultFrameDelay)
				synctest.Wait() // Job 2 is queued behind the blocked writer.
				time.Sleep(maxResultFrameDelay)
				synctest.Wait() // A second tick must not discard queued results.
				close(finishThird)
				synctest.Wait() // Job 3 is completed, but must wait for the next tick.
				time.Sleep(maxResultFrameDelay)
				synctest.Wait() // Eligible results can coalesce until handoff, if they fit.

				unblockedAt := time.Now()
				for _, want := range tc.want {
					stream.sendReturns <- struct{}{}
					synctest.Wait()
					frame := receiveSlotTestResults(t, stream)
					require.Equal(t, want, frame)
					require.LessOrEqual(t, frame.Size(), tc.maxMessageSize)
				}
				require.Equal(t, unblockedAt, time.Now(), "queued results must flush without waiting for another tick")
				// Later accumulation must not mutate frames already handed to the writer.
				require.Equal(t, slotTestResults(1), first)
				cancel()
				require.ErrorIs(t, <-done, context.Canceled)
			})
		})
	}
}

func TestSlotProcessorResultsAfterTickWaitForNextTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stream := &mockSlotProcessClient{
			ctx:         ctx,
			incoming:    make(chan *frontendv1pb.FrontendToClient),
			outgoing:    make(chan *frontendv1pb.ClientToFrontend),
			sendReturns: make(chan struct{}),
		}
		fp := newSlotTestProcessor(nil)
		fp.maxMessageSize = slotTestResults(1, 2).Size() - 1 // One result per frame.
		done := make(chan error, 1)
		go func() { done <- fp.processSlotRequests(ctx, stream, slotTestAssignments(1, 2), 2, cancel) }()
		synctest.Wait()
		requireNoSlotTestResults(t, stream)
		time.Sleep(maxResultFrameDelay)
		synctest.Wait()
		first := receiveSlotTestResults(t, stream)
		synctest.Wait()

		// Replace a delivered job while Send is still blocked. Its completion
		// must not join the previous tick's results, even when those are split.
		time.Sleep(time.Nanosecond)
		stream.incoming <- slotTestAssignments(3)
		synctest.Wait()
		require.Empty(t, done)
		stream.sendReturns <- struct{}{}
		synctest.Wait()
		second := receiveSlotTestResults(t, stream)
		results := append([]*frontendv1pb.JobResult(nil), first.Results...)
		results = append(results, second.Results...)
		require.ElementsMatch(t, slotTestResults(1, 2).Results, results)
		stream.sendReturns <- struct{}{}
		synctest.Wait()
		requireNoSlotTestResults(t, stream)

		time.Sleep(maxResultFrameDelay - time.Nanosecond)
		synctest.Wait()
		require.Equal(t, slotTestResults(3), receiveSlotTestResults(t, stream))
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
	})
}

func TestSlotProcessorCancelsWhileWriterBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cancelled := make(chan struct{})
		fp := newSlotTestProcessor(func(ctx context.Context, req *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
			if req.Url == "/job/2" {
				<-ctx.Done()
				close(cancelled)
				return nil, ctx.Err()
			}
			return &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: []byte(req.Url)}, nil
		})
		stream := &mockSlotProcessClient{
			ctx:         ctx,
			incoming:    make(chan *frontendv1pb.FrontendToClient),
			outgoing:    make(chan *frontendv1pb.ClientToFrontend),
			sendReturns: make(chan struct{}),
		}
		done := make(chan error, 1)
		go func() { done <- fp.processSlotRequests(ctx, stream, slotTestAssignments(1, 2), 2, cancel) }()
		synctest.Wait()
		time.Sleep(maxResultFrameDelay)
		synctest.Wait()
		first := receiveSlotTestResults(t, stream)
		synctest.Wait()

		stream.incoming <- &frontendv1pb.FrontendToClient{Type: frontendv1pb.Type_JOB_FRAME, CancelJobIDs: []uint64{2}}
		synctest.Wait()
		select {
		case <-cancelled:
		default:
			t.Fatal("job cancellation must not wait for the writer")
		}
		time.Sleep(maxResultFrameDelay)
		synctest.Wait()
		requireNoSlotTestResults(t, stream)
		stream.sendReturns <- struct{}{}
		synctest.Wait()
		require.Equal(t, &frontendv1pb.ClientToFrontend{
			Results: []*frontendv1pb.JobResult{{JobID: 2, Cancelled: true}},
		}, receiveSlotTestResults(t, stream))
		require.Equal(t, slotTestResults(1), first)
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
	})
}

func TestSlotProcessorShutdownWaitsForHandlers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		started, cancelled, finish := make(chan struct{}), make(chan struct{}), make(chan struct{})
		fp := newSlotTestProcessor(func(ctx context.Context, _ *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
			close(started)
			<-ctx.Done()
			close(cancelled)
			<-finish
			return nil, ctx.Err()
		})
		stream := &mockSlotProcessClient{
			ctx:      ctx,
			incoming: make(chan *frontendv1pb.FrontendToClient),
			outgoing: make(chan *frontendv1pb.ClientToFrontend),
		}
		done := make(chan error, 1)
		go func() { done <- fp.processSlotRequests(ctx, stream, slotTestAssignments(1), 1, cancel) }()
		<-started
		cancel()
		<-cancelled
		synctest.Wait()
		require.Empty(t, done, "the stream allocation cannot be reused while its handler is still running")
		close(finish)
		require.ErrorIs(t, <-done, context.Canceled)
		requireNoSlotTestResults(t, stream)
	})
}

func TestFitSlotResult(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *httpgrpc.HTTPResponse
		limit    int
		wantCode int32
		wantBody string
		wantErr  string
	}{
		{
			name:     "fits",
			response: &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: []byte("ok")},
			limit:    128,
			wantCode: http.StatusOK,
			wantBody: "ok",
		},
		{
			name:     "oversized_body",
			response: &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: []byte(strings.Repeat("x", 128))},
			limit:    128,
			wantCode: http.StatusRequestEntityTooLarge,
			wantBody: "response larger than slot frame limit (128 bytes)",
		},
		{
			name: "oversized_headers",
			response: &httpgrpc.HTTPResponse{
				Code:    http.StatusOK,
				Headers: []*httpgrpc.Header{{Key: "large", Values: []string{strings.Repeat("x", 128)}}},
			},
			limit:    128,
			wantCode: http.StatusRequestEntityTooLarge,
			wantBody: "response larger than slot frame limit (128 bytes)",
		},
		{
			name:     "error_cannot_fit",
			response: &httpgrpc.HTTPResponse{Code: http.StatusOK},
			limit:    1,
			wantErr:  "gRPC send limit 1 is too small for a job result",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fp := &frontendProcessor{maxMessageSize: tc.limit}
			result := &frontendv1pb.JobResult{JobID: 1, Response: tc.response}
			err := fp.fitSlotResult(result)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, &frontendv1pb.JobResult{
				JobID: 1, Response: &httpgrpc.HTTPResponse{Code: tc.wantCode, Body: []byte(tc.wantBody)},
			}, result)
			require.LessOrEqual(t, slotResultSize(result), tc.limit)
		})
	}
}

func newSlotTestProcessor(handler RequestHandlerFunc) *frontendProcessor {
	if handler == nil {
		handler = func(_ context.Context, req *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
			return &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: []byte(req.Url)}, nil
		}
	}
	return &frontendProcessor{
		handler:             handler,
		maxMessageSize:      1 << 20,
		metricRequestsTotal: prometheus.NewCounter(prometheus.CounterOpts{Name: "test_slot_requests_total"}),
		log:                 log.NewNopLogger(),
	}
}

func slotTestAssignments(ids ...uint64) *frontendv1pb.FrontendToClient {
	frame := &frontendv1pb.FrontendToClient{Type: frontendv1pb.Type_JOB_FRAME}
	for _, id := range ids {
		frame.Jobs = append(frame.Jobs, &frontendv1pb.Job{
			JobID: id, Request: &httpgrpc.HTTPRequest{Method: http.MethodGet, Url: fmt.Sprintf("/job/%d", id)},
		})
	}
	return frame
}

func slotTestResults(ids ...uint64) *frontendv1pb.ClientToFrontend {
	frame := &frontendv1pb.ClientToFrontend{}
	for _, id := range ids {
		frame.Results = append(frame.Results, &frontendv1pb.JobResult{
			JobID: id, Response: &httpgrpc.HTTPResponse{Code: http.StatusOK, Body: []byte(fmt.Sprintf("/job/%d", id))},
		})
	}
	return frame
}

// Receive without blocking: a blocking receive could advance synctest time to
// the next tick and conceal a missing count/byte flush or a delayed handoff.
func receiveSlotTestResults(t *testing.T, stream *mockSlotProcessClient) *frontendv1pb.ClientToFrontend {
	t.Helper()
	select {
	case frame := <-stream.outgoing:
		return frame
	default:
		t.Fatal("expected a result frame without advancing time")
		return nil
	}
}

func requireNoSlotTestResults(t *testing.T, stream *mockSlotProcessClient) {
	t.Helper()
	select {
	case frame := <-stream.outgoing:
		t.Fatalf("unexpected result frame: %v", frame)
	default:
	}
}
