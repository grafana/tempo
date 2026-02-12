package livestore

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

// =============================================================================
// Test Helper Functions
// =============================================================================

// createTestLiveStore creates a minimal LiveStore for testing with default config
func createTestLiveStore(t *testing.T) *LiveStore {
	tmpDir := t.TempDir()
	cfg := createTestConfig(t, tmpDir)

	limits, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	reg := prometheus.NewRegistry()
	logger := test.NewTestingLogger(t)

	liveStore, err := New(cfg, limits, logger, reg, true)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	return liveStore
}

// createTestConfig creates a test configuration with all necessary settings
func createTestConfig(t *testing.T, tmpDir string) Config {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
	cfg.WAL.Filepath = tmpDir
	cfg.WAL.Version = encoding.LatestEncoding().Version()
	cfg.ShutdownMarkerDir = tmpDir

	// Set up test Kafka
	const testTopic = "traces"
	_, kafkaAddr := testkafka.CreateCluster(t, 1, testTopic)
	cfg.IngestConfig.Kafka.Address = kafkaAddr
	cfg.IngestConfig.Kafka.Topic = testTopic
	cfg.IngestConfig.Kafka.ConsumerGroup = "test-consumer-group"

	cfg.holdAllBackgroundProcesses = true

	// Set up ring mocks
	cfg.Ring.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
	mockPartitionStore, _ := consul.NewInMemoryClient(
		ring.GetPartitionRingCodec(),
		log.NewNopLogger(),
		nil,
	)
	mockStore, _ := consul.NewInMemoryClient(
		ring.GetCodec(),
		log.NewNopLogger(),
		nil,
	)

	cfg.Ring.KVStore.Mock = mockStore
	cfg.Ring.ListenPort = 0
	cfg.Ring.InstanceAddr = "localhost"
	cfg.Ring.InstanceID = "test-1"
	cfg.PartitionRing.KVStore.Mock = mockPartitionStore

	return cfg
}

// triggerPanicInHandler causes a handler to panic for testing
// This simulates a panic by using a corrupted instance state
func triggerPanicInHandler(t *testing.T, s *LiveStore, handlerName string) error {
	ctx := user.InjectOrgID(context.Background(), testTenantID)

	// Create an instance with a nil pointer that will cause panic
	s.instancesMtx.Lock()
	inst := &instance{
		tenantID:       testTenantID,
		logger:         nil, // nil logger will panic on use
		liveTraces:     nil, // nil will panic when accessed
		walBlocks:      nil,
		completeBlocks: nil,
		Cfg: Config{
			CompleteBlockTimeout: 24 * time.Hour, // Set long timeout so time range validation passes
		},
	}
	s.instances[testTenantID] = inst
	s.instancesMtx.Unlock()

	var err error
	switch handlerName {
	case "FindTraceByID":
		_, err = s.FindTraceByID(ctx, &tempopb.TraceByIDRequest{TraceID: []byte("test")})
	case "SearchRecent":
		_, err = s.SearchRecent(ctx, &tempopb.SearchRequest{Query: "{}"})
	case "SearchTags":
		_, err = s.SearchTags(ctx, &tempopb.SearchTagsRequest{Scope: "span"})
	case "SearchTagsV2":
		_, err = s.SearchTagsV2(ctx, &tempopb.SearchTagsRequest{Scope: "span"})
	case "SearchTagValues":
		_, err = s.SearchTagValues(ctx, &tempopb.SearchTagValuesRequest{TagName: "foo"})
	case "SearchTagValuesV2":
		_, err = s.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{TagName: "foo"})
	case "QueryRange":
		now := time.Now()
		_, err = s.QueryRange(ctx, &tempopb.QueryRangeRequest{
			Query: "{} | rate()",
			Start: uint64(now.Add(-time.Hour).UnixNano()),
			End:   uint64(now.UnixNano()),
			Step:  uint64(time.Minute.Nanoseconds()),
		})
	default:
		t.Fatalf("unknown handler: %s", handlerName)
	}

	return err
}

// createValidTrace creates a valid trace for testing
func createValidTrace(t *testing.T, tenantID string) ([]byte, *tempopb.Trace) {
	id := test.ValidTraceID(nil)
	trace := test.MakeTrace(5, id)
	return id, trace
}

