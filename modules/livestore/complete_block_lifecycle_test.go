package livestore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/flushqueues"
	testutils "github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

type completeBlockLifecycleCall struct {
	tenantID string
	blockID  uuid.UUID
}

type noopCompleteBlockFlusher struct{}

func (noopCompleteBlockFlusher) WriteBlock(context.Context, tempodb.WriteableBlock) error {
	return nil
}

type recordingCompleteBlockFlusher struct {
	mu       sync.Mutex
	blockIDs []uuid.UUID
}

type failOnceCompleteBlockFlusher struct {
	mu       sync.Mutex
	attempts int
	blockIDs []uuid.UUID
}

// blockingCompleteBlockFlusher simulates an in-flight flush that only exits
// once the provided context is canceled.
type blockingCompleteBlockFlusher struct {
	started chan struct{}
	done    chan struct{}
}

func newBlockingCompleteBlockFlusher() *blockingCompleteBlockFlusher {
	return &blockingCompleteBlockFlusher{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (f *blockingCompleteBlockFlusher) WriteBlock(ctx context.Context, _ tempodb.WriteableBlock) error {
	close(f.started)
	<-ctx.Done()
	close(f.done)
	return ctx.Err()
}

func (f *recordingCompleteBlockFlusher) WriteBlock(_ context.Context, block tempodb.WriteableBlock) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blockIDs = append(f.blockIDs, uuid.UUID(block.BlockMeta().BlockID))
	return nil
}

func (f *failOnceCompleteBlockFlusher) WriteBlock(_ context.Context, block tempodb.WriteableBlock) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.attempts++
	if f.attempts == 1 {
		return errors.New("forced flush failure")
	}

	f.blockIDs = append(f.blockIDs, uuid.UUID(block.BlockMeta().BlockID))
	return nil
}

func (f *recordingCompleteBlockFlusher) flushedBlockIDs() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]uuid.UUID, len(f.blockIDs))
	copy(out, f.blockIDs)
	return out
}

func (f *failOnceCompleteBlockFlusher) attemptCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attempts
}

func (f *failOnceCompleteBlockFlusher) flushedBlockIDs() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]uuid.UUID, len(f.blockIDs))
	copy(out, f.blockIDs)
	return out
}

type mockCompleteBlockLifecycle struct {
	completedCalls []completeBlockLifecycleCall
	reloadedCalls  []completeBlockLifecycleCall
	deleteResult   bool
	started        bool
	stopped        bool
}

type failOnceOnCompletedBlockLifecycle struct {
	completedCalls []completeBlockLifecycleCall
	reloadedCalls  []completeBlockLifecycleCall
	failuresLeft   int
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

func (m *failOnceOnCompletedBlockLifecycle) start(context.Context) {
	m.started = true
}

func (m *failOnceOnCompletedBlockLifecycle) stop() {
	m.stopped = true
}

func (m *failOnceOnCompletedBlockLifecycle) onCompletedBlock(_ context.Context, tenantID string, block *LocalBlock) error {
	m.completedCalls = append(m.completedCalls, completeBlockLifecycleCall{tenantID: tenantID, blockID: uuid.UUID(block.BlockMeta().BlockID)})
	if m.failuresLeft > 0 {
		m.failuresLeft--
		return errors.New("forced lifecycle failure")
	}
	return nil
}

func (m *failOnceOnCompletedBlockLifecycle) onReloadedBlock(_ context.Context, tenantID string, block *LocalBlock) error {
	m.reloadedCalls = append(m.reloadedCalls, completeBlockLifecycleCall{tenantID: tenantID, blockID: uuid.UUID(block.BlockMeta().BlockID)})
	return nil
}

func (m *failOnceOnCompletedBlockLifecycle) shouldDeleteCompleteBlock(_ *LocalBlock, _ time.Time) bool {
	return false
}

func TestNewCompleteBlockLifecycleUsesKafkaModeWhenConsumingFromKafka(t *testing.T) {
	cfg := defaultConfig(t, t.TempDir())
	cfg.ConsumeFromKafka = true

	lifecycle, err := newCompleteBlockLifecycle(cfg, nil, log.NewNopLogger())
	require.NoError(t, err)
	require.IsType(t, kafkaCompleteBlockLifecycle{}, lifecycle)
}

func TestNewCompleteBlockLifecycleUsesLocalModeWhenKafkaConsumptionIsDisabled(t *testing.T) {
	cfg := defaultConfig(t, t.TempDir())
	cfg.ConsumeFromKafka = false

	lifecycle, err := newCompleteBlockLifecycle(cfg, noopCompleteBlockFlusher{}, log.NewNopLogger())
	require.NoError(t, err)
	require.IsType(t, &localCompleteBlockLifecycle{}, lifecycle)
}

func TestNewCompleteBlockLifecycleLocalModeRequiresFlusher(t *testing.T) {
	cfg := defaultConfig(t, t.TempDir())
	cfg.ConsumeFromKafka = false

	lifecycle, err := newCompleteBlockLifecycle(cfg, nil, log.NewNopLogger())
	require.Error(t, err)
	require.Nil(t, lifecycle)
}

func TestLocalCompleteBlockLifecycleOnCompletedBlockEnqueuesBlock(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	lifecycleAny, err := newCompleteBlockLifecycle(cfg, noopCompleteBlockFlusher{}, log.NewNopLogger())
	require.NoError(t, err)
	lifecycle, ok := lifecycleAny.(*localCompleteBlockLifecycle)
	require.True(t, ok)

	block := inst.blocks.Load().completeBlocks[blockID]
	require.NoError(t, lifecycle.onCompletedBlock(t.Context(), testTenantID, block))
	require.False(t, lifecycle.completeBlockQueue.IsEmpty())
}

func TestLocalCompleteBlockLifecycleStopCancelsInFlightFlush(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	flusher := newBlockingCompleteBlockFlusher()
	lifecycleAny, err := newCompleteBlockLifecycle(cfg, flusher, log.NewNopLogger())
	require.NoError(t, err)
	lifecycle, ok := lifecycleAny.(*localCompleteBlockLifecycle)
	require.True(t, ok)

	block := inst.blocks.Load().completeBlocks[blockID]
	require.NoError(t, lifecycle.onCompletedBlock(t.Context(), testTenantID, block))

	lifecycle.start(t.Context())

	select {
	case <-flusher.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for in-flight flush to start")
	}

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		lifecycle.stop()
	}()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lifecycle stop")
	}

	select {
	case <-flusher.done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for in-flight flush cancellation")
	}
}

