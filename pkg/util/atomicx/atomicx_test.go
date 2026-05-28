package atomicx

import (
	"errors"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloat64(t *testing.T) {
	f := NewFloat64(1.5)
	require.Equal(t, 1.5, f.Load())

	f.Store(2.5)
	require.Equal(t, 2.5, f.Load())

	require.Equal(t, 5.0, f.Add(2.5))
	require.Equal(t, 5.0, f.Load())

	require.Equal(t, 3.0, f.Sub(2.0))

	require.True(t, f.CompareAndSwap(3.0, 7.0))
	require.False(t, f.CompareAndSwap(3.0, 9.0))
	require.Equal(t, 7.0, f.Load())

	// NaN round-trip via bits
	nan := NewFloat64(math.NaN())
	require.True(t, math.IsNaN(nan.Load()))
}

func TestFloat64ConcurrentAdd(t *testing.T) {
	f := NewFloat64(0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				f.Add(1)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 10000.0, f.Load())
}

func TestString(t *testing.T) {
	s := NewString("hello")
	require.Equal(t, "hello", s.Load())

	s.Store("world")
	require.Equal(t, "world", s.Load())

	// zero value reads as ""
	var z String
	require.Equal(t, "", z.Load())
}

func TestError(t *testing.T) {
	e := NewError(nil)
	require.NoError(t, e.Load())

	err1 := errors.New("boom")
	e.Store(err1)
	require.ErrorIs(t, e.Load(), err1)

	// CAS: only set if currently nil
	first := errors.New("first")
	second := errors.New("second")
	overall := &Error{}
	require.True(t, overall.CompareAndSwap(nil, first))
	require.False(t, overall.CompareAndSwap(nil, second))
	require.ErrorIs(t, overall.Load(), first)

	// zero value reads as nil
	var z Error
	assert.NoError(t, z.Load())
}