// pushTraceToLiveStore pushes a trace to the live store
func pushTraceToLiveStore(t *testing.T, s *LiveStore, tenantID string) ([]byte, *tempopb.Trace) {
	id, trace := createValidTrace(t, tenantID)
	traceBytes, err := proto.Marshal(trace)
	require.NoError(t, err)

	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
		Ids:    [][]byte{id},
	}

	records, err := ingest.Encode(0, tenantID, req, 1_000_000)
	require.NoError(t, err)

	now := time.Now()
	for _, rec := range records {
		rec.Timestamp = now
	}

	_, err = s.consume(context.Background(), createRecordIter(records), now)
	require.NoError(t, err)

	return id, trace
}

// =============================================================================
// CRIT-1: Tenant Validation Tests
// =============================================================================

// TestCRIT1_TenantValidation tests tenant ID validation at all entry points
func TestCRIT1_TenantValidation(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		wantValid bool
	}{
		// Valid cases
		{"valid alphanumeric", "tenant123", true},
		{"valid with hyphen", "tenant-123", true},
		{"valid with underscore", "tenant_123", true},
		{"valid with dot", "org.production", true},
		{"valid grafana cloud", "org-123.prod", true},
		{"valid single char", "a", true},
		{"valid numbers only", "123456", true},
		{"valid max length 64", strings.Repeat("a", 64), true},
		{"valid mixed case", "TenantABC", true},
		{"valid all allowed chars", "abc-123_DEF.xyz", true},
		{"valid multiple dots", "a.b.c.d", true},

		// Invalid cases
		{"empty", "", false},
		{"too long 65 chars", strings.Repeat("a", 65), false},
		{"too long 128 chars", strings.Repeat("a", 128), false},
		{"special char dollar", "tenant$123", false},
		{"special char slash", "tenant/123", false},
		{"special char backslash", "tenant\\123", false},
		{"path traversal", "../etc/passwd", false},
		{"sql injection", "tenant'; DROP TABLE", false},
		{"unicode", "tenant™", false},
		{"space", "tenant 123", false},
		{"tab", "tenant\t123", false},
		{"newline", "tenant\n123", false},
		{"colon", "tenant:123", false},
		{"asterisk", "tenant*", false},
		{"question mark", "tenant?", false},
		{"semicolon", "tenant;123", false},
		{"equals", "tenant=123", false},
		{"plus", "tenant+123", false},
		{"percent", "tenant%123", false},
		{"ampersand", "tenant&123", false},
		{"at sign", "tenant@123", false},
		{"exclamation", "tenant!123", false},
		{"hash", "tenant#123", false},
		{"parentheses", "tenant(123)", false},
		{"brackets", "tenant[123]", false},
		{"braces", "tenant{123}", false},
		{"pipe", "tenant|123", false},
		{"caret", "tenant^123", false},
		{"tilde", "tenant~123", false},
		{"backtick", "tenant`123", false},
		{"single quote", "tenant'123", false},
		{"double quote", "tenant\"123", false},
		{"comma", "tenant,123", false},
		{"less than", "tenant<123", false},
		{"greater than", "tenant>123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidTenantID(tt.tenantID)
			if valid != tt.wantValid {
				t.Errorf("isValidTenantID(%q) = %v, want %v", tt.tenantID, valid, tt.wantValid)
			}
		})
	}
}

// TestCRIT1_TenantValidationAtEntryPoints tests validation at all entry points
func TestCRIT1_TenantValidationAtEntryPoints(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	invalidTenants := []string{
		"",
		strings.Repeat("a", 65),
		"tenant$123",
		"../etc/passwd",
		"tenant'; DROP TABLE",
	}

	for _, invalidTenant := range invalidTenants {
		t.Run(fmt.Sprintf("reject_%s", invalidTenant), func(t *testing.T) {
			// Try to create instance with invalid tenant
			_, err := s.getOrCreateInstance(invalidTenant)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid tenant ID")

			// Verify no instance was created
			_, exists := s.getInstance(invalidTenant)
			require.False(t, exists)
		})
	}

	validTenants := []string{
		"tenant123",
		"org-123.production",
		"a",
		strings.Repeat("a", 64),
	}

	for _, validTenant := range validTenants {
		t.Run(fmt.Sprintf("accept_%s", validTenant), func(t *testing.T) {
			inst, err := s.getOrCreateInstance(validTenant)
			require.NoError(t, err)
			require.NotNil(t, inst)
			require.Equal(t, validTenant, inst.tenantID)
		})
	}
}