func TestLocalCompleteBlockLifecycleOnReloadedBlockEnqueuesUnflushedBlock(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	lifecycleAny, err := newCompleteBlockLifecycle(cfg, noopCompleteBlockFlusher{}, log.NewNopLogger())
	require.NoError(t, err)
	lifecycle, ok := lifecycleAny.(*localCompleteBlockLifecycle)
	require.True(t, ok)

	block := inst.blocks.Load().completeBlocks[blockID]
	require.NoError(t, lifecycle.onReloadedBlock(t.Context(), testTenantID, block))
	require.False(t, lifecycle.completeBlockQueue.IsEmpty())
}

func TestLocalCompleteBlockLifecycleOnReloadedBlockSkipsFlushedBlock(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	lifecycleAny, err := newCompleteBlockLifecycle(cfg, noopCompleteBlockFlusher{}, log.NewNopLogger())
	require.NoError(t, err)
	lifecycle, ok := lifecycleAny.(*localCompleteBlockLifecycle)
	require.True(t, ok)

	block := inst.blocks.Load().completeBlocks[blockID]
	require.NoError(t, block.SetFlushed(t.Context()))
	require.NoError(t, lifecycle.onReloadedBlock(t.Context(), testTenantID, block))
	require.True(t, lifecycle.completeBlockQueue.IsEmpty())
}

func TestLocalCompleteBlockLifecycleRetriesFailedFlush(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false
	cfg.CompleteBlockConcurrency = 1
	cfg.initialBackoff = 5 * time.Second

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	block := inst.blocks.Load().completeBlocks[blockID]

	synctest.Test(t, func(t *testing.T) {
		flusher := &failOnceCompleteBlockFlusher{}
		lifecycleAny, err := newCompleteBlockLifecycle(cfg, flusher, log.NewNopLogger())
		require.NoError(t, err)
		lifecycle, ok := lifecycleAny.(*localCompleteBlockLifecycle)
		require.True(t, ok)

		lifecycle.start(context.Background())
		defer lifecycle.stop()
		require.NoError(t, lifecycle.onCompletedBlock(context.Background(), testTenantID, block))

		time.Sleep(2 * cfg.initialBackoff)

		require.Equal(t, 2, flusher.attemptCount())
		require.Equal(t, []uuid.UUID{blockID}, flusher.flushedBlockIDs())
	})
}

func TestLocalCompleteBlockLifecycleStartStopProcessesQueuedBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	flusher := &recordingCompleteBlockFlusher{}
	lifecycleAny, err := newCompleteBlockLifecycle(cfg, flusher, log.NewNopLogger())
	require.NoError(t, err)
	lifecycle, ok := lifecycleAny.(*localCompleteBlockLifecycle)
	require.True(t, ok)

	block := inst.blocks.Load().completeBlocks[blockID]
	require.NoError(t, lifecycle.onCompletedBlock(t.Context(), testTenantID, block))
	require.False(t, lifecycle.completeBlockQueue.IsEmpty())

	lifecycle.start(t.Context())
	t.Cleanup(lifecycle.stop)

	require.Eventually(t, func() bool {
		return lifecycle.completeBlockQueue.IsEmpty()
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, []uuid.UUID{blockID}, flusher.flushedBlockIDs())
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
		completeQueues:         flushqueues.New[*completeOp](nil),
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
	require.Contains(t, inst.blocks.Load().completeBlocks, blockID)
	require.Equal(t, []completeBlockLifecycleCall{{tenantID: testTenantID, blockID: blockID}}, lifecycle.completedCalls)
}

