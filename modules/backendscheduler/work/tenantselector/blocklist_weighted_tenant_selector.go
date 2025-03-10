package tenantselector

import (
	"time"
)

var _ TenantSelector = (*BlockListWeightedTenantSelector)(nil)

type BlockListWeightedTenantSelector struct {
	tenants map[string]Tenant
}

const (
	LastWorkWeight = 1.21 // 1.21 is the magic number to curve the time weight
)

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
		v           float64
		length      = float64(s.tenants[tenantID].BlocklistLength)
		outstanding = float64(s.tenants[tenantID].OutstanidngBlocklistLength)
		since       = float64(time.Since(s.tenants[tenantID].LastWork).Minutes()) * LastWorkWeight
	)

	if length == 0 {
		return 0
	}

	v += ((length * outstanding) / length) * since

	if v == 0 {
		v++
	}

	return int(v)
}
