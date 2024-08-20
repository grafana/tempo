package distributor

import (
	"context"
	"flag"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v2/modules/overrides"
	v1 "github.com/grafana/tempo/v2/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v2/pkg/util"
	"github.com/grafana/tempo/v2/pkg/util/test"
)

const tenantID = "tenant-id"

func TestForwarder(t *testing.T) {
	oCfg := overrides.Config{}
	oCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

	id, err := util.HexStringToTraceID("1234567890abcdef")
	require.NoError(t, err)

	b := test.MakeBatch(10, id)
	keys, rebatchedTraces, err := requestsByTraceID([]*v1.ResourceSpans{b}, tenantID, 10)
	require.NoError(t, err)

	o, err := overrides.NewOverrides(oCfg, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	f := newGeneratorForwarder(
		log.NewNopLogger(),
		func(ctx context.Context, userID string, k []uint32, traces []*rebatchedTrace) error {
			assert.Equal(t, tenantID, userID)
			assert.Equal(t, keys, k)
			assert.Equal(t, rebatchedTraces, traces)
			wg.Done()
			return nil
		},
		o,
	)
	require.NoError(t, f.start(context.Background()))
	defer func() {
		require.NoError(t, f.stop(nil))
	}()

	wg.Add(1)
	f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	wg.Wait()

	wg.Add(1)
	f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	wg.Wait()
}

func TestForwarder_shutdown(t *testing.T) {
	oCfg := overrides.Config{}
	oCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
	oCfg.Defaults.MetricsGenerator.Forwarder.QueueSize = 200

	id, err := util.HexStringToTraceID("1234567890abcdef")
	require.NoError(t, err)

	b := test.MakeBatch(10, id)
	keys, rebatchedTraces, err := requestsByTraceID([]*v1.ResourceSpans{b}, tenantID, 10)
	require.NoError(t, err)

	o, err := overrides.NewOverrides(oCfg, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	signalCh := make(chan struct{})
	f := newGeneratorForwarder(
		log.NewNopLogger(),
		func(ctx context.Context, userID string, k []uint32, traces []*rebatchedTrace) error {
			<-signalCh

			assert.Equal(t, tenantID, userID)
			assert.Equal(t, keys, k)
			assert.Equal(t, rebatchedTraces, traces)
			return nil
		},
		o,
	)

	require.NoError(t, f.start(context.Background()))
	defer func() {
		go func() {
			// Wait to unblock processing of requests so shutdown and draining the queue is done in parallel
			time.Sleep(time.Second)
			close(signalCh)
		}()
		require.NoError(t, f.stop(nil))
	}()

	for i := 0; i < 100; i++ {
		f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	}
}
