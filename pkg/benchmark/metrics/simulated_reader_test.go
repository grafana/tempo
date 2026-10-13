package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

func simulated(latency time.Duration, bandwidth int64) (*SimulatedReader, *[]time.Duration) {
	var slept []time.Duration
	return &SimulatedReader{
		RawReader: &backend.MockRawReader{R: make([]byte, 2<<20)},
		Latency:   latency,
		Bandwidth: bandwidth,
		sleep: func(_ context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		},
	}, &slept
}

func TestSimulatedReaderChargesLatencyPlusTransfer(t *testing.T) {
	ctx := context.Background()
	r, slept := simulated(30*time.Millisecond, 100<<20)

	require.NoError(t, r.ReadRange(ctx, "data", nil, 0, make([]byte, 1<<20), nil))
	require.NoError(t, r.ReadRange(ctx, "data", nil, 0, make([]byte, 8<<20), nil))
	rc, size, err := r.Read(ctx, "meta", nil, nil)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, int64(2<<20), size)

	require.Equal(t, []time.Duration{
		30*time.Millisecond + 10*time.Millisecond,
		30*time.Millisecond + 80*time.Millisecond,
		30*time.Millisecond + 20*time.Millisecond,
	}, *slept)
}

func TestSimulatedReaderUnlimitedBandwidth(t *testing.T) {
	r, slept := simulated(5*time.Millisecond, 0)

	require.NoError(t, r.ReadRange(context.Background(), "data", nil, 0, make([]byte, 64<<20), nil))
	require.Equal(t, []time.Duration{5 * time.Millisecond}, *slept)
}

func TestNewSimulatedReaderPassesThroughWhenDisabled(t *testing.T) {
	raw := &backend.MockRawReader{}
	require.Same(t, backend.RawReader(raw), NewSimulatedReader(raw, 0, 0))
	require.IsType(t, &SimulatedReader{}, NewSimulatedReader(raw, time.Millisecond, 0))
	require.IsType(t, &SimulatedReader{}, NewSimulatedReader(raw, 0, 1<<20))
}

func TestSimulatedReaderStopsOnCancel(t *testing.T) {
	r := NewSimulatedReader(&backend.MockRawReader{R: []byte("x")}, time.Hour, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.ErrorIs(t, r.ReadRange(ctx, "data", nil, 0, make([]byte, 1), nil), context.Canceled)
	_, _, err := r.Read(ctx, "meta", nil, nil)
	require.ErrorIs(t, err, context.Canceled)
}
