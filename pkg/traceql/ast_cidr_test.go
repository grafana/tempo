package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func cidrSpan(ip string) *mockSpan {
	return &mockSpan{
		attributes: map[Attribute]Static{
			NewAttribute("addr"): NewStaticString(ip),
		},
	}
}

func TestCIDRExecute(t *testing.T) {
	tests := []struct {
		name     string
		prefixes []string
		ip       string
		match    bool
	}{
		{"ipv4 in range", []string{"10.0.0.0/8"}, "10.1.2.3", true},
		{"ipv4 out of range", []string{"10.0.0.0/8"}, "192.168.1.1", false},
		{"ipv4 host route", []string{"203.0.113.5/32"}, "203.0.113.5", true},
		{"ipv4 /0 matches all", []string{"0.0.0.0/0"}, "8.8.8.8", true},
		{"ipv6 in range", []string{"fc00::/7"}, "fd12:3456::1", true},
		{"ipv6 out of range", []string{"fc00::/7"}, "2001:db8::1", false},
		{"ipv4-mapped ipv6 matches ipv4 prefix", []string{"10.0.0.0/8"}, "::ffff:10.1.2.3", true},
		{"mixed list matches ipv6", []string{"10.0.0.0/8", "fc00::/7"}, "fd00::1", true},
		{"mixed list matches ipv4", []string{"10.0.0.0/8", "fc00::/7"}, "10.9.9.9", true},
		{"family mismatch v4 addr v6 prefix", []string{"fc00::/7"}, "10.0.0.1", false},
		{"unparseable ip is no match", []string{"10.0.0.0/8"}, "not-an-ip", false},
		{"ip with port is no match", []string{"10.0.0.0/8"}, "10.1.2.3:8080", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCIDR(NewAttribute("addr"), tt.prefixes)
			out, err := c.execute(cidrSpan(tt.ip))
			require.NoError(t, err)
			require.Equal(t, NewStaticBool(tt.match), out)
		})
	}
}

func TestCIDRExecuteMissingAttribute(t *testing.T) {
	c := newCIDR(NewAttribute("addr"), []string{"10.0.0.0/8"})
	out, err := c.execute(&mockSpan{attributes: map[Attribute]Static{}})
	require.NoError(t, err)
	require.Equal(t, StaticFalse, out)
}

func TestCIDRValidate(t *testing.T) {
	tests := []struct {
		name     string
		field    FieldExpression
		prefixes []string
		wantErr  bool
	}{
		{"valid ipv4 and ipv6 prefixes", NewAttribute("addr"), []string{"10.0.0.0/8", "fc00::/7"}, false},
		{"invalid prefix literal", NewAttribute("addr"), []string{"not-a-cidr"}, true},
		{"non-string first argument", NewStaticInt(5), []string{"10.0.0.0/8"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newCIDR(tt.field, tt.prefixes).validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCIDRImpliedTypeAndString(t *testing.T) {
	c := newCIDR(NewAttribute("addr"), []string{"10.0.0.0/8", "fc00::/7"})
	require.Equal(t, TypeBoolean, c.impliedType())
	require.Equal(t, "cidr(.addr, `10.0.0.0/8`, `fc00::/7`)", c.String())
}
