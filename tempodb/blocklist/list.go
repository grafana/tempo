package blocklist

import (
	"sync"

	"github.com/google/uuid"

	backend_v1 "github.com/grafana/tempo/tempodb/backend/v1"
)

// PerTenant is a map of tenant ids to backend_v1.BlockMetas
type PerTenant map[string][]*backend_v1.BlockMeta

// PerTenantCompacted is a map of tenant ids to backend_v1.CompactedBlockMetas
type PerTenantCompacted map[string][]*backend_v1.CompactedBlockMeta

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
}

func New() *List {
	return &List{
		metas:          make(PerTenant),
		compactedMetas: make(PerTenantCompacted),

		added:            make(PerTenant),
		removed:          make(PerTenant),
		compactedAdded:   make(PerTenantCompacted),
		compactedRemoved: make(PerTenantCompacted),
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

func (l *List) Metas(tenantID string) []*backend_v1.BlockMeta {
	if tenantID == "" {
		return nil
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	copiedBlocklist := make([]*backend_v1.BlockMeta, 0, len(l.metas[tenantID]))
	copiedBlocklist = append(copiedBlocklist, l.metas[tenantID]...)
	return copiedBlocklist
}

func (l *List) CompactedMetas(tenantID string) []*backend_v1.CompactedBlockMeta {
	if tenantID == "" {
		return nil
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	copiedBlocklist := make([]*backend_v1.CompactedBlockMeta, 0, len(l.compactedMetas[tenantID]))
	copiedBlocklist = append(copiedBlocklist, l.compactedMetas[tenantID]...)

	return copiedBlocklist
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
func (l *List) Update(tenantID string, add []*backend_v1.BlockMeta, remove []*backend_v1.BlockMeta, compactedAdd []*backend_v1.CompactedBlockMeta, compactedRemove []*backend_v1.CompactedBlockMeta) {
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
func (l *List) updateInternal(tenantID string, add []*backend_v1.BlockMeta, remove []*backend_v1.BlockMeta, compactedAdd []*backend_v1.CompactedBlockMeta, compactedRemove []*backend_v1.CompactedBlockMeta) {
	// ******** Regular blocks ********
	blocklist := l.metas[tenantID]

	matchedRemovals := make(map[uuid.UUID]struct{})
	for _, b := range blocklist {
		for _, rem := range remove {
			if b.BlockID == rem.BlockID {
				matchedRemovals[rem.BlockID] = struct{}{}
				break
			}
		}
	}

	existingMetas := make(map[uuid.UUID]struct{})
	newblocklist := make([]*backend_v1.BlockMeta, 0, len(blocklist)-len(matchedRemovals)+len(add))
	// rebuild the blocklist dropping all removals
	for _, b := range blocklist {
		existingMetas[b.BlockID] = struct{}{}
		if _, ok := matchedRemovals[b.BlockID]; !ok {
			newblocklist = append(newblocklist, b)
		}
	}
	// add new blocks (only if they don't already exist)
	for _, b := range add {
		if _, ok := existingMetas[b.BlockID]; !ok {
			newblocklist = append(newblocklist, b)
		}
	}

	l.metas[tenantID] = newblocklist

	// ******** Compacted blocks ********
	compactedBlocklist := l.compactedMetas[tenantID]

	compactedRemovals := map[uuid.UUID]struct{}{}
	for _, c := range compactedBlocklist {
		for _, rem := range compactedRemove {
			if c.BlockID == rem.BlockID {
				compactedRemovals[rem.BlockID] = struct{}{}
				break
			}
		}
	}

	existingMetas = make(map[uuid.UUID]struct{})
	newCompactedBlocklist := make([]*backend_v1.CompactedBlockMeta, 0, len(compactedBlocklist)-len(compactedRemovals)+len(compactedAdd))
	// rebuild the blocklist dropping all removals
	for _, b := range compactedBlocklist {
		existingMetas[b.BlockID] = struct{}{}
		if _, ok := compactedRemovals[b.BlockID]; !ok {
			newCompactedBlocklist = append(newCompactedBlocklist, b)
		}
	}
	for _, b := range compactedAdd {
		if _, ok := existingMetas[b.BlockID]; !ok {
			newCompactedBlocklist = append(newCompactedBlocklist, b)
		}
	}

	l.compactedMetas[tenantID] = newCompactedBlocklist
}
