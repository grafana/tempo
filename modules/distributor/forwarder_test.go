package distributor

import (
	"context"
	"flag"
	"sync"
	"testing"

	"github.com/grafana/tempo/modules/overrides"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForwarder(t *testing.T) {
	oCfg := overrides.Limits{}
	oCfg.RegisterFlags(&flag.FlagSet{})

	tenantID := "tenant-id"
	id, err := util.HexStringToTraceID("1234567890abcdef")
	require.NoError(t, err)

	b := test.MakeBatch(10, id)
	keys, rebatchedTraces, err := requestsByTraceID([]*v1.ResourceSpans{b}, tenantID, 10)
	require.NoError(t, err)

	o, err := overrides.NewOverrides(oCfg)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	f := newForwarder(func(ctx context.Context, userID string, k []uint32, traces []*rebatchedTrace) error {
		assert.Equal(t, tenantID, userID)
		assert.Equal(t, keys, k)
		assert.Equal(t, rebatchedTraces, traces)
		wg.Done()
		return nil
	}, o)

	wg.Add(1)
	f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	wg.Wait()
}

func TestForwarder_pushesQueued(t *testing.T) {
	oCfg := overrides.Limits{
		MetricsGeneratorSendQueueSize: 10,
		MetricsGeneratorSendWorkers:   1,
	}
	oCfg.RegisterFlags(&flag.FlagSet{})

	tenantID := "tenant-id"
	id, err := util.HexStringToTraceID("1234567890abcdef")
	require.NoError(t, err)

	b := test.MakeBatch(10, id)
	keys, rebatchedTraces, err := requestsByTraceID([]*v1.ResourceSpans{b}, tenantID, 10)
	require.NoError(t, err)

	o, err := overrides.NewOverrides(oCfg)
	require.NoError(t, err)

	f := newForwarder(func(ctx context.Context, userID string, k []uint32, traces []*rebatchedTrace) error {
		<-make(chan struct{}) // forever block
		return nil
	}, o)

	// 10 pushes are buffered, 1 is picked up by the worker
	for i := 0; i < 11; i++ {
		f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	}

	// queue is full with 10 items
	assert.Equal(t, 10, len(f.queueManagers[tenantID].reqChan))
}
