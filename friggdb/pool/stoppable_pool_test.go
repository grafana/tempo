package pool

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStoppableJobs(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 1000,
		QueueDepth: 10000,
	})

	wg := &sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			ret := []byte{0x01, 0x03, 0x04}
			fn := func(payload interface{}, stopCh <-chan struct{}) error {
				for {
					select {
					case <-stopCh:
						return nil
					default:
						time.Sleep(time.Duration(rand.Uint32()%100) * time.Millisecond)
					}
				}
			}
			payloads := []interface{}{1, 2, 3, 4, 5}

			stopper, err := p.RunStoppableJobs(payloads, fn)
			assert.NoError(t, err)
			assert.NotNil(t, ret, stopper)
			err = stopper.Stop()
			assert.NoError(t, err)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestStoppableReturnImmediate(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 100,
	})
	ret := []byte{0x01, 0x03, 0x04}
	fn := func(payload interface{}, stopCh <-chan struct{}) error {
		return nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}
	time.Sleep(30 * time.Millisecond)

	stopper, err := p.RunStoppableJobs(payloads, fn)
	assert.NoError(t, err)
	assert.NotNil(t, ret, stopper)
	err = stopper.Stop()
	assert.NoError(t, err)
}

func TestStoppableErrorImmediate(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 100,
	})
	expectedErr := fmt.Errorf("super error")
	ret := []byte{0x01, 0x03, 0x04}
	fn := func(payload interface{}, stopCh <-chan struct{}) error {
		return expectedErr
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	stopper, err := p.RunStoppableJobs(payloads, fn)
	assert.NoError(t, err)
	assert.NotNil(t, ret, stopper)
	time.Sleep(10 * time.Millisecond)
	err = stopper.Stop()
	assert.Equal(t, expectedErr, err)
}

func TestStoppableErrors(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 100,
	})
	expectedErr := fmt.Errorf("super error")
	ret := []byte{0x01, 0x03, 0x04}
	fn := func(payload interface{}, stopCh <-chan struct{}) error {
		for {
			select {
			case <-stopCh:
				return expectedErr
			default:
				time.Sleep(time.Duration(rand.Uint32()%100) * time.Millisecond)
			}
		}
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	stopper, err := p.RunStoppableJobs(payloads, fn)
	assert.NoError(t, err)
	assert.NotNil(t, ret, stopper)
	time.Sleep(10 * time.Millisecond)
	err = stopper.Stop()
	assert.Equal(t, expectedErr, err)
}
