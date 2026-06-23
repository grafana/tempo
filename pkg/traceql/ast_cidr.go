package traceql

import (
	"fmt"
	"net/netip"
	"strings"
)

// CIDR is a span-filter function: cidr(<field>, "<prefix>"[, "<prefix>"...]).
// It returns true when the field's value parses as an IP address contained in
// any of the listed CIDR prefixes. IPv4 and IPv6 are both supported and may be
// mixed in a single call.
type CIDR struct {
	Field    FieldExpression
	Prefixes []string

	parsed []netip.Prefix // lazily compiled from Prefixes, cached
}

func newCIDR(field FieldExpression, prefixes []string) FieldExpression {
	return &CIDR{
		Field:    field,
		Prefixes: prefixes,
	}
}

// nolint: revive
func (*CIDR) __fieldExpression() {}

func (c *CIDR) impliedType() StaticType {
	return TypeBoolean
}

func (c *CIDR) referencesSpan() bool {
	return c.Field.referencesSpan()
}

// compile parses all prefix literals once and caches them. It is idempotent and
// returns an error on the first unparseable prefix.
func (c *CIDR) compile() error {
	if c.parsed != nil {
		return nil
	}
	parsed := make([]netip.Prefix, 0, len(c.Prefixes))
	for _, p := range c.Prefixes {
		prefix, err := netip.ParsePrefix(p)
		if err != nil {
			return fmt.Errorf("cidr() has an invalid prefix %q: %w", p, err)
		}
		parsed = append(parsed, prefix.Masked())
	}
	c.parsed = parsed
	return nil
}

func (c *CIDR) validate() error {
	if err := c.Field.validate(); err != nil {
		return err
	}
	t := c.Field.impliedType()
	if t != TypeString && t != TypeAttribute {
		return fmt.Errorf("cidr() requires a string attribute as its first argument: %s", c.String())
	}
	if len(c.Prefixes) == 0 {
		return fmt.Errorf("cidr() requires at least one prefix: %s", c.String())
	}
	return c.compile()
}

func (c *CIDR) extractConditions(request *FetchSpansRequest) {
	// CIDR containment cannot be pushed to storage; fetch spans that have the
	// attribute (OpNone via the field) and let the engine filter.
	c.Field.extractConditions(request)
}

func (c *CIDR) execute(span Span) (Static, error) {
	if err := c.compile(); err != nil {
		return NewStaticNil(), err
	}
	v, err := c.Field.execute(span)
	if err != nil {
		return NewStaticNil(), err
	}
	if v.Type != TypeString {
		return StaticFalse, nil
	}
	addr, perr := netip.ParseAddr(v.EncodeToString(false))
	if perr != nil {
		return StaticFalse, nil
	}
	addr = addr.Unmap()
	for _, p := range c.parsed {
		if p.Contains(addr) {
			return StaticTrue, nil
		}
	}
	return StaticFalse, nil
}

func (c *CIDR) String() string {
	var sb strings.Builder
	sb.WriteString("cidr(")
	sb.WriteString(c.Field.String())
	for _, p := range c.Prefixes {
		sb.WriteString(", ")
		sb.WriteString(NewStaticString(p).String())
	}
	sb.WriteString(")")
	return sb.String()
}
