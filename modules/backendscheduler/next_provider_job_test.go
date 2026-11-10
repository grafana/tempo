package backendscheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/tempo/v3/modules/backendscheduler/provider"
	"github.com/grafana/tempo/v3/modules/backendscheduler/work"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNextProviderJob(t *testing.T) {
	newChans := func(n int) ([]chan *work.Job, []<-chan *work.Job) {
		bi := make([]chan *work.Job, n)
		ro := make([]<-chan *work.Job, n)
		for i := range bi {
			bi[i] = make(chan *work.Job)
			ro[i] = bi[i]
		}
		return bi, ro
	}

	want := &work.Job{ID: "want"}
	tests := []struct {
		name    string
		n       int         // providers
		closed  []int       // closed before the call
		send    int         // provider that sends...
		sends   []*work.Job // ...these jobs in order; the last one is expected back
		wantErr error
	}{
		{name: "first provider", n: 3, send: 0, sends: []*work.Job{want}},
		{name: "middle provider", n: 3, send: 1, sends: []*work.Job{want}},
		{name: "last provider", n: 3, send: 2, sends: []*work.Job{want}},
		{name: "closed channel is skipped and keeps its index", n: 2, closed: []int{0}, send: 1, sends: []*work.Job{want}},
		{name: "nil job is skipped", n: 1, send: 0, sends: []*work.Job{nil, want}},
		{name: "more providers than the stack buffer", n: maxStackProviders + 3, send: maxStackProviders + 2, sends: []*work.Job{want}},
		{name: "all channels closed", n: 2, closed: []int{0, 1}, wantErr: errProvidersClosed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			send, recv := newChans(tc.n)
			for _, i := range tc.closed {
				close(send[i])
			}
			go func() {
				for _, j := range tc.sends {
					send[tc.send] <- j
				}
			}()

			j, idx, err := nextProviderJob(context.Background(), recv)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Same(t, want, j)
			require.Equal(t, tc.send, idx)
		})
	}

	// The call must still be blocked before ctx is cancelled, so the result proves it waited.
	for _, n := range []int{0, 2} {
		t.Run(fmt.Sprintf("waits for ctx with %d idle providers", n), func(t *testing.T) {
			_, recv := newChans(n)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			errc := make(chan error, 1)
			go func() {
				_, _, err := nextProviderJob(ctx, recv)
				errc <- err
			}()

			select {
			case err := <-errc:
				t.Fatalf("returned before ctx was cancelled: %v", err)
			case <-time.After(50 * time.Millisecond):
			}

			cancel()
			require.ErrorIs(t, <-errc, context.Canceled)
		})
	}
}

// fakeProvider counts the jobs it has generated; each one is generated, then blocks on the send.
type fakeProvider struct {
	generated atomic.Int32
	closed    bool
}

func (f *fakeProvider) Start(ctx context.Context) <-chan *work.Job {
	jobs := make(chan *work.Job)
	if f.closed {
		close(jobs)
		return jobs
	}
	go func() {
		defer close(jobs)
		for {
			n := f.generated.Add(1)
			select {
			case jobs <- &work.Job{ID: fmt.Sprintf("fake-%p-%d", f, n)}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return jobs
}

var _ provider.Provider = (*fakeProvider)(nil)

// TestStartProvidersHoldsOneJobEach verifies the unbuffered handoff: each provider has generated one
// job and is blocked on the send, and generates another only once Next() takes one.
func TestStartProvidersHoldsOneJobEach(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	a, b := &fakeProvider{}, &fakeProvider{}
	s.providers = []provider.Provider{a, b}

	s.startProviders(ctx)

	// settle waits for the providers to have generated want jobs in total, then checks that no more follow.
	settle := func(want int32) {
		generated := func() int32 { return a.generated.Load() + b.generated.Load() }
		require.Eventually(t, func() bool { return generated() == want }, time.Second, time.Millisecond)
		time.Sleep(50 * time.Millisecond)
		require.Equal(t, want, generated(), "providers must not generate ahead of a worker")
	}

	settle(2)
	_, _, err := nextProviderJob(ctx, s.jobChans)
	require.NoError(t, err)
	settle(3)
}

// TestNextConcurrentWithStartProviders runs Next() while the providers are being started, as gRPC can
// before starting() finishes. It exists for the race detector: the channels must be published safely.
func TestNextConcurrentWithStartProviders(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	s.cfg.JobTimeout = 100 * time.Millisecond
	s.providers = []provider.Provider{&fakeProvider{}, &fakeProvider{}}

	var wg sync.WaitGroup
	begin := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-begin
			_, _ = s.Next(ctx, &tempopb.NextJobRequest{WorkerId: fmt.Sprintf("w%d", i)})
		}()
	}

	close(begin)
	s.startProviders(ctx)
	wg.Wait()
}

// TestNextBeforeProvidersStartGetsJob verifies a Next() that arrives before the providers start is
// served once they do, instead of waiting out job_timeout.
func TestNextBeforeProvidersStartGetsJob(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	s.cfg.JobTimeout = 10 * time.Second
	s.providers = []provider.Provider{&fakeProvider{}}

	done := make(chan error, 1)
	go func() {
		_, err := s.Next(ctx, &tempopb.NextJobRequest{WorkerId: "w"})
		done <- err
	}()

	time.Sleep(50 * time.Millisecond) // let Next() start waiting
	s.startProviders(ctx)

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Next() did not pick up a job after the providers started")
	}
}

func TestNextErrorCodes(t *testing.T) {
	tests := []struct {
		name      string
		providers []provider.Provider // started before Next, unless nil
		want      codes.Code
	}{
		{name: "providers not started wait out the timeout", want: codes.NotFound},
		{name: "all providers closed", providers: []provider.Provider{&fakeProvider{closed: true}}, want: codes.Internal},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, s := newQuiescenceScheduler(t)
			s.cfg.JobTimeout = 20 * time.Millisecond
			if tc.providers != nil {
				s.providers = tc.providers
				s.startProviders(ctx)
			}

			_, err := s.Next(ctx, &tempopb.NextJobRequest{WorkerId: "w"})
			require.Equal(t, tc.want, status.Code(err))
		})
	}
}

// TestProvidersFitStackBuffers fails when a provider is added beyond what nextProviderJob keeps
// on the stack, so the constant gets revisited deliberately instead of silently allocating.
func TestProvidersFitStackBuffers(t *testing.T) {
	_, s := newQuiescenceScheduler(t)
	require.LessOrEqual(t, len(s.providers), maxStackProviders)
}