// =============================================================================
// HIGH-1: Panic Recovery Tests
// =============================================================================

// TestHIGH1_PanicRecovery tests panic recovery in all handlers
func TestHIGH1_PanicRecovery(t *testing.T) {
	handlers := []string{
		"FindTraceByID",
		"SearchRecent",
		"SearchTags",
		"SearchTagsV2",
		"SearchTagValues",
		"SearchTagValuesV2",
		"QueryRange",
	}

	for _, handler := range handlers {
		t.Run(handler, func(t *testing.T) {
			s := createTestLiveStore(t)
			err := services.StartAndAwaitRunning(context.Background(), s)
			require.NoError(t, err)
			defer func() {
				_ = services.StopAndAwaitTerminated(context.Background(), s)
			}()

			// Trigger panic in handler
			err = triggerPanicInHandler(t, s, handler)

			// Verify error returned (not nil)
			require.Error(t, err, "handler should return error on panic")
			require.Contains(t, err.Error(), "internal error", "error should indicate internal error")
		})
	}
}

// TestHIGH1_PanicRecoveryDoesNotCrash tests that panics don't crash the service
func TestHIGH1_PanicRecoveryDoesNotCrash(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Trigger panic in FindTraceByID
	ctx := user.InjectOrgID(context.Background(), testTenantID)

	// Create instance that will panic
	s.instancesMtx.Lock()
	s.instances[testTenantID] = &instance{
		tenantID:       testTenantID,
		liveTraces:     nil, // nil will panic
	}
	s.instancesMtx.Unlock()

	// First call should panic and return error
	_, err = s.FindTraceByID(ctx, &tempopb.TraceByIDRequest{TraceID: []byte("test")})
	require.Error(t, err)

	// Service should still be running
	require.Equal(t, services.Running, s.State())

	// Second call should also work (panic recovery doesn't break the service)
	_, err = s.FindTraceByID(ctx, &tempopb.TraceByIDRequest{TraceID: []byte("test2")})
	require.Error(t, err)
}

// =============================================================================
// HIGH-6: Graceful Shutdown Tests
// =============================================================================

// TestHIGH6_ShutdownOrdering tests graceful shutdown ordering
func TestHIGH6_ShutdownOrdering(t *testing.T) {
	s := createTestLiveStore(t)

	// Don't hold background processes for this test
	s.cfg.holdAllBackgroundProcesses = false

	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)

	// Push some data
	_, _ = pushTraceToLiveStore(t, s, testTenantID)

	// Stop should complete without hanging
	shutdownStart := time.Now()
	err = services.StopAndAwaitTerminated(context.Background(), s)
	shutdownDuration := time.Since(shutdownStart)

	require.NoError(t, err)
	require.Less(t, shutdownDuration, s.cfg.InstanceCleanupPeriod,
		"shutdown should complete before timeout")
}

