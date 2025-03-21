package tenantselector

// TenantSelector is an interface for selecting a tenant based on some criteria.
type TenantSelector interface {
	PriorityForTenant(tenantID string) int
}

type Tenant struct {
	ID                         string
	BlocklistLength            int
	OutstandingBlocklistLength int
}
