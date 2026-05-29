package spanfilter

import (
	"github.com/grafana/tempo/pkg/spanfilter/config"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

type SpanFilter struct {
	include    []*splitPolicy
	includeAny []*splitPolicy
	exclude    []*splitPolicy

	hasInclude    bool
	hasIncludeAny bool
	hasExclude    bool
}

type ResourceFilter struct {
	filter *SpanFilter
	rs     *v1.Resource

	includeResourceMask    uint64
	includeAnyResourceMask uint64
	excludeResourceMask    uint64
	useResourceMasks       bool
}

const maxResourceMaskedPolicies = 64

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

		includeAny, err := getSplitPolicy(policy.IncludeAny)
		if err != nil {
			return nil, err
		}

		if includeAny != nil {
			sf.includeAny = append(sf.includeAny, includeAny)
			sf.hasIncludeAny = true
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

func (f *SpanFilter) ForResource(rs *v1.Resource) ResourceFilter {
	rf := ResourceFilter{
		filter: f,
		rs:     rs,
	}
	if !f.hasInclude && !f.hasIncludeAny && !f.hasExclude {
		return rf
	}
	if len(f.include) > maxResourceMaskedPolicies ||
		len(f.includeAny) > maxResourceMaskedPolicies ||
		len(f.exclude) > maxResourceMaskedPolicies {
		return rf
	}

	rf.useResourceMasks = true
	rf.includeResourceMask = resourceMatchMask(rs, f.include)
	rf.includeAnyResourceMask = resourceMatchMask(rs, f.includeAny)
	rf.excludeResourceMask = resourceMatchMask(rs, f.exclude)
	return rf
}

func resourceMatchMask(rs *v1.Resource, policies []*splitPolicy) uint64 {
	var mask uint64
	for i, policy := range policies {
		if policy.matchResource(rs) {
			mask |= 1 << uint(i)
		}
	}
	return mask
}

// ApplyFilterPolicy returns true if the span should be included in the metrics.
func (f *SpanFilter) ApplyFilterPolicy(rs *v1.Resource, span *tracev1.Span) bool {
	// With no filter policies specified, all spans are included.
	if !f.hasInclude && !f.hasIncludeAny && !f.hasExclude {
		return true
	}

	if f.hasExclude && f.isExcluded(rs, span) {
		return false
	}

	if f.hasIncludeAny && f.isIncludedAny(rs, span) {
		return true
	}

	if f.hasInclude {
		return f.isIncluded(rs, span)
	}

	// If we have an include_any but NO standard include, and we reached
	// here, it means include_any didn't match. -> return false.
	// IF NO inclusion rules exist at all -> return true.
	return !f.hasIncludeAny
}

func (f ResourceFilter) Apply(span *tracev1.Span) bool {
	if !f.useResourceMasks {
		return f.filter.ApplyFilterPolicy(f.rs, span)
	}

	if f.filter.hasExclude && f.matchesAnySpan(f.filter.exclude, f.excludeResourceMask, span) {
		return false
	}

	if f.filter.hasIncludeAny && f.matchesAnySpan(f.filter.includeAny, f.includeAnyResourceMask, span) {
		return true
	}

	if f.filter.hasInclude {
		return f.matchesAllSpans(f.filter.include, f.includeResourceMask, span)
	}

	return !f.filter.hasIncludeAny
}

func (f ResourceFilter) matchesAnySpan(policies []*splitPolicy, resourceMask uint64, span *tracev1.Span) bool {
	for i, policy := range policies {
		if resourceMask&(1<<uint(i)) != 0 && (!policy.hasSpanConditions() || policy.matchSpan(span)) {
			return true
		}
	}
	return false
}

func (f ResourceFilter) matchesAllSpans(policies []*splitPolicy, resourceMask uint64, span *tracev1.Span) bool {
	for i, policy := range policies {
		if resourceMask&(1<<uint(i)) == 0 || (policy.hasSpanConditions() && !policy.matchSpan(span)) {
			return false
		}
	}
	return true
}

// This is different than the isIncluded. It's a VIP pass,working as an OR expression.
// if ANY policy matches the span is included
func (f *SpanFilter) isIncludedAny(rs *v1.Resource, span *tracev1.Span) bool {
	for _, policy := range f.includeAny {
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
