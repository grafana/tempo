package blocklist

import (
	"slices"
	"sync"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

// PerTenant is a map of tenant ids to backend.BlockMetas
type PerTenant map[string][]*backend.BlockMeta

// PerTenantCompacted is a map of tenant ids to backend.CompactedBlockMetas
type PerTenantCompacted map[string][]*backend.CompactedBlockMeta

// PerTenantLastCompacted is a map of tenant ids to the last time a tenant was compacted
type PerTenantLastCompacted map[string]time.Time

// List controls access to a per tenant blocklist and compacted blocklist
type List struct {
	mtx            sync.Mutex
	metas          PerTenant
	compactedMetas PerTenantCompacted

	// used by the compactor to track local changes it is aware of
	added            PerTenant
	removed          PerTenant
	compactedAdded   PerTenantCompacted
	compactedRemoved PerTenantCompacted

	// used to track the last time a tenant was compacted
	lastCompacted PerTenantLastCompacted
}

func New() *List {
	return &List{
		metas:          make(PerTenant),
		compactedMetas: make(PerTenantCompacted),

		added:            make(PerTenant),
		removed:          make(PerTenant),
		compactedAdded:   make(PerTenantCompacted),
		compactedRemoved: make(PerTenantCompacted),

		lastCompacted: make(PerTenantLastCompacted),
	}
}

// Tenants returns a slice of tenant ids with metas (compacted metas are ignored.)
func (l *List) Tenants() []string {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	tenants := make([]string, 0, len(l.metas))
	for tenant := range l.metas {
		tenants = append(tenants, tenant)
	}

	return tenants
}

func (l *List) Metas(tenantID string) []*backend.BlockMeta {
	if tenantID == "" {
		return nil
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	copiedBlocklist := make([]*backend.BlockMeta, 0, len(l.metas[tenantID]))
	copiedBlocklist = append(copiedBlocklist, l.metas[tenantID]...)
	return copiedBlocklist
}

func (l *List) CompactedMetas(tenantID string) []*backend.CompactedBlockMeta {
	if tenantID == "" {
		return nil
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	copiedBlocklist := make([]*backend.CompactedBlockMeta, 0, len(l.compactedMetas[tenantID]))
	copiedBlocklist = append(copiedBlocklist, l.compactedMetas[tenantID]...)

	return copiedBlocklist
}

func (l *List) LastCompacted(tenantID string) *time.Time {
	if tenantID == "" {
		return nil
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	if t, ok := l.lastCompacted[tenantID]; ok {
		return &t
	}

	return nil
}

// ApplyPollResults applies the PerTenant and PerTenantCompacted maps to this blocklist
// Note that it also applies any known local changes and then wipes them out to be restored
// in the next polling cycle.
func (l *List) ApplyPollResults(m PerTenant, c PerTenantCompacted) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	l.metas = m
	l.compactedMetas = c

	// now reapply all updates and clear
	for tenantID := range l.added {
		l.updateInternal(tenantID, l.added[tenantID], l.removed[tenantID], l.compactedAdded[tenantID], l.compactedRemoved[tenantID])
	}

	clear(l.added)
	clear(l.removed)
	clear(l.compactedAdded)
	clear(l.compactedRemoved)
}

// Update Adds and removes regular or compacted blocks from the in-memory blocklist.
// Changes are temporary and will be preserved only for one poll
func (l *List) Update(tenantID string, add []*backend.BlockMeta, remove []*backend.BlockMeta, compactedAdd []*backend.CompactedBlockMeta, compactedRemove []*backend.CompactedBlockMeta) {
	if tenantID == "" {
		return
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	l.updateInternal(tenantID, add, remove, compactedAdd, compactedRemove)

	// We have updated the current blocklist, but we may be in the middle of a
	// polling cycle.  When the Apply is called above, we will have lost the
	// changes that we have just added. So we keep track of them here and apply
	// them again after the Apply to save them for the next polling cycle.  On
	// the next polling cycle, the changes here will rediscovered.
	l.added[tenantID] = append(l.added[tenantID], add...)
	l.removed[tenantID] = append(l.removed[tenantID], remove...)
	l.compactedAdded[tenantID] = append(l.compactedAdded[tenantID], compactedAdd...)
	l.compactedRemoved[tenantID] = append(l.compactedRemoved[tenantID], compactedRemove...)
}

// updateInternal exists to do the work of applying updates to held PerTenant and PerTenantCompacted maps
// it must be called under lock
func (l *List) updateInternal(tenantID string, add []*backend.BlockMeta, remove []*backend.BlockMeta, compactedAdd []*backend.CompactedBlockMeta, compactedRemove []*backend.CompactedBlockMeta) {
	hasID := func(id backend.UUID) func(*backend.BlockMeta) bool {
		return func(b *backend.BlockMeta) bool {
			return b.BlockID == id
		}
	}

	hasIDC := func(id backend.UUID) func(*backend.CompactedBlockMeta) bool {
		return func(b *backend.CompactedBlockMeta) bool {
			return b.BlockID == id
		}
	}

	// ******** Regular blocks ********
	if len(add) > 0 || len(remove) > 0 || len(compactedAdd) > 0 || len(compactedRemove) > 0 {
		var (
			existing = l.metas[tenantID]
			final    = make([]*backend.BlockMeta, 0, max(0, len(existing)+len(add)-len(remove)))
		)

		// rebuild dropping all removals
		for _, b := range existing {
			if slices.ContainsFunc(remove, hasID(b.BlockID)) {
				continue
			}
			final = append(final, b)
		}
		// add new if they don't already exist and weren't also removed
		for _, b := range add {
			if slices.ContainsFunc(final, hasID(b.BlockID)) ||
				slices.ContainsFunc(remove, hasID(b.BlockID)) ||
				slices.ContainsFunc(compactedAdd, hasIDC(b.BlockID)) ||
				slices.ContainsFunc(compactedRemove, hasIDC(b.BlockID)) {
				continue
			}

			final = append(final, b)
		}

		l.metas[tenantID] = final
	}

	// ******** Compacted blocks ********
	if len(compactedAdd) > 0 || len(compactedRemove) > 0 {
		var (
			existing = l.compactedMetas[tenantID]
			final    = make([]*backend.CompactedBlockMeta, 0, max(0, len(existing)+len(compactedAdd)-len(compactedRemove)))
		)
		l.lastCompacted[tenantID] = time.Now()

		// rebuild dropping all removals
		for _, b := range existing {
			if slices.ContainsFunc(compactedRemove, hasIDC(b.BlockID)) {
				continue
			}
			final = append(final, b)
		}
		// add new if they don't already exist and weren't also removed
		for _, b := range compactedAdd {
			if slices.ContainsFunc(existing, hasIDC(b.BlockID)) ||
				slices.ContainsFunc(compactedRemove, hasIDC(b.BlockID)) {
				continue
			}
			final = append(final, b)
		}

		l.compactedMetas[tenantID] = final
	}
}
