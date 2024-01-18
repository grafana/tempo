package spanfilter

import (
	"github.com/grafana/tempo/pkg/spanfilter/config"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

type SpanFilter struct {
	filterPolicies []*filterPolicy
}

type filterPolicy struct {
	Include *splitPolicy
	Exclude *splitPolicy
}

// NewSpanFilter returns a SpanFilter that will filter spans based on the given filter policies.
func NewSpanFilter(filterPolicies []config.FilterPolicy) (*SpanFilter, error) {
	var policies []*filterPolicy

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
			policies = append(policies, &p)
		}
	}

	return &SpanFilter{
		filterPolicies: policies,
	}, nil
}

// ApplyFilterPolicy returns true if the span should be included in the
// metrics.  Each filter policy is evaluated in order.
func (f *SpanFilter) ApplyFilterPolicy(rs *v1.Resource, span *tracev1.Span) bool {
	// With no filter policies specified, all spans are included.
	if len(f.filterPolicies) == 0 {
		return true
	}

	var (
		included bool // a policy matched and included the span
		excluded bool // a policy matched and excluded the span
		tries    int  // the number of attempts to exclude a span
	)

	for _, policy := range f.filterPolicies {
		// if both matched, the span is excluded unless included by a later policy
		// if neither matched, the span is included
		// if the include policy matched, but the exclude policy did not, include the span
		// if the exclude policy matched, but the include policy did not, exclude the span
		// if only exclude policies are specified, and none match, include the span
		// if only exclude policies are specified, and any match, exclude the span

		if policy.Include != nil {
			if policy.Include.Match(rs, span) {
				included = true
				if policy.Exclude != nil {
					// continue to check additional policies for inclusion
					if policy.Exclude.Match(rs, span) {
						excluded = true
						continue
					}
				}
				return true
			}
		}

		if policy.Exclude != nil {
			tries++
			if policy.Exclude.Match(rs, span) {
				excluded = true
			}
		}
	}

	// attempts were made to exclude the span
	if tries > 0 {
		// we didn't include nor exclude the span, include it
		if !included && !excluded {
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
