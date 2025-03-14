package tenantselector

import "time"

type TenantSelector interface {
	PriorityForTenant(tenantID string) int
}

type Tenant struct {
	ID                         string
	BlocklistLength            int
	OutstanidngBlocklistLength int
	LastWork                   time.Time
}
