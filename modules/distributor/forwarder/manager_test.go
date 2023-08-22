package forwarder

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	dslog "github.com/grafana/dskit/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"golang.org/x/exp/slices"
)

type mockWorkingOverrides struct {
	tenantIDs            []string
	tenantIDToForwarders map[string][]string
}

func (m *mockWorkingOverrides) TenantIDs() []string {
	return m.tenantIDs
}

func (m *mockWorkingOverrides) Forwarders(tenantID string) []string {
	return m.tenantIDToForwarders[tenantID]
}

type mockWorkingForwarder struct{}

func (m *mockWorkingForwarder) ForwardTraces(_ context.Context, _ ptrace.Traces) error {
	return nil
}

func (m *mockWorkingForwarder) Shutdown(_ context.Context) error {
	return nil
}

type mockFailingForwarder struct {
	forwardTracesErr error
	shutdownErr      error
}

func (m *mockFailingForwarder) ForwardTraces(_ context.Context, _ ptrace.Traces) error {
	return m.forwardTracesErr
}

func (m *mockFailingForwarder) Shutdown(_ context.Context) error {
	return m.shutdownErr
}

type mockChannelledInterceptorForwarder struct {
	next   Forwarder
	traces chan ptrace.Traces
}

func (m *mockChannelledInterceptorForwarder) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	m.traces <- traces
	return m.next.ForwardTraces(ctx, traces)
}

func (m *mockChannelledInterceptorForwarder) Shutdown(ctx context.Context) error {
	return m.next.Shutdown(ctx)
}

func newManagerWithForwarders(t *testing.T, forwarderNameToForwarder map[string]Forwarder, logger log.Logger, o Overrides) *Manager {
	t.Helper()

	manager, err := NewManager(ConfigList{}, logger, o, dslog.Level{})
	require.NoError(t, err)
	manager.forwarderNameToForwarder = forwarderNameToForwarder

	require.NoError(t, manager.start(context.Background()))
	t.Cleanup(func() {
		require.NoError(t, manager.stop(nil))
	})

	return manager
}

func TestNewManager_ReturnsNoErrorAndNonNilManagerWithValidConfigList(t *testing.T) {
	// Given
	cfgs := ConfigList{}
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{}

	// When
	got, err := NewManager(cfgs, logger, o, dslog.Level{})

	// Then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestNewManager_ReturnsErrorAndNilManagerWithInvalidConfigList(t *testing.T) {
	// Given
	cfgs := ConfigList{
		Config{Backend: "unknown"},
	}
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{}

	// When
	got, err := NewManager(cfgs, logger, o, dslog.Level{})

	// Then
	require.Error(t, err)
	require.Nil(t, got)
}

func TestManager_ForTenant_ReturnsSingleForwarderWhenSingleForwarderForTenantConfigured(t *testing.T) {
	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs: []string{"testTenantID"},
		tenantIDToForwarders: map[string][]string{
			"testTenantID": {"testForwarder"},
		},
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder": &mockWorkingForwarder{},
	}
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	forwarderList := manager.ForTenant("testTenantID")

	// Then
	require.Len(t, forwarderList, 1)
}

func TestManager_ForTenant_ReturnsTwoForwardersWhenTwoForwarderForTenantConfigured(t *testing.T) {
	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs: []string{"testTenantID"},
		tenantIDToForwarders: map[string][]string{
			"testTenantID": {"testForwarder1", "testForwarder2"},
		},
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": &mockWorkingForwarder{},
		"testForwarder2": &mockWorkingForwarder{},
	}
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	forwarderList := manager.ForTenant("testTenantID")

	// Then
	require.Len(t, forwarderList, 2)
}

func TestManager_ForTenant_ReturnsEmptySliceWhenNoForwardersForTenantConfigured(t *testing.T) {
	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs:            []string{"testTenantID"},
		tenantIDToForwarders: map[string][]string{},
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": &mockWorkingForwarder{},
		"testForwarder2": &mockWorkingForwarder{},
	}
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	forwarderList := manager.ForTenant("testTenantID")

	// Then
	require.Empty(t, forwarderList)
}

func TestManager_ForTenant_ReturnsEmptySliceWhenTenantNotConfigured(t *testing.T) {
	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs:            []string{},
		tenantIDToForwarders: map[string][]string{},
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": &mockWorkingForwarder{},
		"testForwarder2": &mockWorkingForwarder{},
	}
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	forwarderList := manager.ForTenant("testTenantID")

	// Then
	require.Empty(t, forwarderList)
}

func TestManager_ForTenant_List_ForwardTraces_ReturnsNoErrorAndCorrectlyForwardsBatchesToSingleForwarder(t *testing.T) {
	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs: []string{"testTenantID"},
		tenantIDToForwarders: map[string][]string{
			"testTenantID": {"testForwarder"},
		},
	}

	forwarderCh := make(chan ptrace.Traces)
	forwarder := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarderCh,
	}

	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder": forwarder,
	}

	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL")
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	err := manager.ForTenant("testTenantID").ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Equal(t, traces, <-forwarderCh)
}