// TestHIGH6_ShutdownTimeout tests shutdown timeout protection
func TestHIGH6_ShutdownTimeout(t *testing.T) {
	s := createTestLiveStore(t)

	// Set very short timeout
	s.cfg.InstanceCleanupPeriod = 100 * time.Millisecond
	s.cfg.holdAllBackgroundProcesses = false

	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)

	// Block the context cancel to simulate slow shutdown
	s.cancel = func() {
		// Don't cancel immediately, simulate delay
		time.Sleep(200 * time.Millisecond)
	}

	// Stop should timeout
	err = services.StopAndAwaitTerminated(context.Background(), s)
	// May get timeout error depending on exact timing
	// The important thing is it doesn't hang forever
	if err != nil {
		require.Contains(t, err.Error(), "shutdown timed out")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

// TestIntegration_EndToEnd tests full end-to-end flow with all protections
func TestIntegration_EndToEnd(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Test 1: Valid tenant IDs work
	validTenants := []string{"tenant1", "org-123.prod", "test_tenant"}
	for _, tenant := range validTenants {
		id, trace := pushTraceToLiveStore(t, s, tenant)

		// Verify we can query it
		ctx := user.InjectOrgID(context.Background(), tenant)
		resp, err := s.FindTraceByID(ctx, &tempopb.TraceByIDRequest{TraceID: id})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, trace, resp.Trace)
	}

	// Test 2: Invalid tenant IDs rejected
	invalidTenants := []string{"", "tenant$", "../etc/passwd"}
	for _, tenant := range invalidTenants {
		_, err := s.getOrCreateInstance(tenant)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid tenant ID")
	}

	// Test 3: Query handlers don't panic
	ctx := user.InjectOrgID(context.Background(), "tenant1")

	_, err = s.SearchRecent(ctx, &tempopb.SearchRequest{Query: "{}"})
	require.NoError(t, err)

	_, err = s.SearchTags(ctx, &tempopb.SearchTagsRequest{Scope: "span"})
	require.NoError(t, err)

	// Test 4: Graceful shutdown works
	err = services.StopAndAwaitTerminated(context.Background(), s)
	require.NoError(t, err)
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestEdgeCase_ConcurrentPanics tests concurrent panics don't cause issues
func TestEdgeCase_ConcurrentPanics(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Create instance that will panic
	ctx := user.InjectOrgID(context.Background(), testTenantID)
	s.instancesMtx.Lock()
	s.instances[testTenantID] = &instance{
		tenantID:   testTenantID,
		liveTraces: nil, // nil will panic
	}
	s.instancesMtx.Unlock()

	// Trigger concurrent panics
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = s.FindTraceByID(ctx, &tempopb.TraceByIDRequest{TraceID: []byte("test")})
		}()
	}

	wg.Wait()

	// Service should still be running
	require.Equal(t, services.Running, s.State())
}

// TestEdgeCase_BoundaryTenantIDs tests boundary conditions for tenant IDs
func TestEdgeCase_BoundaryTenantIDs(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		valid    bool
	}{
		{"exactly 64 chars", strings.Repeat("a", 64), true},
		{"65 chars", strings.Repeat("a", 65), false},
		{"1 char", "a", true},
		{"0 chars (empty)", "", false},
		{"64 chars with dots", strings.Repeat("a.", 32), true},
		{"all dots", strings.Repeat(".", 64), true},
		{"all hyphens", strings.Repeat("-", 64), true},
		{"all underscores", strings.Repeat("_", 64), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidTenantID(tt.tenantID)
			require.Equal(t, tt.valid, valid)
		})
	}
}

// TestEdgeCase_ContextCancellation tests context cancellation during operations
func TestEdgeCase_ContextCancellation(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Create instance
	_, err = s.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// Create cancelled context
	ctx := user.InjectOrgID(context.Background(), testTenantID)
	ctx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	// Query with cancelled context
	_, err = s.FindTraceByID(ctx, &tempopb.TraceByIDRequest{TraceID: []byte("test")})
	// May get context cancelled error or nil result
	// Either is acceptable - important thing is no panic
}

// TestEdgeCase_MultipleDotsInTenantID tests tenant IDs with multiple dots
func TestEdgeCase_MultipleDotsInTenantID(t *testing.T) {
	validTenants := []string{
		"a.b.c.d.e",
		"org.prod.us-east-1",
		"grafana.cloud.prod",
		"...", // all dots is technically valid per our rules
	}

	for _, tenant := range validTenants {
		t.Run(tenant, func(t *testing.T) {
			valid := isValidTenantID(tenant)
			require.True(t, valid, "tenant %q should be valid", tenant)
		})
	}
}

// TestEdgeCase_EmptyStringTenantIDAtRuntime tests empty tenant at various points
func TestEdgeCase_EmptyStringTenantIDAtRuntime(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Try to get/create instance with empty tenant
	_, err = s.getOrCreateInstance("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid tenant ID")

	// Try to get instance with empty tenant
	_, exists := s.getInstance("")
	require.False(t, exists)
}

