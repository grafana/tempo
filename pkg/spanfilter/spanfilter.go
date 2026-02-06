package spanfilter

import (
	"github.com/grafana/tempo/pkg/spanfilter/config"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

type SpanFilter struct {
	include     []*splitPolicy
	includeOnly []*splitPolicy
	exclude     []*splitPolicy

	hasInclude     bool
	hasIncludeOnly bool
	hasExclude     bool
}

// NewSpanFilter returns a SpanFilter that will filter spans based on the given filter policies.
func NewSpanFilter(filterPolicies []config.FilterPolicy) (*SpanFilter, error) {
	sf := new(SpanFilter)
	for _, policy := range filterPolicies {
		err := config.ValidateFilterPolicy(policy)
		if err != nil {
			return nil, err
		}

		include, err := getSplitPolicy(policy.Include)
		if err != nil {
			return nil, err
		}

		if include != nil {
			sf.include = append(sf.include, include)
			sf.hasInclude = true
		}

		includeOnly, err := getSplitPolicy(policy.IncludeOnly)
		if err != nil {
			return nil, err
		}

		if includeOnly != nil {
			sf.includeOnly = append(sf.includeOnly, includeOnly)
			sf.hasIncludeOnly = true
		}

		exclude, err := getSplitPolicy(policy.Exclude)
		if err != nil {
			return nil, err
		}
		if exclude != nil {
			sf.exclude = append(sf.exclude, exclude)
			sf.hasExclude = true
		}
	}

	return sf, nil
}

// ApplyFilterPolicy returns true if the span should be included in the metrics.
func (f *SpanFilter) ApplyFilterPolicy(rs *v1.Resource, span *tracev1.Span) bool {
	// With no filter policies specified, all spans are included.
	if !f.hasInclude && !f.hasIncludeOnly && !f.hasExclude {
		return true
	}

	if f.hasExclude && f.isExcluded(rs, span) {
		return false
	}

	if f.hasIncludeOnly && f.isIncludedOnly(rs, span) {
		return true
	}

	if f.hasInclude {
		return f.isIncluded(rs, span)
	}

	// If we have an include_only but NO standard include, and we reached
	// here, it means include_only didn't match. -> return false.
	// IF NO inclusion rules exist at all -> return true.
	return !f.hasIncludeOnly
}

// This is different than the isIncluded. It's a VIP pass,working as an OR expression.
// if ANY policy matches the span is included
func (f *SpanFilter) isIncludedOnly(rs *v1.Resource, span *tracev1.Span) bool {
	for _, policy := range f.includeOnly {
		if policy.Match(rs, span) {
			return true
		}
	}
	return false
}

func (f *SpanFilter) isIncluded(rs *v1.Resource, span *tracev1.Span) bool {
	for _, policy := range f.include {
		if !policy.Match(rs, span) {
			return false
		}
	}
	return true
}

func (f *SpanFilter) isExcluded(rs *v1.Resource, span *tracev1.Span) bool {
	for _, policy := range f.exclude {
		if policy.Match(rs, span) {
			return true
		}
	}
	return false
}

func getSplitPolicy(policy *config.PolicyMatch) (*splitPolicy, error) {
	if policy == nil {
		return nil, nil
	}
	return newSplitPolicy(policy)
}
