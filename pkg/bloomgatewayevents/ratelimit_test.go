package bloomgatewayevents

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantRateLimiter_ZeroMeansUnlimited proves a tenant with no configured
// rate (limits returns 0, the override's default) is never throttled and --
// just as importantly -- never gets a *rate.Limiter allocated for it: a cell
// with thousands of tenants that never configure this override must not pay
// a per-tenant map entry for it.
func TestTenantRateLimiter_ZeroMeansUnlimited(t *testing.T) {
	l := newTenantRateLimiter(func(string) float64 { return 0 })

	for i := 0; i < 1000; i++ {
		require.True(t, l.allow("tenant-a"))
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	assert.Empty(t, l.limiters, "an unlimited tenant must never get a cached limiter")
}

// TestTenantRateLimiter_EnforcesRate locks in the one-token-per-publish
// contract at the smallest useful rate: burst=max(1, ceil(1))=1, so the
// first call consumes the only token and an immediate second call must be
// denied.
func TestTenantRateLimiter_EnforcesRate(t *testing.T) {
	l := newTenantRateLimiter(func(string) float64 { return 1 })

	assert.True(t, l.allow("tenant-a"), "first publish must be allowed")
	assert.False(t, l.allow("tenant-a"), "second immediate publish must be denied")
}

// TestTenantRateLimiter_PerTenantIsolation proves one tenant exhausting its
// own budget can never throttle another: limiters are keyed per tenant.
func TestTenantRateLimiter_PerTenantIsolation(t *testing.T) {
	l := newTenantRateLimiter(func(string) float64 { return 1 })

	require.True(t, l.allow("tenant-a"))
	require.False(t, l.allow("tenant-a"), "tenant-a must now be exhausted")

	assert.True(t, l.allow("tenant-b"), "tenant-b must be unaffected by tenant-a's exhaustion")
}

// TestTenantRateLimiter_LiveLimitChange proves runtime-overrides reload
// works without a watcher: allow re-reads limits(tenantID) on every call, so
// a changed value takes effect on the very next publish -- including
// dropping to unlimited and coming back.
func TestTenantRateLimiter_LiveLimitChange(t *testing.T) {
	var mu sync.Mutex
	configured := 1.0
	l := newTenantRateLimiter(func(string) float64 {
		mu.Lock()
		defer mu.Unlock()
		return configured
	})

	require.True(t, l.allow("tenant-a"), "first publish at rate 1 must be allowed")
	require.False(t, l.allow("tenant-a"), "must be exhausted at rate 1")

	mu.Lock()
	configured = 0
	mu.Unlock()
	assert.True(t, l.allow("tenant-a"), "switching to 0 (unlimited) must allow immediately, even though the rate-1 limiter was exhausted")

	mu.Lock()
	configured = 1
	mu.Unlock()
	assert.False(t, l.allow("tenant-a"), "switching back to the same rate must resume limiting from where it left off, not grant a fresh burst")
}

// TestTenantRateLimiter_ConcurrentAccess is the -race assertion itself: the
// mutex guarding limiters must make concurrent allow calls from many
// goroutines (as PublishAdd/PublishDelete see under real concurrency from
// block-builder and backend-worker) safe.
func TestTenantRateLimiter_ConcurrentAccess(_ *testing.T) {
	l := newTenantRateLimiter(func(string) float64 { return 1000 })

	tenants := []string{"tenant-a", "tenant-b", "tenant-c"}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			tenant := tenants[n%len(tenants)]
			for j := 0; j < 20; j++ {
				l.allow(tenant)
			}
		}(i)
	}
	wg.Wait()
}
