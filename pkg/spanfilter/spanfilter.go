package spanfilter

import (
	"github.com/grafana/tempo/pkg/spanfilter/config"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

type SpanFilter struct {
	filterPolicies []filterPolicy
}

type filterPolicy struct {
	Include *splitPolicy
	Exclude *splitPolicy
}

// NewSpanFilter returns a SpanFilter that will filter spans based on the given filter policies.
func NewSpanFilter(filterPolicies []config.FilterPolicy) (*SpanFilter, error) {
	var policies []filterPolicy

	for _, policy := range filterPolicies {
		err := config.ValidateFilterPolicy(policy)
		if err != nil {
			return nil, err
		}

		include, err := getSplitPolicy(policy.Include)
		if err != nil {
			return nil, err
		}

		exclude, err := getSplitPolicy(policy.Exclude)
		if err != nil {
			return nil, err
		}
		p := filterPolicy{
			Include: include,
			Exclude: exclude,
		}

		if p.Include != nil || p.Exclude != nil {
			policies = append(policies, p)
		}
	}

	return &SpanFilter{
		filterPolicies: policies,
	}, nil
}

// ApplyFilterPolicy returns true if the span should be included in the metrics.
func (f *SpanFilter) ApplyFilterPolicy(rs *v1.Resource, span *tracev1.Span) bool {
	// With no filter policies specified, all spans are included.
	if len(f.filterPolicies) == 0 {
		return true
	}

	for _, policy := range f.filterPolicies {
		if policy.Include != nil && !policy.Include.Match(rs, span) {
			return false
		}

		if policy.Exclude != nil && policy.Exclude.Match(rs, span) {
			return false
		}
	}

	return true
}

func getSplitPolicy(policy *config.PolicyMatch) (*splitPolicy, error) {
	if policy == nil {
		return nil, nil
	}
	return newSplitPolicy(policy)
}