func TestLiveStoreProcessCompleteOpRetriesLifecycleUsingExistingCompleteBlock(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false
	cfg.initialBackoff = 0

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createWalBlockForLifecycleTest(t, liveStore)
	lifecycle := &failOnceOnCompletedBlockLifecycle{failuresLeft: 1}
	liveStore.completeBlockLifecycle = lifecycle
	inst.completeBlockLifecycle = lifecycle

	op := &completeOp{
		tenantID:   testTenantID,
		blockID:    blockID,
		at:         time.Now(),
		bo:         liveStore.cfg.initialBackoff,
		maxBackoff: liveStore.cfg.maxBackoff,
	}

	err = liveStore.processCompleteOp(op)
	require.NoError(t, err)
	require.Contains(t, inst.blocks.Load().completeBlocks, blockID)
	require.Equal(t, []completeBlockLifecycleCall{{tenantID: testTenantID, blockID: blockID}}, lifecycle.completedCalls)

	err = liveStore.processCompleteOp(op)
	require.NoError(t, err)
	require.Equal(t, []completeBlockLifecycleCall{
		{tenantID: testTenantID, blockID: blockID},
		{tenantID: testTenantID, blockID: blockID},
	}, lifecycle.completedCalls)
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

	reloadedStore, err := New(reloadCfg, limits, noopCompleteBlockFlusher{}, testutils.NewTestingLogger(t), prometheus.NewRegistry())
	require.NoError(t, err)

	lifecycle := &mockCompleteBlockLifecycle{}
	reloadedStore.completeBlockLifecycle = lifecycle

	require.NoError(t, services.StartAndAwaitRunning(t.Context(), reloadedStore))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), reloadedStore))
	})

	require.Equal(t, []completeBlockLifecycleCall{{tenantID: testTenantID, blockID: blockID}}, lifecycle.reloadedCalls)
}

func TestLocalCompleteBlockLifecycleDeleteOldBlocksDeletesFlushedBlocksByAge(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	block := inst.blocks.Load().completeBlocks[blockID]
	require.NoError(t, block.SetFlushed(t.Context()))
	inst.blocks.Load().completeBlocks[blockID].BlockMeta().EndTime = time.Now().Add(-liveStore.cfg.CompleteBlockTimeout - time.Second)

	require.NoError(t, inst.deleteOldBlocks())
	require.Len(t, inst.blocks.Load().completeBlocks, 0)
}

func TestLocalCompleteBlockLifecycleDeleteOldBlocksKeepsUnflushedBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createCompleteBlockForLifecycleTest(t, liveStore)
	inst.blocks.Load().completeBlocks[blockID].BlockMeta().EndTime = time.Now().Add(-liveStore.cfg.CompleteBlockTimeout - time.Second)

	require.NoError(t, inst.deleteOldBlocks())
	require.Len(t, inst.blocks.Load().completeBlocks, 1)
}

func TestInstanceDeleteOldBlocksUsesCompleteBlockLifecycle(t *testing.T) {
	tests := []struct {
		name          string
		lifecycle     completeBlockLifecycle
		wantRemaining int
	}{
		{
			name:          "local lifecycle keeps old unflushed complete blocks",
			lifecycle:     nil,
			wantRemaining: 1,
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
			inst.blocks.Load().completeBlocks[blockID].BlockMeta().EndTime = time.Now().Add(-liveStore.cfg.CompleteBlockTimeout - time.Second)

			require.NoError(t, inst.deleteOldBlocks())
			require.Len(t, inst.blocks.Load().completeBlocks, tc.wantRemaining)
		})
	}
}

func createWalBlockForLifecycleTest(t *testing.T, liveStore *LiveStore) (*instance, uuid.UUID) {
	t.Helper()

	_, _ = pushToLiveStore(t, liveStore)

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)
	drained, err := inst.cutIdleTraces(t.Context(), true)
	require.NoError(t, err)
	require.True(t, drained, "should drain live traces in one iteration")

	blockID, err := inst.cutBlocks(t.Context(), true)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, blockID)
	require.NotContains(t, inst.blocks.Load().completeBlocks, blockID)

	return inst, blockID
}

func createCompleteBlockForLifecycleTest(t *testing.T, liveStore *LiveStore) (*instance, uuid.UUID) {
	t.Helper()

	inst, blockID := createWalBlockForLifecycleTest(t, liveStore)
	_, err := inst.completeBlock(t.Context(), blockID)
	require.NoError(t, err)
	require.Contains(t, inst.blocks.Load().completeBlocks, blockID)

	return inst, blockID
}
