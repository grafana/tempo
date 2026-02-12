package livestore

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsValidTenantID_ValidTenantIDs tests valid tenant ID formats
func TestIsValidTenantID_ValidTenantIDs(t *testing.T) {
	testCases := []struct {
		name     string
		tenantID string
	}{
		{"single char lower", "a"},
		{"single char upper", "A"},
		{"single char digit", "0"},
		{"simple alphanumeric", "tenant1"},
		{"all lowercase", "tenant"},
		{"all uppercase", "TENANT"},
		{"mixed case", "TenantID"},
		{"with hyphen", "tenant-1"},
		{"with underscore", "tenant_1"},
		{"with dot", "tenant.1"},
		{"multiple hyphens", "my-tenant-id"},
		{"multiple underscores", "my_tenant_id"},
		{"multiple dots", "org.team.service"},
		{"grafana cloud format", "org-123.production"},
		{"complex valid", "dev-team-a.prod"},
		{"all special chars allowed", "a-b_c.d"},
		{"max length 64", strings.Repeat("a", 64)},
		{"alphanumeric with all separators", "tenant-123_prod.v2"},
		{"starts with digit", "123tenant"},
		{"ends with digit", "tenant123"},
		{"starts with hyphen is invalid but tested separately", ""},
		{"ends with hyphen", "tenant-"},
		{"ends with underscore", "tenant_"},
		{"ends with dot", "tenant."},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tenantID == "" {
				t.Skip("empty test case")
				return
			}
			valid := isValidTenantID(tc.tenantID)
			require.True(t, valid, "expected %q to be valid", tc.tenantID)
		})
	}
}

// TestIsValidTenantID_InvalidTenantIDs tests invalid tenant ID formats
func TestIsValidTenantID_InvalidTenantIDs(t *testing.T) {
	testCases := []struct {
		name     string
		tenantID string
		reason   string
	}{
		{"empty string", "", "zero length"},
		{"too long", strings.Repeat("a", 65), "exceeds 64 chars"},
		{"with space", "tenant 1", "contains space"},
		{"with slash", "tenant/1", "contains slash"},
		{"with backslash", "tenant\\1", "contains backslash"},
		{"with dollar sign", "tenant$", "contains dollar sign"},
		{"with percent", "tenant%", "contains percent"},
		{"with ampersand", "tenant&", "contains ampersand"},
		{"with asterisk", "tenant*", "contains asterisk"},
		{"with plus", "tenant+", "contains plus"},
		{"with equals", "tenant=", "contains equals"},
		{"with brackets", "tenant[1]", "contains brackets"},
		{"with braces", "tenant{1}", "contains braces"},
		{"with parentheses", "tenant(1)", "contains parentheses"},
		{"with semicolon", "tenant;", "contains semicolon"},
		{"with colon", "tenant:", "contains colon"},
		{"with quote", "tenant'", "contains single quote"},
		{"with double quote", "tenant\"", "contains double quote"},
		{"with comma", "tenant,1", "contains comma"},
		{"with less than", "tenant<1", "contains less than"},
		{"with greater than", "tenant>1", "contains greater than"},
		{"with question mark", "tenant?", "contains question mark"},
		{"with exclamation", "tenant!", "contains exclamation"},
		{"with at sign", "tenant@", "contains at sign"},
		{"with hash", "tenant#", "contains hash"},
		{"with caret", "tenant^", "contains caret"},
		{"with tilde", "tenant~", "contains tilde"},
		{"with backtick", "tenant`", "contains backtick"},
		{"with pipe", "tenant|", "contains pipe"},
		{"path traversal attempt", "../etc/passwd", "path traversal"},
		{"path traversal with tenant", "tenant/../admin", "path traversal"},
		{"windows path traversal", "..\\windows\\system32", "windows path traversal"},
		{"null byte attempt", "tenant\x00", "null byte"},
		{"unicode char", "tenant™", "unicode character"},
		{"emoji", "tenant😀", "emoji"},
		{"newline", "tenant\n", "newline"},
		{"tab", "tenant\t", "tab character"},
		{"carriage return", "tenant\r", "carriage return"},
		{"multiple spaces", "tenant  1", "multiple spaces"},
		{"leading space", " tenant", "leading space"},
		{"trailing space", "tenant ", "trailing space"},
		{"only special chars", "!@#$%", "only special chars"},
		{"sql injection attempt", "tenant'; DROP TABLE users--", "sql injection"},
		{"command injection attempt", "tenant; rm -rf /", "command injection"},
		{"xss attempt", "tenant<script>alert(1)</script>", "xss attempt"},
		{"very long with special chars", strings.Repeat("a", 100) + "!@#", "too long with special chars"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid := isValidTenantID(tc.tenantID)
			require.False(t, valid, "expected %q to be invalid: %s", tc.tenantID, tc.reason)
		})
	}
}