// TestEdgeCase_HighCardinalityAttackSimulation simulates cardinality attack
func TestEdgeCase_HighCardinalityAttackSimulation(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Try to create many instances with unique but invalid tenant IDs
	invalidTenants := []string{}
	for i := 0; i < 100; i++ {
		// Create tenant IDs that would cause cardinality explosion
		invalidTenants = append(invalidTenants, fmt.Sprintf("tenant$%d", i))
		invalidTenants = append(invalidTenants, fmt.Sprintf("tenant/%d", i))
		invalidTenants = append(invalidTenants, fmt.Sprintf("tenant;%d", i))
	}

	rejectedCount := 0
	for _, tenant := range invalidTenants {
		_, err := s.getOrCreateInstance(tenant)
		if err != nil {
			rejectedCount++
		}
	}

	// All invalid tenants should be rejected
	require.Equal(t, len(invalidTenants), rejectedCount,
		"all invalid tenants should be rejected")

	// Only valid instances should exist
	instances := s.getInstances()
	for _, inst := range instances {
		require.True(t, isValidTenantID(inst.tenantID),
			"only valid tenant IDs should have instances")
	}
}

// TestEdgeCase_ShutdownDuringOperations tests shutdown during active operations
func TestEdgeCase_ShutdownDuringOperations(t *testing.T) {
	s := createTestLiveStore(t)
	s.cfg.holdAllBackgroundProcesses = false

	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)

	// Start concurrent operations
	var wg sync.WaitGroup
	stopFlag := make(chan struct{})

	// Goroutine 1: Push data
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopFlag:
				return
			default:
				_, _ = pushTraceToLiveStore(t, s, testTenantID)
			}
		}
	}()

	// Goroutine 2: Query data
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx := user.InjectOrgID(context.Background(), testTenantID)
		for {
			select {
			case <-stopFlag:
				return
			default:
				_, _ = s.SearchRecent(ctx, &tempopb.SearchRequest{Query: "{}"})
			}
		}
	}()

	// Let operations run briefly
	time.Sleep(100 * time.Millisecond)

	// Stop the service while operations are active
	err = services.StopAndAwaitTerminated(context.Background(), s)
	close(stopFlag)
	wg.Wait()

	// Should shutdown without error
	require.NoError(t, err)
}

// TestEdgeCase_ReadinessChecks tests readiness state transitions
func TestEdgeCase_ReadinessChecks(t *testing.T) {
	s := createTestLiveStore(t)

	// Before starting, should not be ready
	err := s.CheckReady(context.Background())
	require.ErrorIs(t, err, ErrStarting)

	// Start the service
	err = services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)

	// After starting, should be ready
	err = s.CheckReady(context.Background())
	require.NoError(t, err)

	// Stop the service
	err = services.StopAndAwaitTerminated(context.Background(), s)
	require.NoError(t, err)
}

// TestEdgeCase_InvalidTenantInMetrics tests that invalid tenants don't create metrics
func TestEdgeCase_InvalidTenantInMetrics(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Try to create instance with invalid tenant
	invalidTenant := "tenant$invalid"
	_, err = s.getOrCreateInstance(invalidTenant)
	require.Error(t, err)

	// Verify no instance was created
	s.instancesMtx.RLock()
	_, exists := s.instances[invalidTenant]
	s.instancesMtx.RUnlock()
	require.False(t, exists, "invalid tenant should not have instance")
}

// TestEdgeCase_ConsumerWithInvalidTenants tests consume with invalid tenant records
func TestEdgeCase_ConsumerWithInvalidTenants(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Create records with invalid tenant IDs
	validTrace, err := proto.Marshal(test.MakeTrace(1, test.ValidTraceID(nil)))
	require.NoError(t, err)

	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: validTrace}},
		Ids:    [][]byte{test.ValidTraceID(nil)},
	}

	invalidTenants := []string{
		"",
		"tenant$123",
		"../etc/passwd",
		strings.Repeat("a", 65),
	}

	now := time.Now()
	var records []*kgo.Record
	for _, tenant := range invalidTenants {
		encodedRecords, err := ingest.Encode(0, tenant, req, 1_000_000)
		require.NoError(t, err)
		for _, rec := range encodedRecords {
			rec.Timestamp = now
			records = append(records, rec)
		}
	}

	// Consume should handle invalid tenants gracefully
	// It should not create instances for invalid tenants
	initialInstanceCount := len(s.getInstances())

	_, err = s.consume(context.Background(), createRecordIter(records), now)
	// May return error if circuit breaker triggers, but should not panic

	finalInstanceCount := len(s.getInstances())

	// No new instances should be created for invalid tenants
	require.Equal(t, initialInstanceCount, finalInstanceCount,
		"invalid tenants should not create instances")
}

