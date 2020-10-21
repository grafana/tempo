package pool

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestResults(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 10,
	})
	opts := goleak.IgnoreCurrent()

	ret := []byte{0x01, 0x02}
	fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
		i := payload.(int)

		if i == 3 {
			return ret, nil
		}
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(context.Background(), payloads, fn)
	assert.NoError(t, err)
	assert.Equal(t, ret, msg)
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestNoResults(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 10,
	})
	opts := goleak.IgnoreCurrent()

	fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(context.Background(), payloads, fn)
	assert.Nil(t, msg)
	assert.Nil(t, err)
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestMultipleHits(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 10,
	})
	opts := goleak.IgnoreCurrent()

	ret := []byte{0x01, 0x02}
	fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
		return ret, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(context.Background(), payloads, fn)
	assert.Equal(t, ret, msg)
	assert.Nil(t, err)
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestError(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 1,
		QueueDepth: 10,
	})
	opts := goleak.IgnoreCurrent()

	ret := fmt.Errorf("blerg")
	fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
		i := payload.(int)

		if i == 3 {
			return nil, ret
		}
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(context.Background(), payloads, fn)
	assert.Nil(t, msg)
	assert.Equal(t, ret, err)
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestMultipleErrors(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 10,
	})
	opts := goleak.IgnoreCurrent()

	ret := fmt.Errorf("blerg")
	fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
		return nil, ret
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(context.Background(), payloads, fn)
	assert.Nil(t, msg)
	assert.Equal(t, ret, err)
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestTooManyJobs(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 3,
	})
	opts := goleak.IgnoreCurrent()

	fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(context.Background(), payloads, fn)
	assert.Nil(t, msg)
	assert.Error(t, err)
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestOneWorker(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 1,
		QueueDepth: 10,
	})
	opts := goleak.IgnoreCurrent()

	ret := []byte{0x01, 0x02, 0x03}
	fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
		i := payload.(int)

		if i == 3 {
			return ret, nil
		}
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(context.Background(), payloads, fn)
	assert.NoError(t, err)
	assert.Equal(t, ret, msg)
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestGoingHam(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 1000,
		QueueDepth: 10000,
	})
	opts := goleak.IgnoreCurrent()

	wg := &sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			ret := []byte{0x01, 0x03, 0x04}
			fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
				i := payload.(int)

				time.Sleep(time.Duration(rand.Uint32()%100) * time.Millisecond)
				if i == 5 {
					return ret, nil
				}
				return nil, nil
			}
			payloads := []interface{}{1, 2, 3, 4, 5}

			msg, err := p.RunJobs(context.Background(), payloads, fn)
			assert.NoError(t, err)
			assert.Equal(t, ret, msg)
			wg.Done()
		}()
	}

	wg.Wait()
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestOverloadingASmallPool(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()

	p := NewPool(&Config{
		MaxWorkers: 1,
		QueueDepth: 11,
	})
	opts := goleak.IgnoreCurrent()

	wg := &sync.WaitGroup{}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
				time.Sleep(time.Duration(rand.Uint32()%100) * time.Millisecond)
				return nil, nil
			}
			payloads := []interface{}{1, 2}
			_, _ = p.RunJobs(context.Background(), payloads, fn)

			wg.Done()
		}()
	}

	wg.Wait()
	goleak.VerifyNone(t, opts)

	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)
}

func TestShutdown(t *testing.T) {
	prePoolOpts := goleak.IgnoreCurrent()
	p := NewPool(&Config{
		MaxWorkers: 1,
		QueueDepth: 10,
	})

	ret := []byte{0x01, 0x03, 0x04}
	fn := func(ctx context.Context, payload interface{}) ([]byte, error) {
		i := payload.(int)

		if i == 3 {
			return ret, nil
		}
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5}
	_, _ = p.RunJobs(context.Background(), payloads, fn)
	p.Shutdown()
	goleak.VerifyNone(t, prePoolOpts)

	opts := goleak.IgnoreCurrent()
	msg, err := p.RunJobs(context.Background(), payloads, fn)
	assert.Nil(t, msg)
	assert.Error(t, err)
	goleak.VerifyNone(t, opts)
}
