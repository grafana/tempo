package livestore

import "testing"

func TestIsValidTenantID(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		want     bool
	}{
		// Valid cases
		{name: "simple alphanumeric", tenantID: "tenant1", want: true},
		{name: "with hyphens", tenantID: "tenant-1", want: true},
		{name: "with underscores", tenantID: "tenant_1", want: true},
		{name: "with dots", tenantID: "org.prod", want: true},
		{name: "grafana cloud style", tenantID: "org-123.production", want: true},
		{name: "mixed case", tenantID: "TenantABC", want: true},
		{name: "all allowed chars", tenantID: "abc-123_DEF.xyz", want: true},
		{name: "exactly 64 chars", tenantID: "a234567890123456789012345678901234567890123456789012345678901234", want: true},

		// Invalid cases
		{name: "empty string", tenantID: "", want: false},
		{name: "too long (65 chars)", tenantID: "a2345678901234567890123456789012345678901234567890123456789012345", want: false},
		{name: "with special char $", tenantID: "tenant$", want: false},
		{name: "with slash", tenantID: "tenant/1", want: false},
		{name: "path traversal attempt", tenantID: "../etc/passwd", want: false},
		{name: "with space", tenantID: "tenant 1", want: false},
		{name: "with colon", tenantID: "tenant:1", want: false},
		{name: "with asterisk", tenantID: "tenant*", want: false},
		{name: "with question mark", tenantID: "tenant?", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTenantID(tt.tenantID); got != tt.want {
				t.Errorf("isValidTenantID(%q) = %v, want %v", tt.tenantID, got, tt.want)
			}
		})
	}
}