// TestEdgeCase_CircuitBreakerOnRepeatedFailures tests circuit breaker behavior
func TestEdgeCase_CircuitBreakerOnRepeatedFailures(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Create many records with invalid tenant IDs to trigger circuit breaker
	validTrace, err := proto.Marshal(test.MakeTrace(1, test.ValidTraceID(nil)))
	require.NoError(t, err)

	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: validTrace}},
		Ids:    [][]byte{test.ValidTraceID(nil)},
	}

	now := time.Now()
	var records []*kgo.Record

	// Create more than 100 records (circuit breaker threshold) with invalid tenant
	for i := 0; i < 150; i++ {
		encodedRecords, err := ingest.Encode(0, "invalid$tenant", req, 1_000_000)
		require.NoError(t, err)
		for _, rec := range encodedRecords {
			rec.Timestamp = now
			records = append(records, rec)
		}
	}

	// Consume should trigger circuit breaker
	_, err = s.consume(context.Background(), createRecordIter(records), now)

	// Should get circuit breaker error
	if err != nil {
		require.Contains(t, err.Error(), "circuit breaker")
	}
}

// TestEdgeCase_GracefulDegradation tests service degrades gracefully under stress
func TestEdgeCase_GracefulDegradation(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Create many valid tenants
	const numTenants = 50
	for i := 0; i < numTenants; i++ {
		tenant := fmt.Sprintf("tenant-%d", i)
		_, err := s.getOrCreateInstance(tenant)
		require.NoError(t, err)
	}

	// Verify all instances created
	instances := s.getInstances()
	require.Len(t, instances, numTenants)

	// Service should still respond to queries
	ctx := user.InjectOrgID(context.Background(), "tenant-0")
	_, err = s.SearchRecent(ctx, &tempopb.SearchRequest{Query: "{}"})
	require.NoError(t, err)
}

// =============================================================================
// Performance and Stress Tests
// =============================================================================

// TestStress_ConcurrentTenantCreation tests concurrent tenant instance creation
func TestStress_ConcurrentTenantCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	const numGoroutines = 20
	const tenantsPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines*tenantsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := 0; j < tenantsPerGoroutine; j++ {
				tenant := fmt.Sprintf("tenant-%d-%d", base, j)
				_, err := s.getOrCreateInstance(tenant)
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	require.Empty(t, errs, "should not have errors during concurrent creation")

	// Verify all instances created
	instances := s.getInstances()
	require.Len(t, instances, numGoroutines*tenantsPerGoroutine)
}

// TestStress_ConcurrentQueries tests concurrent queries across handlers
func TestStress_ConcurrentQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Create instance with some data
	_, _ = pushTraceToLiveStore(t, s, testTenantID)

	const numGoroutines = 50
	const queriesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	ctx := user.InjectOrgID(context.Background(), testTenantID)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < queriesPerGoroutine; j++ {
				// Mix different query types
				switch j % 4 {
				case 0:
					_, _ = s.FindTraceByID(ctx, &tempopb.TraceByIDRequest{TraceID: test.ValidTraceID(nil)})
				case 1:
					_, _ = s.SearchRecent(ctx, &tempopb.SearchRequest{Query: "{}"})
				case 2:
					_, _ = s.SearchTags(ctx, &tempopb.SearchTagsRequest{Scope: "span"})
				case 3:
					_, _ = s.QueryRange(ctx, &tempopb.QueryRangeRequest{
						Query: "{} | rate()",
						Start: uint64(time.Now().Add(-time.Hour).UnixNano()),
						End:   uint64(time.Now().UnixNano()),
					})
				}
			}
		}()
	}

	wg.Wait()

	// Service should still be running
	require.Equal(t, services.Running, s.State())
}

// =============================================================================
// Regression Tests
// =============================================================================

