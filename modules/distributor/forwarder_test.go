package distributor

import (
	"context"
	"sync"
	"testing"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForwarder(t *testing.T) {
	userID := "user-id"

	id, err := util.HexStringToTraceID("1234567890abcdef")
	require.NoError(t, err)

	b := test.MakeBatch(10, id)
	keys, rebatchedTraces, err := requestsByTraceID([]*v1.ResourceSpans{b}, userID, 10)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	f := newForwarder(func(_ context.Context, id string, keys []uint32, traces []*rebatchedTrace) error {
		assert.Equal(t, userID, id)
		assert.Equal(t, 1, len(keys))
		assert.Equal(t, 1, len(traces))
		assert.Equal(t, rebatchedTraces, traces)
		wg.Done()
		return nil
	})

	require.NoError(t, f.StartAsync(context.Background()))

	f.ForwardTraces(context.Background(), userID, keys, rebatchedTraces)
	wg.Wait()
}
