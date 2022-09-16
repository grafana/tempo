package distributor

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const tenantID = "tenant-id"

func TestForwarder(t *testing.T) {
	oCfg := overrides.Limits{}
	oCfg.RegisterFlags(&flag.FlagSet{})

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

	assert.Equal(t, 0, len(f.queueManagers[tenantID].reqChan))
}

func TestForwarder_pushesQueued(t *testing.T) {
	oCfg := overrides.Limits{
		MetricsGeneratorForwarderQueueSize: 10,
		MetricsGeneratorForwarderWorkers:   1,
	}
	oCfg.RegisterFlags(&flag.FlagSet{})

	id, err := util.HexStringToTraceID("1234567890abcdef")
	require.NoError(t, err)

	b := test.MakeBatch(10, id)
	keys, rebatchedTraces, err := requestsByTraceID([]*v1.ResourceSpans{b}, tenantID, 10)
	require.NoError(t, err)

	o, err := overrides.NewOverrides(oCfg)
	require.NoError(t, err)

	shutdownCh := make(chan struct{})

	f := newForwarder(func(ctx context.Context, userID string, k []uint32, traces []*rebatchedTrace) error {
		<-shutdownCh
		return nil
	}, o)

	require.NoError(t, f.start(context.Background()))
	defer func() {
		close(shutdownCh)
		require.NoError(t, f.stop(nil))
		assert.Equal(t, 0, len(f.queueManagers[tenantID].reqChan))
	}()

	// 10 pushes are buffered, 1 is picked up by the worker

	for i := 0; i < 11; i++ {
		f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
		fmt.Println("sent trace #: ", i + 1)
		fmt.Println("Length: ", len(f.queueManagers[tenantID].reqChan))
	}

	// queue is full with 10 items
	fmt.Println("Length after all sent: ", len(f.queueManagers[tenantID].reqChan))
	assert.Equal(t, 10, len(f.queueManagers[tenantID].reqChan))
}

func TestForwarder_shutdown(t *testing.T) {
	oCfg := overrides.Limits{}
	oCfg.RegisterFlags(&flag.FlagSet{})
	oCfg.MetricsGeneratorForwarderQueueSize = 200

	id, err := util.HexStringToTraceID("1234567890abcdef")
	require.NoError(t, err)

	b := test.MakeBatch(10, id)
	keys, rebatchedTraces, err := requestsByTraceID([]*v1.ResourceSpans{b}, tenantID, 10)
	require.NoError(t, err)

	o, err := overrides.NewOverrides(oCfg)
	require.NoError(t, err)

	signalCh := make(chan struct{})
	f := newForwarder(func(ctx context.Context, userID string, k []uint32, traces []*rebatchedTrace) error {
		<-signalCh

		assert.Equal(t, tenantID, userID)
		assert.Equal(t, keys, k)
		assert.Equal(t, rebatchedTraces, traces)
		return nil
	}, o)

	require.NoError(t, f.start(context.Background()))
	defer func() {
		go func() {
			// Wait to unblock processing of requests so shutdown and draining the queue is done in parallel
			time.Sleep(time.Second)
			close(signalCh)
		}()
		require.NoError(t, f.stop(nil))
		assert.Equal(t, 0, len(f.queueManagers[tenantID].reqChan))
	}()

	for i := 0; i < 100; i++ {
		f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	}
}

func TestForwarder_overrides(t *testing.T) {
	overridesReloadInterval := 100 * time.Millisecond
	limits := overrides.Limits{}
	overridesFile := filepath.Join(t.TempDir(), "overrides.yaml")

	buff, err := yaml.Marshal(map[string]map[string]*overrides.Limits{
		"overrides": {
			tenantID: {
				MetricsGeneratorForwarderQueueSize: 10,
				MetricsGeneratorForwarderWorkers:   1,
			},
		},
	})
	require.NoError(t, err)

	err = os.WriteFile(overridesFile, buff, os.ModePerm)
	require.NoError(t, err)

	limits.PerTenantOverrideConfig = overridesFile
	limits.PerTenantOverridePeriod = model.Duration(overridesReloadInterval)

	id, err := util.HexStringToTraceID("1234567890abcdef")
	require.NoError(t, err)

	b := test.MakeBatch(10, id)
	keys, rebatchedTraces, err := requestsByTraceID([]*v1.ResourceSpans{b}, tenantID, 10)
	require.NoError(t, err)

	o, err := overrides.NewOverrides(limits)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.TODO(), o))

	signalCh := make(chan struct{})
	wg := sync.WaitGroup{}
	f := newForwarder(func(ctx context.Context, userID string, k []uint32, traces []*rebatchedTrace) error {
		wg.Done()
		<-signalCh
		return nil
	}, o)
	f.overridesInterval = overridesReloadInterval

	require.NoError(t, f.start(context.Background()))
	defer func() {
		close(signalCh)
		require.NoError(t, f.stop(nil))
		assert.Equal(t, 0, len(f.queueManagers[tenantID].reqChan))
	}()

	wg.Add(1)
	f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	wg.Wait()

	// 10 pushes are buffered, 10 are discarded
	for i := 0; i < 20; i++ {
		f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	}

	// queue is full with 10 items
	f.mutex.Lock()
	assert.Equal(t, 10, len(f.queueManagers[tenantID].reqChan))
	f.mutex.Unlock()
	wg.Add(10)

	buff, err = yaml.Marshal(map[string]map[string]*overrides.Limits{
		"overrides": {
			tenantID: {
				MetricsGeneratorForwarderQueueSize: 20,
				MetricsGeneratorForwarderWorkers:   2,
			},
		},
	})
	require.NoError(t, err)

	err = os.WriteFile(overridesFile, buff, os.ModePerm)
	require.NoError(t, err)

	// Wait triple the reload interval to ensure overrides are updated (overrides interval + forwarder interval)
	time.Sleep(3 * overridesReloadInterval)

	// Allow for pending requests to be processed so queueManager can be closed and a new one created
	for i := 0; i < 11; i++ {
		signalCh <- struct{}{}
	}
	wg.Wait()

	wg.Add(2)
	for i := 0; i < 2; i++ {
		f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	}
	wg.Wait()

	// 20 pushes are buffered, 10 are discarded
	for i := 0; i < 30; i++ {
		f.SendTraces(context.Background(), tenantID, keys, rebatchedTraces)
	}

	// queue is full with 20 items
	f.mutex.Lock()
	assert.Equal(t, 20, len(f.queueManagers[tenantID].reqChan))
	f.mutex.Unlock()
	wg.Add(20)
}
