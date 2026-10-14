package pool

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestRunJobsAfterShutdown(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := NewPool(&Config{MaxWorkers: 1, QueueDepth: 2})
	p.Shutdown()

	require.NotPanics(t, func() {
		_, _, err := p.RunJobs(context.Background(), []interface{}{1}, func(context.Context, interface{}) (interface{}, error) {
			t.Error("job ran after shutdown")
			return nil, nil
		})
		require.ErrorIs(t, err, errPoolShutdown)
	})
}

func TestShutdownDuringRunJobs(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := NewPool(&Config{MaxWorkers: 1, QueueDepth: 2})
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	defer close(release)

	go func() {
		_, _, err := p.RunJobs(context.Background(), []interface{}{1, 2}, func(_ context.Context, payload interface{}) (interface{}, error) {
			if payload == 1 {
				close(started)
				<-release
			}
			return "late result", nil
		})
		finished <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("job did not start")
	}
	require.Eventually(t, func() bool { return len(p.workQueue) == 1 }, time.Second, time.Millisecond)
	p.Shutdown()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, errPoolShutdown)
	case <-time.After(time.Second):
		t.Fatal("RunJobs did not return on shutdown while a job was running")
	}
}

func TestConcurrentRunJobsAndShutdown(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	for range 50 {
		p := NewPool(&Config{MaxWorkers: 4, QueueDepth: 128})
		start := make(chan struct{})
		var wg sync.WaitGroup
		for range 16 {
			wg.Go(func() {
				<-start
				assert.NotPanics(t, func() {
					_, _, _ = p.RunJobs(context.Background(), []interface{}{1, 2, 3, 4}, func(context.Context, interface{}) (interface{}, error) {
						return "result", nil
					})
				})
			})
		}
		for range 4 {
			wg.Go(func() {
				<-start
				p.Shutdown()
			})
		}
		close(start)
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("concurrent RunJobs and Shutdown did not finish")
		}
	}
}
