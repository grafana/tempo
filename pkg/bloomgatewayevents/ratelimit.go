package bloomgatewayevents

import (
	"math"
	"sync"

	"golang.org/x/time/rate"
)

// TenantLimits returns tenantID's configured bloom-gateway publish rate in
// publishes per second, with 0 meaning unlimited. This package must not
// import modules/overrides (pkg/* may not depend on modules/*), so
// publisher.go's WithTenantLimits Option takes this getter instead --
// callers hand in a method value, typically
// overrides.Interface.BloomGatewayPublishesPerSecond, which already matches
// this signature.
type TenantLimits func(tenantID string) float64

// tenantRateLimiter enforces DESIGN.md § Multi-tenant cells's producer-side
// guardrail: one token per PUBLISH OPERATION -- one Add for an entire block
// regardless of how many chunks it was split into, or one Delete -- never
// one token per record/chunk. A token-per-chunk scheme with a small burst
// would permanently starve large compacted blocks (~30 chunks, per
// DESIGN.md's reference chunking); per-block tokens can never starve
// anyone, and this matches DESIGN.md's own operational unit ("Add (publish)
// per new block").
type tenantRateLimiter struct {
	limits TenantLimits

	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

// newTenantRateLimiter wraps limits. limits == nil (the zero value used
// when no WithTenantLimits Option is supplied) is treated as "every tenant
// unlimited", so allow never needs a nil check at its call sites.
func newTenantRateLimiter(limits TenantLimits) *tenantRateLimiter {
	return &tenantRateLimiter{
		limits:   limits,
		limiters: make(map[string]*rate.Limiter),
	}
}

// allow reports whether tenantID may perform one publish operation now,
// consuming one token from its budget if so.
func (l *tenantRateLimiter) allow(tenantID string) bool {
	if l.limits == nil {
		return true
	}

	r := l.limits(tenantID)
	if r <= 0 {
		// 0 (or a misconfigured negative) means unlimited: never allocate a
		// limiter for a tenant that has none configured, so a cell with
		// thousands of tenants and no rate limiting in use costs this
		// package nothing per-tenant.
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	lim, ok := l.limiters[tenantID]
	if !ok || lim.Limit() != rate.Limit(r) {
		// First publish from this tenant, or its configured rate changed
		// since the last one (a runtime-overrides reload): build a fresh
		// limiter rather than mutating the existing one's rate, so a rate
		// CHANGE always starts from a full burst instead of inheriting
		// however depleted the previous rate's bucket happened to be.
		// Comparing against the limiter's own Limit() -- rather than
		// tracking the configured rate separately -- is what lets a reload
		// be detected on every call with no separate watcher.
		lim = rate.NewLimiter(rate.Limit(r), max(1, int(math.Ceil(r))))
		l.limiters[tenantID] = lim
	}

	return lim.Allow()
}