func TestManager_ForTenant_List_ForwardTraces_ReturnsNoErrorAndCorrectlyForwardsBatchesToMultipleForwarders(t *testing.T) {
	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs: []string{"testTenantID"},
		tenantIDToForwarders: map[string][]string{
			"testTenantID": {"testForwarder1", "testForwarder2"},
		},
	}

	forwarder1Ch := make(chan ptrace.Traces)
	forwarder1 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder1Ch,
	}
	forwarder2Ch := make(chan ptrace.Traces)
	forwarder2 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder2Ch,
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": forwarder1,
		"testForwarder2": forwarder2,
	}
	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL")
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	err := manager.ForTenant("testTenantID").ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Equal(t, traces, <-forwarder1Ch)
	require.Equal(t, traces, <-forwarder2Ch)
}

func TestManager_ForTenant_List_ForwardTraces_ReturnsNoErrorAndCorrectlyForwardsBatchesToProperForwardersWhenNewForwarderIsAddedToOverridesConfig(t *testing.T) {
	// Step 1 - Setup manager with two forwarders for tenant and verify that both forwarders receive the trace.

	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs: []string{"testTenantID"},
		tenantIDToForwarders: map[string][]string{
			"testTenantID": {"testForwarder1", "testForwarder2"},
		},
	}

	forwarder1Ch := make(chan ptrace.Traces)
	forwarder1 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder1Ch,
	}
	forwarder2Ch := make(chan ptrace.Traces)
	forwarder2 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder2Ch,
	}
	forwarder3Ch := make(chan ptrace.Traces)
	forwarder3 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder3Ch,
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": forwarder1,
		"testForwarder2": forwarder2,
		"testForwarder3": forwarder3,
	}
	traces1 := ptrace.NewTraces()
	traces1.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL1")
	traces2 := ptrace.NewTraces()
	traces2.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL2")
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	err := manager.ForTenant("testTenantID").ForwardTraces(context.Background(), traces1)

	// Then
	require.NoError(t, err)
	require.Equal(t, traces1, <-forwarder1Ch)
	require.Equal(t, traces1, <-forwarder2Ch)
	require.Len(t, forwarder3Ch, 0)

	// Step 2 - Add additional forwarder, simulate "tick" and verify that all three forwarders receive the trace.
	currentForwardersForTenant := o.tenantIDToForwarders["testTenantID"]
	o.tenantIDToForwarders["testTenantID"] = append(currentForwardersForTenant, "testForwarder3")
	manager.updateQueueLists()

	// When
	err = manager.ForTenant("testTenantID").ForwardTraces(context.Background(), traces2)

	// Then
	require.NoError(t, err)
	require.Equal(t, traces2, <-forwarder1Ch)
	require.Equal(t, traces2, <-forwarder2Ch)
	require.Equal(t, traces2, <-forwarder3Ch)
}

func TestManager_ForTenant_List_ForwardTraces_ReturnsNoErrorAndCorrectlyForwardsBatchesToProperForwardersWhenOldForwarderIsRemovedFromOverridesConfig(t *testing.T) {
	// Step 1 - Setup manager with three forwarders for tenant and verify that all three forwarders receive the trace.

	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs: []string{"testTenantID"},
		tenantIDToForwarders: map[string][]string{
			"testTenantID": {"testForwarder1", "testForwarder2", "testForwarder3"},
		},
	}

	forwarder1Ch := make(chan ptrace.Traces)
	forwarder1 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder1Ch,
	}
	forwarder2Ch := make(chan ptrace.Traces)
	forwarder2 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder2Ch,
	}
	forwarder3Ch := make(chan ptrace.Traces)
	forwarder3 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder3Ch,
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": forwarder1,
		"testForwarder2": forwarder2,
		"testForwarder3": forwarder3,
	}
	traces1 := ptrace.NewTraces()
	traces1.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL1")
	traces2 := ptrace.NewTraces()
	traces2.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL2")
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	err := manager.ForTenant("testTenantID").ForwardTraces(context.Background(), traces1)

	// Then
	require.NoError(t, err)
	require.Equal(t, traces1, <-forwarder1Ch)
	require.Equal(t, traces1, <-forwarder2Ch)
	require.Equal(t, traces1, <-forwarder3Ch)

	// Step 2 - Remove one forwarder, simulate "tick" and verify that remaining forwarders receive the trace.
	idx := slices.Index(o.tenantIDToForwarders["testTenantID"], "testForwarder2")
	require.NotEqual(t, -1, idx)
	slices.Delete(o.tenantIDToForwarders["testTenantID"], idx, idx)
	manager.updateQueueLists()

	// When
	err = manager.ForTenant("testTenantID").ForwardTraces(context.Background(), traces2)

	// Then
	require.NoError(t, err)
	require.Equal(t, traces2, <-forwarder1Ch)
	require.Len(t, forwarder2Ch, 0)
	require.Equal(t, traces2, <-forwarder3Ch)
}

