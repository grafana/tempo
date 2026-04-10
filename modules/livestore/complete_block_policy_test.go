package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	testutils "github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

type completeBlockPolicyCall struct {
	tenantID string
	blockID  uuid.UUID
}

type mockCompleteBlockPolicy struct {
	completedCalls []completeBlockPolicyCall
	reloadedCalls  []completeBlockPolicyCall
	deleteResult   bool
}

func (m *mockCompleteBlockPolicy) onCompletedBlock(_ context.Context, tenantID string, blockID uuid.UUID) error {
	m.completedCalls = append(m.completedCalls, completeBlockPolicyCall{tenantID: tenantID, blockID: blockID})
	return nil
}

func (m *mockCompleteBlockPolicy) onReloadedBlock(_ context.Context, tenantID string, blockID uuid.UUID, _ *LocalBlock) error {
	m.reloadedCalls = append(m.reloadedCalls, completeBlockPolicyCall{tenantID: tenantID, blockID: blockID})
	return nil
}

func (m *mockCompleteBlockPolicy) shouldDeleteCompleteBlock(_ *LocalBlock, _ time.Time) bool {
	return m.deleteResult
}

func TestLiveStoreProcessCompleteOpCallsCompleteBlockPolicy(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), liveStore))
	})

	inst, blockID := createWalBlockForPolicyTest(t, liveStore)
	policy := &mockCompleteBlockPolicy{}
	liveStore.completeBlockPolicy = policy

	err = liveStore.processCompleteOp(&completeOp{
		tenantID:   testTenantID,
		blockID:    blockID,
		at:         time.Now(),
		bo:         liveStore.cfg.initialBackoff,
		maxBackoff: liveStore.cfg.maxBackoff,
	})
	require.NoError(t, err)
	require.Contains(t, inst.completeBlocks, blockID)
	require.Equal(t, []completeBlockPolicyCall{{tenantID: testTenantID, blockID: blockID}}, policy.completedCalls)
}

func TestLiveStoreReloadBlocksCallsCompleteBlockPolicy(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.ConsumeFromKafka = false

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)

	_, blockID := createCompleteBlockForPolicyTest(t, liveStore)
	require.NoError(t, services.StopAndAwaitTerminated(t.Context(), liveStore))

	reloadCfg := defaultConfig(t, tmpDir)
	reloadCfg.ConsumeFromKafka = false

	limits, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	reloadedStore, err := New(reloadCfg, limits, testutils.NewTestingLogger(t), prometheus.NewRegistry())
	require.NoError(t, err)

	policy := &mockCompleteBlockPolicy{}
	reloadedStore.completeBlockPolicy = policy

	require.NoError(t, services.StartAndAwaitRunning(t.Context(), reloadedStore))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), reloadedStore))
	})

	require.Equal(t, []completeBlockPolicyCall{{tenantID: testTenantID, blockID: blockID}}, policy.reloadedCalls)
}

func TestInstanceDeleteOldBlocksUsesCompleteBlockPolicy(t *testing.T) {
	tests := []struct {
		name          string
		policy        completeBlockPolicy
		wantRemaining int
	}{
		{
			name:          "default policy deletes old complete blocks",
			policy:        nil,
			wantRemaining: 0,
		},
		{
			name:          "custom policy can keep old complete blocks",
			policy:        &mockCompleteBlockPolicy{deleteResult: false},
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

			if tc.policy != nil {
				liveStore.completeBlockPolicy = tc.policy
			}

			inst, blockID := createCompleteBlockForPolicyTest(t, liveStore)
			inst.completeBlocks[blockID].BlockMeta().EndTime = time.Now().Add(-liveStore.cfg.CompleteBlockTimeout - time.Second)

			require.NoError(t, inst.deleteOldBlocks())
			require.Len(t, inst.completeBlocks, tc.wantRemaining)
		})
	}
}

func createWalBlockForPolicyTest(t *testing.T, liveStore *LiveStore) (*instance, uuid.UUID) {
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

func createCompleteBlockForPolicyTest(t *testing.T, liveStore *LiveStore) (*instance, uuid.UUID) {
	t.Helper()

	inst, blockID := createWalBlockForPolicyTest(t, liveStore)
	require.NoError(t, inst.completeBlock(t.Context(), blockID))
	require.Contains(t, inst.completeBlocks, blockID)

	return inst, blockID
}
