package tenantselector

import (
	"testing"
	"time"

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
			name: "A large tenant with 1% outstanding blocks should win against a smaller tenant",
			tenants: []Tenant{
				{
					ID:                         "tenant1",
					BlocklistLength:            1000000,
					OutstanidngBlocklistLength: 10000,
					LastWork:                   time.Now().Add(-time.Minute),
				},
				{
					ID:                         "tenant2",
					BlocklistLength:            1000,
					OutstanidngBlocklistLength: 10,
					LastWork:                   time.Now().Add(-time.Minute),
				},
			},
			expectedTenant:   "tenant1",
			expectedPriority: 121e2,
		},
		{
			name: "A smaller tenant which has not been worked on for a long time should have a high priority",
			tenants: []Tenant{
				{
					ID:                         "tenant1",
					BlocklistLength:            1e5,
					OutstanidngBlocklistLength: 1e2,
					LastWork:                   time.Now().Add(-1 * time.Minute),
				},
				{
					ID:                         "tenant2",
					BlocklistLength:            1e3,
					OutstanidngBlocklistLength: 1e1,
					LastWork:                   time.Now().Add(-120 * time.Minute),
				},
			},
			expectedTenant:   "tenant2",
			expectedPriority: 1452,
		},
		{
			name: "A tenant with 1 block should win against a tenant with 0 blocks",
			tenants: []Tenant{
				{
					ID:                         "tenant1",
					BlocklistLength:            0,
					OutstanidngBlocklistLength: 0,
					LastWork:                   time.Now().Add(-1 * time.Minute),
				},
				{
					ID:                         "tenant2",
					BlocklistLength:            1,
					OutstanidngBlocklistLength: 1,
					LastWork:                   time.Now().Add(-1 * time.Minute),
				},
			},
			expectedTenant:   "tenant2",
			expectedPriority: 1,
		},
		{
			name: "A tenant with 0 blocks has zero priority",
			tenants: []Tenant{
				{
					ID:                         "tenant1",
					BlocklistLength:            0,
					OutstanidngBlocklistLength: 0,
					LastWork:                   time.Now().Add(-1 * time.Minute),
				},
			},
			expectedTenant:   "tenant1",
			expectedPriority: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewBlockListWeightedTenantSelector(tt.tenants)
			var maxPriority int
			var priorityTenant string

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
