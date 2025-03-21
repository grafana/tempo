package tenantselector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BlockListWeightedTenantSelector_PriorityForTenant_single(t *testing.T) {
	tests := []struct {
		name             string
		tenants          []Tenant
		expectedTenant   string
		expectedPriority int
	}{
		{
			name: "Large tenants should have a higher priority",
			tenants: []Tenant{
				{
					ID:                         "tenant1",
					BlocklistLength:            1000000,
					OutstandingBlocklistLength: 10000,
				},
				{
					ID:                         "tenant2",
					BlocklistLength:            1000,
					OutstandingBlocklistLength: 10,
				},
			},
			expectedTenant:   "tenant1",
			expectedPriority: 1010000,
		},
		{
			name: "Tenants with no blocklist should have a priority of 0",
			tenants: []Tenant{
				{
					ID:                         "tenant1",
					BlocklistLength:            0,
					OutstandingBlocklistLength: 0,
				},
			},
			expectedTenant:   "tenant1",
			expectedPriority: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				s              = NewBlockListWeightedTenantSelector(tt.tenants)
				maxPriority    int
				priorityTenant string
			)

			for _, tenant := range tt.tenants {

				got := s.PriorityForTenant(tenant.ID)
				if got == 0 || got > maxPriority {
					maxPriority = got
					priorityTenant = tenant.ID
				}

			}

			assert.Equal(t, tt.expectedTenant, priorityTenant)
			assert.Equal(t, tt.expectedPriority, maxPriority)
		})
	}
}