func TestManager_ForTenant_List_ForwardTraces_ReturnsNoErrorAndCorrectlyForwardsBatchesToMultipleForwardersForMultipleTenants(t *testing.T) {
	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs: []string{"testTenantID1", "testTenantID2"},
		tenantIDToForwarders: map[string][]string{
			"testTenantID1": {"testForwarder1", "testForwarder2"},
			"testTenantID2": {"testForwarder2"},
		},
	}

	forwarder1Ch := make(chan ptrace.Traces)
	forwarder1 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder1Ch,
	}
	forwarder2Ch := make(chan ptrace.Traces)
	forwarder2 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder2Ch,
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": forwarder1,
		"testForwarder2": forwarder2,
	}
	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL")
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	err := manager.ForTenant("testTenantID1").ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Equal(t, traces, <-forwarder1Ch)
	require.Equal(t, traces, <-forwarder2Ch)

	// When
	err = manager.ForTenant("testTenantID2").ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Len(t, forwarder1Ch, 0)
	require.Equal(t, traces, <-forwarder2Ch)
}

func TestManager_ForTenant_List_ForwardTraces_ReturnsNoErrorAndCorrectlyForwardsBatchesToProperForwardersWhenNewTenantIsAddedToOverridesConfig(t *testing.T) {
	// Step 1 - Setup manager with no tenants and verify that no batches are being forwarded

	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs:            []string{},
		tenantIDToForwarders: map[string][]string{},
	}

	forwarder1Ch := make(chan ptrace.Traces)
	forwarder1 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder1Ch,
	}
	forwarder2Ch := make(chan ptrace.Traces)
	forwarder2 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder2Ch,
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": forwarder1,
		"testForwarder2": forwarder2,
	}
	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL")
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	forwarderList := manager.ForTenant("testTenantID")
	err := forwarderList.ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Len(t, forwarderList, 0)
	require.Len(t, forwarder1Ch, 0)
	require.Len(t, forwarder2Ch, 0)

	// Step 2 - Add tenant to overrides config, simulate "tick", and verify that both forwarders receive the traces.
	o.tenantIDs = []string{"testTenantID"}
	o.tenantIDToForwarders = map[string][]string{
		"testTenantID": {"testForwarder1", "testForwarder2"},
	}
	manager.updateQueueLists()

	// When
	forwarderList = manager.ForTenant("testTenantID")
	err = forwarderList.ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Len(t, forwarderList, 2)
	require.Equal(t, traces, <-forwarder1Ch)
	require.Equal(t, traces, <-forwarder2Ch)
}

func TestManager_ForTenant_List_ForwardTraces_ReturnsNoErrorAndCorrectlyDoesNotForwardTracesToForwardersWhenTenantIsRemovedFromOverridesConfig(t *testing.T) {
	// Step 1 - Setup manager with two forwarders for tenant and verify that both forwarders receive the trace.

	// Given
	logger := log.NewNopLogger()
	o := &mockWorkingOverrides{
		tenantIDs: []string{"testTenantID"},
		tenantIDToForwarders: map[string][]string{
			"testTenantID": {"testForwarder1", "testForwarder2"},
		},
	}

	forwarder1Ch := make(chan ptrace.Traces)
	forwarder1 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder1Ch,
	}
	forwarder2Ch := make(chan ptrace.Traces)
	forwarder2 := &mockChannelledInterceptorForwarder{
		next:   &mockWorkingForwarder{},
		traces: forwarder2Ch,
	}
	forwarderNameToForwarder := map[string]Forwarder{
		"testForwarder1": forwarder1,
		"testForwarder2": forwarder2,
	}
	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL")
	manager := newManagerWithForwarders(t, forwarderNameToForwarder, logger, o)

	// When
	forwarderList := manager.ForTenant("testTenantID")
	err := forwarderList.ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Len(t, forwarderList, 2)
	require.Equal(t, traces, <-forwarder1Ch)
	require.Equal(t, traces, <-forwarder2Ch)

	// Step 2 - Remove forwarder from overrides config and verify that the traces are no longer forwarded.
	o.tenantIDs = []string{}
	o.tenantIDToForwarders = map[string][]string{}
	manager.updateQueueLists()

	// When
	forwarderList = manager.ForTenant("testTenantID")
	err = forwarderList.ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Len(t, forwarderList, 0)
	require.Len(t, forwarder1Ch, 0)
	require.Len(t, forwarder2Ch, 0)
}
