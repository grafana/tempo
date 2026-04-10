package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/flushqueues"
	testutils "github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

type completeBlockLifecycleCall struct {
	tenantID string
	blockID  uuid.UUID
}

type mockCompleteBlockLifecycle struct {
	completedCalls []completeBlockLifecycleCall
	reloadedCalls  []completeBlockLifecycleCall
	deleteResult   bool
	started        bool
	stopped        bool
}

func (m *mockCompleteBlockLifecycle) start(context.Context) {
	m.started = true
}

func (m *mockCompleteBlockLifecycle) stop() {
	m.stopped = true
}

func (m *mockCompleteBlockLifecycle) onCompletedBlock(_ context.Context, tenantID string, block *LocalBlock) error {
	m.completedCalls = append(m.completedCalls, completeBlockLifecycleCall{tenantID: tenantID, blockID: uuid.UUID(block.BlockMeta().BlockID)})
	return nil
}

func (m *mockCompleteBlockLifecycle) onReloadedBlock(_ context.Context, tenantID string, block *LocalBlock) error {
	m.reloadedCalls = append(m.reloadedCalls, completeBlockLifecycleCall{tenantID: tenantID, blockID: uuid.UUID(block.BlockMeta().BlockID)})
	return nil
}

func (m *mockCompleteBlockLifecycle) shouldDeleteCompleteBlock(_ *LocalBlock, _ time.Time) bool {
	return m.deleteResult
}

func TestLiveStoreStartStopBackgroundProcessesControlsCompleteBlockLifecycle(t *testing.T) {
	cfg := defaultConfig(t, t.TempDir())
	cfg.ConsumeFromKafka = false
	cfg.holdAllBackgroundProcesses = false

	lifecycle := &mockCompleteBlockLifecycle{}
	liveStore := &LiveStore{
		cfg:                    cfg,
		logger:                 testutils.NewTestingLogger(t),
		completeBlockLifecycle: lifecycle,
		ctx:                    context.Background(),
		cancel:                 func() {},
		completeQueues:         flushqueues.New[*completeOp](1, nil),
		startupComplete:        make(chan struct{}),
	}

	liveStore.startAllBackgroundProcesses()
	require.True(t, lifecycle.started)

	liveStore.stopAllBackgroundProcesses()
	require.True(t, lifecycle.stopped)
}

func TestLiveStoreProcessCompleteOpCallsCompleteBlockLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createWalBlockForLifecycleTest(t, liveStore)
	lifecycle := &mockCompleteBlockLifecycle{}
	liveStore.completeBlockLifecycle = lifecycle
	inst.completeBlockLifecycle = lifecycle

	err = liveStore.processCompleteOp(&completeOp{
		tenantID:   testTenantID,
		blockID:    blockID,
		at:         time.Now(),
		bo:         liveStore.cfg.initialBackoff,
		maxBackoff: liveStore.cfg.maxBackoff,
	})
	require.NoError(t, err)
	require.Contains(t, inst.completeBlocks, blockID)
	require.Equal(t, []completeBlockLifecycleCall{{tenantID: testTenantID, blockID: blockID}}, lifecycle.completedCalls)
}

func TestLiveStoreReloadBlocksCallsCompleteBlockLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)

	_, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	require.NoError(t, services.StopAndAwaitTerminated(t.Context(), liveStore))

	reloadCfg := defaultConfig(t, tmpDir)
	reloadCfg.ConsumeFromKafka = false

	limits, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	reloadedStore, err := New(reloadCfg, limits, nil, testutils.NewTestingLogger(t), prometheus.NewRegistry())
	require.NoError(t, err)

	lifecycle := &mockCompleteBlockLifecycle{}
	reloadedStore.completeBlockLifecycle = lifecycle

	require.NoError(t, services.StartAndAwaitRunning(t.Context(), reloadedStore))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), reloadedStore))
	})

	require.Equal(t, []completeBlockLifecycleCall{{tenantID: testTenantID, blockID: blockID}}, lifecycle.reloadedCalls)
}

func TestInstanceDeleteOldBlocksUsesCompleteBlockLifecycle(t *testing.T) {
	tests := []struct {
		name          string
		lifecycle     completeBlockLifecycle
		wantRemaining int
	}{
		{
			name:          "default lifecycle deletes old complete blocks",
			lifecycle:     nil,
			wantRemaining: 0,
		},
		{
			name:          "custom lifecycle can keep old complete blocks",
			lifecycle:     &mockCompleteBlockLifecycle{deleteResult: false},
			wantRemaining: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			cfg := defaultConfig(t, tmpDir)
			cfg.ConsumeFromKafka = false

			liveStore, err := liveStoreWithConfig(t, cfg)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
			})

			if tc.lifecycle != nil {
				liveStore.completeBlockLifecycle = tc.lifecycle
			}

			inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
			if tc.lifecycle != nil {
				inst.completeBlockLifecycle = tc.lifecycle
			}
			inst.completeBlocks[blockID].BlockMeta().EndTime = time.Now().Add(-liveStore.cfg.CompleteBlockTimeout - time.Second)

			require.NoError(t, inst.deleteOldBlocks())
			require.Len(t, inst.completeBlocks, tc.wantRemaining)
		})
	}
}

func createWalBlockForLifecycleTest(t *testing.T, liveStore *LiveStore) (*instance, uuid.UUID) {
	t.Helper()

	_, _ = pushToLiveStore(t, liveStore)

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	require.NoError(t, inst.cutIdleTraces(t.Context(), true))

	blockID, err := inst.cutBlocks(t.Context(), true)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, blockID)
	require.NotContains(t, inst.completeBlocks, blockID)

	return inst, blockID
}

func createCompleteBlockForLifecycleTest(t *testing.T, liveStore *LiveStore) (*instance, uuid.UUID) {
	t.Helper()

	inst, blockID := createWalBlockForLifecycleTest(t, liveStore)
	_, err := inst.completeBlock(t.Context(), blockID)
	require.NoError(t, err)
	require.Contains(t, inst.completeBlocks, blockID)

	return inst, blockID
}
