package blocklist

import (
	"sync"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

// PerTenant is a map of tenant ids to backend.BlockMetas
type PerTenant map[string][]*backend.BlockMeta

// PerTenantCompacted is a map of tenant ids to backend.CompactedBlockMetas
type PerTenantCompacted map[string][]*backend.CompactedBlockMeta

// List controls access to a per tenant blocklist and compacted blocklist
type List struct {
	mtx            sync.Mutex
	metas          PerTenant
	compactedMetas PerTenantCompacted

	// used by the compactor to track local changes it is aware of
	added          PerTenant
	removed        PerTenant
	compactedAdded PerTenantCompacted
}

func New() *List {
	return &List{
		metas:          make(PerTenant),
		compactedMetas: make(PerTenantCompacted),

		added:          make(PerTenant),
		removed:        make(PerTenant),
		compactedAdded: make(PerTenantCompacted),
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
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if tenantID == "" {
		return nil
	}

	copiedBlocklist := make([]*backend.BlockMeta, 0, len(l.metas[tenantID]))
	copiedBlocklist = append(copiedBlocklist, l.metas[tenantID]...)
	return copiedBlocklist
}

func (l *List) CompactedMetas(tenantID string) []*backend.CompactedBlockMeta {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if tenantID == "" {
		return nil
	}

	copiedBlocklist := make([]*backend.CompactedBlockMeta, 0, len(l.compactedMetas[tenantID]))
	copiedBlocklist = append(copiedBlocklist, l.compactedMetas[tenantID]...)

	return copiedBlocklist
}

func (l *List) ApplyPollResults(m PerTenant, c PerTenantCompacted) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	l.metas = m
	l.compactedMetas = c

	// now reapply all updates and clear
	for tenantID := range l.added {
		l.updateInternal(tenantID, l.added[tenantID], l.removed[tenantID], l.compactedAdded[tenantID])
	}

	l.added = make(PerTenant)
	l.removed = make(PerTenant)
	l.compactedAdded = make(PerTenantCompacted)
}

// Update Adds and removes regular or compacted blocks from the in-memory blocklist.
// Changes are temporary and will be preserved only for one poll (jpe actually do this)
func (l *List) Update(tenantID string, add []*backend.BlockMeta, remove []*backend.BlockMeta, compactedAdd []*backend.CompactedBlockMeta) {
	if tenantID == "" {
		return
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	l.updateInternal(tenantID, add, remove, compactedAdd)

	// save off
	l.added[tenantID] = append(l.added[tenantID], add...)
	l.removed[tenantID] = append(l.removed[tenantID], remove...)
	l.compactedAdded[tenantID] = append(l.compactedAdded[tenantID], compactedAdd...)
}

// jpe comment
func (l *List) updateInternal(tenantID string, add []*backend.BlockMeta, remove []*backend.BlockMeta, compactedAdd []*backend.CompactedBlockMeta) {
	// ******** Regular blocks ********
	blocklist := l.metas[tenantID]

	matchedRemovals := make(map[uuid.UUID]struct{})
	for _, b := range blocklist {
		for _, rem := range remove {
			if b.BlockID == rem.BlockID {
				matchedRemovals[rem.BlockID] = struct{}{}
			}
		}
	}

	newblocklist := make([]*backend.BlockMeta, 0, len(blocklist)-len(matchedRemovals)+len(add))
	for _, b := range blocklist {
		if _, ok := matchedRemovals[b.BlockID]; !ok {
			newblocklist = append(newblocklist, b)
		}
	}
	newblocklist = append(newblocklist, add...)
	l.metas[tenantID] = newblocklist

	// ******** Compacted blocks ********
	l.compactedMetas[tenantID] = append(l.compactedMetas[tenantID], compactedAdd...)
}
