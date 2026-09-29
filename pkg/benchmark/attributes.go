package benchmark

import (
	"errors"
	"fmt"
)

// Attribute scopes, as recorded in a profile.
const (
	AttributeScopeResource  = "resource"
	AttributeScopeSpan      = "span"
	AttributeScopeIntrinsic = "intrinsic"
)

// Attribute value types, as recorded in a profile.
const (
	AttributeTypeString   = "string"
	AttributeTypeInt      = "int"
	AttributeTypeFloat    = "float"
	AttributeTypeBool     = "bool"
	AttributeTypeStatus   = "status"
	AttributeTypeKind     = "kind"
	AttributeTypeDuration = "duration"
)

// AttributeProfiles ranks a block's attributes by total bytes, so queries can
// be generated from what the block actually holds.
type AttributeProfiles struct {
	// Spans is the denominator of every density and selectivity.
	Spans  uint64             `json:"spans"`
	Ranked []AttributeProfile `json:"ranked"`
}

// AttributeProfile describes one attribute. The same name with two types is
// two profiles, because a query matches them separately.
type AttributeProfile struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	// Dedicated is true when the attribute has a column of its own rather than
	// living in the generic key/value columns.
	Dedicated   bool    `json:"dedicated"`
	TotalBytes  uint64  `json:"totalBytes"`
	Cardinality uint64  `json:"cardinality"`
	Density     float64 `json:"density"`
	// Exactly one of Values and Quantiles is set: Quantiles only for a numeric
	// attribute with too many distinct values to list.
	Values    []AttributeValue    `json:"values,omitempty"`
	Quantiles []AttributeQuantile `json:"quantiles,omitempty"`
}

// AttributeValue is a value and the fraction of all spans carrying it.
type AttributeValue struct {
	Value       string  `json:"value"`
	Selectivity float64 `json:"selectivity"`
}

// AttributeQuantile is the value at quantile Q of the spans carrying the
// attribute, and the fraction of all spans whose value is greater than it.
type AttributeQuantile struct {
	Q           float64 `json:"q"`
	Value       string  `json:"value"`
	Selectivity float64 `json:"selectivity"`
}

func isNumericType(t string) bool {
	return t == AttributeTypeInt || t == AttributeTypeFloat || t == AttributeTypeDuration
}

func (a *AttributeProfiles) validate() error {
	if a.Spans == 0 {
		return fmt.Errorf("attribute profiles cover %d spans", a.Spans)
	}
	for i := range a.Ranked {
		p := &a.Ranked[i]
		if i > 0 && p.TotalBytes > a.Ranked[i-1].TotalBytes {
			return fmt.Errorf("%s is not ranked by total bytes", p.key())
		}
		if err := p.validate(); err != nil {
			return fmt.Errorf("attribute %s: %w", p.key(), err)
		}
	}
	return nil
}

func (p *AttributeProfile) key() string {
	return p.Scope + "." + p.Name + " (" + p.Type + ")"
}

func (p *AttributeProfile) validate() error {
	switch p.Scope {
	case AttributeScopeResource, AttributeScopeSpan, AttributeScopeIntrinsic:
	default:
		return fmt.Errorf("unknown scope %q", p.Scope)
	}
	switch p.Type {
	case AttributeTypeString, AttributeTypeInt, AttributeTypeFloat, AttributeTypeBool,
		AttributeTypeStatus, AttributeTypeKind, AttributeTypeDuration:
	default:
		return fmt.Errorf("unknown type %q", p.Type)
	}
	if !isFraction(p.Density) {
		return fmt.Errorf("density %v is not a fraction", p.Density)
	}
	if (len(p.Values) > 0) == (len(p.Quantiles) > 0) {
		return errors.New("exactly one of values and quantiles must be set")
	}
	if len(p.Quantiles) > 0 && !isNumericType(p.Type) {
		return fmt.Errorf("quantiles are only for numeric types, not %s", p.Type)
	}

	for i, v := range p.Values {
		if !isFraction(v.Selectivity) {
			return fmt.Errorf("value %q selectivity %v is not a fraction", v.Value, v.Selectivity)
		}
		if i > 0 && v.Selectivity > p.Values[i-1].Selectivity {
			return fmt.Errorf("value %q is not sorted by selectivity", v.Value)
		}
	}
	for i, q := range p.Quantiles {
		if q.Q <= 0 || q.Q >= 1 {
			return fmt.Errorf("quantile %v is not in (0, 1)", q.Q)
		}
		if i > 0 && q.Q <= p.Quantiles[i-1].Q {
			return fmt.Errorf("quantiles are not in increasing order at %v", q.Q)
		}
		if !isFraction(q.Selectivity) {
			return fmt.Errorf("quantile %v selectivity %v is not a fraction", q.Q, q.Selectivity)
		}
	}
	return nil
}

func isFraction(f float64) bool {
	return f >= 0 && f <= 1
}
