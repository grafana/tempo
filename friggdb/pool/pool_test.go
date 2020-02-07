package pool

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/grafana/frigg/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestResults(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 10,
	})

	ret := test.MakeTrace(5, nil)
	fn := func(payload interface{}) (proto.Message, error) {
		i := payload.(int)

		if i == 3 {
			return ret, nil
		}
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(payloads, fn)
	assert.NoError(t, err)

	equal := proto.Equal(ret, msg)
	if !equal {
		assert.Equal(t, ret, msg)
	}
}

func TestNoResults(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 10,
	})

	fn := func(payload interface{}) (proto.Message, error) {
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(payloads, fn)
	assert.Nil(t, msg)
	assert.Nil(t, err)
}

func TestError(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 1,
		QueueDepth: 10,
	})

	ret := fmt.Errorf("blerg")
	fn := func(payload interface{}) (proto.Message, error) {
		i := payload.(int)

		if i == 3 {
			return nil, ret
		}
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(payloads, fn)
	assert.Nil(t, msg)
	assert.Equal(t, ret, err)
}

func TestMultipleErrors(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 10,
	})

	ret := fmt.Errorf("blerg")
	fn := func(payload interface{}) (proto.Message, error) {
		return nil, ret
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(payloads, fn)
	assert.Nil(t, msg)
	assert.Equal(t, ret, err)
}

func TestTooManyJobs(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 10,
		QueueDepth: 3,
	})

	fn := func(payload interface{}) (proto.Message, error) {
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(payloads, fn)
	assert.Nil(t, msg)
	assert.Error(t, err)
}

func TestOneWorker(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 1,
		QueueDepth: 10,
	})

	ret := test.MakeTrace(10, nil)
	fn := func(payload interface{}) (proto.Message, error) {
		i := payload.(int)

		if i == 3 {
			return ret, nil
		}
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5}

	msg, err := p.RunJobs(payloads, fn)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(ret, msg))
}

func TestGoingHam(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 1000,
		QueueDepth: 10000,
	})

	wg := &sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			ret := test.MakeTrace(10, nil)
			fn := func(payload interface{}) (proto.Message, error) {
				i := payload.(int)

				time.Sleep(time.Duration(rand.Uint32()%100) * time.Millisecond)
				if i == 5 {
					return ret, nil
				}
				return nil, nil
			}
			payloads := []interface{}{1, 2, 3, 4, 5}

			msg, err := p.RunJobs(payloads, fn)
			assert.NoError(t, err)

			equal := proto.Equal(ret, msg)
			if !equal {
				assert.Equal(t, ret, msg)
			}
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestShutdown(t *testing.T) {
	p := NewPool(&Config{
		MaxWorkers: 1,
		QueueDepth: 10,
	})

	ret := test.MakeTrace(5, nil)
	fn := func(payload interface{}) (proto.Message, error) {
		i := payload.(int)

		if i == 3 {
			return ret, nil
		}
		return nil, nil
	}
	payloads := []interface{}{1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5}
	_, _ = p.RunJobs(payloads, fn)
	p.Shutdown()

	msg, err := p.RunJobs(payloads, fn)
	assert.Nil(t, msg)
	assert.Error(t, err)
}
