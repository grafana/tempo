package lx

import (
	"strings"
	"sync"
	"sync/atomic"
)

// namespaceRule stores the cached result of Enabled.
type namespaceRule struct {
	isEnabledByRule  bool
	isDisabledByRule bool
	generation       uint64 // NEW: track cache validity
}

// Namespace manages thread-safe namespace enable/disable states with caching.
// The store holds explicit user-defined rules (path -> bool).
// The cache holds computed effective states for paths (path -> namespaceRule)
// based on hierarchical rules to optimize lookups.
type Namespace struct {
	store      sync.Map // path (string) -> rule (bool)
	cache      sync.Map // path (string) -> namespaceRule
	genCounter uint64   // NEW: atomic generation counter
}

// Set defines an explicit enable/disable rule for a namespace path.
// It clears the cache to ensure subsequent lookups reflect the change.
func (ns *Namespace) Set(path string, enabled bool) {
	ns.store.Store(path, enabled)
	ns.invalidatePathCache(path)
}

// invalidatePathCache increments generation counter instead of scanning cache.
func (ns *Namespace) invalidatePathCache(path string) {
	// Atomic increment - O(1), no lock contention on cache
	atomic.AddUint64(&ns.genCounter, 1)
}

// Store directly sets a rule in the store, bypassing cache invalidation.
// Intended for internal use or sync.Map parity; prefer Set for standard use.
func (ns *Namespace) Store(path string, rule bool) {
	ns.store.Store(path, rule)
}

// clearCache clears the cache of Enabled results.
// Called by Set to ensure consistency after rule changes.
func (ns *Namespace) clearCache() {
	ns.cache.Range(func(key, _ interface{}) bool {
		ns.cache.Delete(key)
		return true
	})
}

// Enabled checks if a path is enabled by namespace rules, considering the most
// specific rule (path or closest prefix) in the store. Results are cached.
// Args:
//   - path: Absolute namespace path to check.
//   - separator: Character delimiting path segments (e.g., "/", ".").
//
// Returns:
//   - isEnabledByRule: True if an explicit rule enables the path.
//   - isDisabledByRule: True if an explicit rule disables the path.
//
// If both are false, no explicit rule applies to the path or its prefixes.
func (ns *Namespace) Enabled(path string, separator string) (isEnabledByRule bool, isDisabledByRule bool) {
	if path == "" {
		return false, false
	}

	// Check cache with generation validation
	if cachedValue, found := ns.cache.Load(path); found {
		if state, ok := cachedValue.(namespaceRule); ok {
			// If cache generation matches current, result is valid
			if state.generation == atomic.LoadUint64(&ns.genCounter) {
				return state.isEnabledByRule, state.isDisabledByRule
			}
			// Stale cache - fall through to recompute
			ns.cache.Delete(path)
		}
	}

	// Compute: Most specific rule wins (original logic)
	parts := strings.Split(path, separator)
	computedIsEnabled := false
	computedIsDisabled := false
	for i := len(parts); i >= 1; i-- {
		currentPrefix := strings.Join(parts[:i], separator)
		if val, ok := ns.store.Load(currentPrefix); ok {
			if rule := val.(bool); rule {
				computedIsEnabled = true
				computedIsDisabled = false
			} else {
				computedIsEnabled = false
				computedIsDisabled = true
			}
			break
		}
	}

	// Cache result with current generation
	ns.cache.Store(path, namespaceRule{
		isEnabledByRule:  computedIsEnabled,
		isDisabledByRule: computedIsDisabled,
		generation:       atomic.LoadUint64(&ns.genCounter),
	})
	return computedIsEnabled, computedIsDisabled
}