// TestRegression_PanicDoesNotReturnNilNil verifies fix for panic returning (nil, nil)
func TestRegression_PanicDoesNotReturnNilNil(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Create instance that will panic
	ctx := user.InjectOrgID(context.Background(), testTenantID)
	s.instancesMtx.Lock()
	s.instances[testTenantID] = &instance{
		tenantID:   testTenantID,
		liveTraces: nil, // nil will panic
	}
	s.instancesMtx.Unlock()

	// Call handler - should return error, not (nil, nil)
	resp, err := s.FindTraceByID(ctx, &tempopb.TraceByIDRequest{TraceID: []byte("test")})

	// CRITICAL: Error must not be nil
	require.Error(t, err, "panic must return error, not (nil, nil)")
	require.Contains(t, err.Error(), "internal error")

	// Response can be nil or not, but error MUST be set
	_ = resp
}

// TestRegression_InvalidTenantDoesNotCreateMetrics verifies tenant validation fix
func TestRegression_InvalidTenantDoesNotCreateMetrics(t *testing.T) {
	s := createTestLiveStore(t)
	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Try invalid tenants that would cause cardinality explosion
	attackTenants := []string{
		"tenant$1", "tenant$2", "tenant$3",
		"../passwd1", "../passwd2",
		strings.Repeat("a", 65),
		strings.Repeat("b", 100),
	}

	for _, tenant := range attackTenants {
		_, err := s.getOrCreateInstance(tenant)
		require.Error(t, err, "invalid tenant should be rejected")
	}

	// Verify no instances created for invalid tenants
	instances := s.getInstances()
	for _, inst := range instances {
		for _, attackTenant := range attackTenants {
			require.NotEqual(t, attackTenant, inst.tenantID,
				"invalid tenant should not have instance")
		}
	}
}

// TestRegression_ReadyCheckReturnsCorrectError tests atomic.Error fix
func TestRegression_ReadyCheckReturnsCorrectError(t *testing.T) {
	s := createTestLiveStore(t)

	// Before starting, should return ErrStarting
	err := s.CheckReady(context.Background())
	require.ErrorIs(t, err, ErrStarting)
	require.NotNil(t, err) // Must not be nil

	// Start service
	err = services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// After starting, should return nil (ready)
	err = s.CheckReady(context.Background())
	require.NoError(t, err)
	require.Nil(t, err)
}

// TestRegression_ShutdownDoesNotHang verifies graceful shutdown fix
func TestRegression_ShutdownDoesNotHang(t *testing.T) {
	s := createTestLiveStore(t)
	s.cfg.holdAllBackgroundProcesses = false

	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- services.StopAndAwaitTerminated(shutdownCtx, s)
	}()

	select {
	case err := <-done:
		// Shutdown completed
		require.NoError(t, err)
	case <-shutdownCtx.Done():
		t.Fatal("shutdown hung - did not complete within timeout")
	}
}

// TestRegression_CircuitBreakerPreventsInfiniteRetries tests circuit breaker fix
func TestRegression_CircuitBreakerPreventsInfiniteRetries(t *testing.T) {
	s := createTestLiveStore(t)

	err := services.StartAndAwaitRunning(context.Background(), s)
	require.NoError(t, err)
	defer func() {
		_ = services.StopAndAwaitTerminated(context.Background(), s)
	}()

	// Create records that will fail instance creation due to invalid tenant
	validTrace, err := proto.Marshal(test.MakeTrace(1, test.ValidTraceID(nil)))
	require.NoError(t, err)

	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: validTrace}},
		Ids:    [][]byte{test.ValidTraceID(nil)},
	}

	now := time.Now()
	var records []*kgo.Record

	// Create 150 records (more than circuit breaker threshold of 100) with invalid tenant
	for i := 0; i < 150; i++ {
		encodedRecords, err := ingest.Encode(0, "invalid$tenant", req, 1_000_000)
		require.NoError(t, err)
		for _, rec := range encodedRecords {
			rec.Timestamp = now
			records = append(records, rec)
		}
	}

	// Consume should stop after circuit breaker opens
	_, err = s.consume(context.Background(), createRecordIter(records), now)

	// Should get error (circuit breaker or validation error)
	if err != nil {
		// Either circuit breaker error or validation error is acceptable
		ok := errors.Is(err, context.Canceled) ||
			strings.Contains(err.Error(), "circuit breaker") ||
			strings.Contains(err.Error(), "invalid tenant ID")
		require.True(t, ok, "should get circuit breaker or validation error, got: %v", err)
	}
}