// TestIsValidTenantID_BoundaryConditions tests boundary conditions
func TestIsValidTenantID_BoundaryConditions(t *testing.T) {
	testCases := []struct {
		name     string
		tenantID string
		expected bool
	}{
		{"exactly 1 char", "a", true},
		{"exactly 63 chars", strings.Repeat("a", 63), true},
		{"exactly 64 chars", strings.Repeat("a", 64), true},
		{"exactly 65 chars", strings.Repeat("a", 65), false},
		{"exactly 100 chars", strings.Repeat("a", 100), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid := isValidTenantID(tc.tenantID)
			require.Equal(t, tc.expected, valid, "tenant ID: %q", tc.tenantID)
		})
	}
}

// TestGetOrCreateInstance_ValidatesValidTenantID tests that valid tenant IDs work
func TestGetOrCreateInstance_ValidatesValidTenantID(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	validTenants := []string{
		"tenant1",
		"org-123",
		"team_a",
		"org.prod",
		"dev-team-1.staging",
	}

	for _, tenant := range validTenants {
		t.Run(tenant, func(t *testing.T) {
			inst, err := liveStore.getOrCreateInstance(tenant)
			require.NoError(t, err, "valid tenant ID should not error: %s", tenant)
			require.NotNil(t, inst, "instance should be created")
			require.Equal(t, tenant, inst.tenantID)
		})
	}
}

// TestGetOrCreateInstance_RejectsInvalidTenantID tests that invalid tenant IDs are rejected
func TestGetOrCreateInstance_RejectsInvalidTenantID(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	invalidTenants := []string{
		"",
		"tenant with spaces",
		"tenant/path",
		"tenant$special",
		strings.Repeat("a", 65),
		"../etc/passwd",
		"tenant;rm -rf /",
	}

	for _, tenant := range invalidTenants {
		t.Run(fmt.Sprintf("reject_%s", tenant), func(t *testing.T) {
			inst, err := liveStore.getOrCreateInstance(tenant)
			require.Error(t, err, "invalid tenant ID should error: %q", tenant)
			require.Nil(t, inst, "instance should not be created for invalid tenant")
			require.Contains(t, err.Error(), "invalid tenant ID", "error message should mention invalid tenant ID")
		})
	}
}

// TestCircuitBreaker_ThresholdBehavior tests the circuit breaker threshold
func TestCircuitBreaker_ThresholdBehavior(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// The circuit breaker should allow up to invalidTenantLimit invalid attempts
	// After that, it should reject all requests (even valid ones)
	limit := liveStore.invalidTenantLimit

	// Try invalid tenant IDs up to the limit
	for i := int64(0); i < limit; i++ {
		invalidTenant := fmt.Sprintf("invalid/tenant/%d", i)
		inst, err := liveStore.getOrCreateInstance(invalidTenant)
		require.Error(t, err, "iteration %d: invalid tenant should error", i)
		require.Nil(t, inst)
		require.Contains(t, err.Error(), "invalid tenant ID")
	}

	// Count should be exactly at limit
	require.Equal(t, limit, liveStore.invalidTenantCount.Load())

	// One more invalid attempt should trigger circuit breaker
	inst, err := liveStore.getOrCreateInstance("another/invalid")
	require.Error(t, err)
	require.Nil(t, inst)
	require.Contains(t, err.Error(), "too many invalid tenant IDs")
	require.Contains(t, err.Error(), "possible attack")

	// Even valid tenant IDs should now be rejected (circuit is open)
	inst, err = liveStore.getOrCreateInstance("validtenant")
	require.Error(t, err)
	require.Nil(t, inst)
	require.Contains(t, err.Error(), "too many invalid tenant IDs")
}

