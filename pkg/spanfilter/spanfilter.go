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
		}

		includeOnly, err := getSplitPolicy(policy.IncludeOnly)
		if err != nil {
			return nil, err
		}

		if includeOnly != nil {
			sf.includeOnly = append(sf.includeOnly, includeOnly)
		}
		exclude, err := getSplitPolicy(policy.Exclude)
		if err != nil {
			return nil, err
		}
		if exclude != nil {
			sf.exclude = append(sf.exclude, exclude)
		}
	}

	return sf, nil
}

// ApplyFilterPolicy returns true if the span should be included in the metrics.
func (f *SpanFilter) ApplyFilterPolicy(rs *v1.Resource, span *tracev1.Span) bool {
	// With no filter policies specified, all spans are included.
	if len(f.include) == 0 && len(f.exclude) == 0 {
		return true
	}

	return f.isIncluded(rs, span) && !f.isExcluded(rs, span)
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
