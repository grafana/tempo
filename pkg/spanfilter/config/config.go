package config

import (
	"fmt"

	"github.com/grafana/tempo/pkg/traceql"
)

type FilterPolicy struct {
	Include *PolicyMatch `yaml:"include" json:"include,omitempty"`
	Exclude *PolicyMatch `yaml:"exclude" json:"exclude,omitempty"`
}

type MatchType string

const (
	Strict MatchType = "strict"
	Regex  MatchType = "regex"
)

var supportedIntrinsics = []traceql.Intrinsic{
	traceql.IntrinsicKind,
	traceql.IntrinsicName,
	traceql.IntrinsicStatus,
}

type PolicyMatch struct {
	MatchType  MatchType              `yaml:"match_type" json:"match_type"`
	Attributes []MatchPolicyAttribute `yaml:"attributes" json:"attributes"`
}

type MatchPolicyAttribute struct {
	Key   string      `yaml:"key" json:"key"`
	Value interface{} `yaml:"value" json:"value"`
}

func ValidateFilterPolicy(policy FilterPolicy) error {
	if policy.Include == nil && policy.Exclude == nil {
		return fmt.Errorf("invalid filter policy; policies must have at least an `include` or `exclude`: %v", policy)
	}

	if policy.Include != nil {
		if err := ValidatePolicyMatch(policy.Include); err != nil {
			return fmt.Errorf("invalid include policy: %w", err)
		}
	}

	if policy.Exclude != nil {
		if err := ValidatePolicyMatch(policy.Exclude); err != nil {
			return fmt.Errorf("invalid exclude policy: %w", err)
		}
	}

	return nil
}

func ValidatePolicyMatch(match *PolicyMatch) error {
	if match.MatchType != Strict && match.MatchType != Regex {
		return fmt.Errorf("invalid match type: %v", match.MatchType)
	}

	for _, attr := range match.Attributes {
		if attr.Key == "" {
			return fmt.Errorf("invalid attribute: %v", attr)
		}

		a, err := traceql.ParseIdentifier(attr.Key)
		if err != nil {
			return err
		}
		if a.Scope == traceql.AttributeScopeNone {
			switch a.Intrinsic {
			case traceql.IntrinsicKind, traceql.IntrinsicName, traceql.IntrinsicStatus: // currently supported
			default:
				return fmt.Errorf("currently unsupported intrinsic: %s; supported intrinsics: %q", a.Intrinsic, supportedIntrinsics)
			}
		}
	}

	return nil
}
