package tenantselector

var _ TenantSelector = (*BlockListWeightedTenantSelector)(nil)

type BlockListWeightedTenantSelector struct {
	tenants map[string]Tenant
}

func NewBlockListWeightedTenantSelector(tenants []Tenant) *BlockListWeightedTenantSelector {
	s := &BlockListWeightedTenantSelector{}

	s.tenants = make(map[string]Tenant, len(tenants))

	for _, t := range tenants {
		s.tenants[t.ID] = t
	}

	return s
}

func (s *BlockListWeightedTenantSelector) PriorityForTenant(tenantID string) int {
	var (
		length      = s.tenants[tenantID].BlocklistLength
		outstanding = s.tenants[tenantID].OutstandingBlocklistLength
	)

	if length == 0 || outstanding == 0 {
		return 0
	}

	return length + outstanding
}
