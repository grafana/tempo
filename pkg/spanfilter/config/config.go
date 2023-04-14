package config

type FilterPolicy struct {
	Include *PolicyMatch `yaml:"include"`
	Exclude *PolicyMatch `yaml:"exclude"`
}

type MatchType string

const (
	Strict MatchType = "strict"
	Regex  MatchType = "regex"
)

type PolicyMatch struct {
	MatchType  MatchType              `yaml:"match_type"`
	Attributes []MatchPolicyAttribute `yaml:"attributes"`
}

type MatchPolicyAttribute struct {
	Key   string      `yaml:"key"`
	Value interface{} `yaml:"value"`
}
