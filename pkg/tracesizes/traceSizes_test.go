package tracesizes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAddKeepsCountingPastLimit(t *testing.T) {
	tr := New("test")
	id := []byte("trace")

	require.Equal(t, 10, tr.Add(id, 10))

	res := tr.Allow(id, 5, 12)
	require.False(t, res.IsAllowed)
	require.Equal(t, 15, res.CurrentTotalSize)

	// Pushes rejected by the caller still count toward the total.
	require.Equal(t, 20, tr.Add(id, 5))
}

func TestDrainForgetsTraces(t *testing.T) {
	tr := New("test")
	tr.Add([]byte("a"), 10)
	tr.Add([]byte("b"), 20)

	tr.Drain()

	require.Equal(t, 3, tr.Add([]byte("a"), 3))
}

func TestClearIdleKeepsActiveTraces(t *testing.T) {
	tr := New("test")
	tr.Add([]byte("idle"), 10)

	cut := time.Now().Add(time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	tr.Add([]byte("active"), 5)

	tr.ClearIdle(cut)

	require.Equal(t, 6, tr.Add([]byte("active"), 1), "active trace keeps accumulating")
	require.Equal(t, 1, tr.Add([]byte("idle"), 1), "idle trace was forgotten")
}