// TestCircuitBreaker_ConcurrentInvalidRequests tests circuit breaker under concurrent load
func TestCircuitBreaker_ConcurrentInvalidRequests(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// Simulate attack: many goroutines sending invalid tenant IDs
	var wg sync.WaitGroup
	numGoroutines := 50
	attemptsPerGoroutine := 100

	var circuitBreakerTriggered atomic.Bool

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < attemptsPerGoroutine; j++ {
				invalidTenant := fmt.Sprintf("attack/tenant/%d/%d", id, j)
				_, err := liveStore.getOrCreateInstance(invalidTenant)

				// Check if circuit breaker was triggered
				if err != nil && strings.Contains(err.Error(), "too many invalid tenant IDs") {
					circuitBreakerTriggered.Store(true)
					return // Stop this goroutine
				}
			}
		}(i)
	}

	wg.Wait()

	// Circuit breaker should have been triggered
	require.True(t, circuitBreakerTriggered.Load(), "circuit breaker should have been triggered during attack")

	// Count should be at or above limit
	require.GreaterOrEqual(t, liveStore.invalidTenantCount.Load(), liveStore.invalidTenantLimit)
}

// TestCircuitBreaker_AllowsValidBeforeLimit tests that valid tenants work before hitting limit
func TestCircuitBreaker_AllowsValidBeforeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// Mix of valid and invalid requests, staying under limit
	halfLimit := liveStore.invalidTenantLimit / 2

	// Send some invalid requests
	for i := int64(0); i < halfLimit; i++ {
		invalidTenant := fmt.Sprintf("invalid/tenant/%d", i)
		inst, err := liveStore.getOrCreateInstance(invalidTenant)
		require.Error(t, err)
		require.Nil(t, inst)
	}

	// Valid requests should still work
	validTenants := []string{"tenant1", "tenant2", "tenant3"}
	for _, tenant := range validTenants {
		inst, err := liveStore.getOrCreateInstance(tenant)
		require.NoError(t, err, "valid tenant should work before circuit breaker trips")
		require.NotNil(t, inst)
		require.Equal(t, tenant, inst.tenantID)
	}

	// Invalid count should be at halfLimit
	require.Equal(t, halfLimit, liveStore.invalidTenantCount.Load())
}

// TestCircuitBreaker_InitializedCorrectly tests that circuit breaker is initialized
func TestCircuitBreaker_InitializedCorrectly(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// Check that circuit breaker fields are initialized
	require.NotNil(t, liveStore.invalidTenantCount)
	require.Equal(t, int64(0), liveStore.invalidTenantCount.Load(), "initial count should be 0")
	require.Equal(t, int64(1000), liveStore.invalidTenantLimit, "limit should be 1000")
}

// TestIsValidTenantID_CharacterCoverage ensures all allowed characters work
func TestIsValidTenantID_CharacterCoverage(t *testing.T) {
	// Test all allowed character ranges
	testCases := []struct {
		name     string
		tenantID string
		expected bool
	}{
		// Lowercase letters
		{"lowercase a-z", "abcdefghijklmnopqrstuvwxyz", true},
		// Uppercase letters
		{"uppercase A-Z", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", true},
		// Digits
		{"digits 0-9", "0123456789", true},
		// Hyphens
		{"hyphens", "tenant-with-hyphens", true},
		// Underscores
		{"underscores", "tenant_with_underscores", true},
		// Dots
		{"dots", "tenant.with.dots", true},
		// All allowed together
		{"all allowed", "aA0-_.", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid := isValidTenantID(tc.tenantID)
			require.Equal(t, tc.expected, valid, "tenant ID: %q", tc.tenantID)
		})
	}
}

// TestGetOrCreateInstance_PathTraversalAttempts tests specific path traversal patterns
func TestGetOrCreateInstance_PathTraversalAttempts(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	pathTraversalAttempts := []string{
		"../etc/passwd",
		"../../etc/passwd",
		"tenant/../admin",
		"./tenant",
		"..\\windows\\system32",
		"tenant\\..\\admin",
		// Note: ".." and "." alone are blocked by having / or \ which makes them dangerous
		// But as standalone dots they're technically valid characters
	}

	for _, attempt := range pathTraversalAttempts {
		t.Run(fmt.Sprintf("block_%s", attempt), func(t *testing.T) {
			inst, err := liveStore.getOrCreateInstance(attempt)
			require.Error(t, err, "path traversal attempt should be blocked: %q", attempt)
			require.Nil(t, inst)
		})
	}
}
