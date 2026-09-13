package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp/syntax"
)

const (
	maxPolicyCustomRules  = 16
	maxPolicyStringLength = 512
	maxCustomRegexLength  = 4096
	// Bound expanded exact regexp programs before regexp.Compile or Simplify.
	// Optimization caps may fall back, but this policy-wide resource cap rejects.
	maxCustomPolicyInstructions = 1 << 20
)

type FeatureConfig struct {
	DetectionEnabled bool      `yaml:"detection_enabled,omitempty" json:"detection_enabled,omitempty"`
	EnabledRules     *[]string `yaml:"enabled_rules,omitempty" json:"enabled_rules,omitempty"`
}

// MarshalJSON preserves an explicitly empty Go-created selection as [] rather
// than null, which would enable the full catalog when the config is reloaded.
func (c FeatureConfig) MarshalJSON() ([]byte, error) {
	type plainFeatureConfig FeatureConfig
	if c.EnabledRules != nil && *c.EnabledRules == nil {
		empty := []string{}
		c.EnabledRules = &empty
	}
	return json.Marshal(plainFeatureConfig(c))
}

// Validate checks operator configuration even when detection is disabled.
func (c FeatureConfig) Validate() error {
	_, err := validateNativeSelection(c.EnabledRules)
	return err
}

// Policy excludes selected native rules and adds tenant-owned value rules.
// The process selection bounds native coverage. Attribute names are never input.
type Policy struct {
	DisabledRules []string     `yaml:"disabled_rules,omitempty" json:"disabled_rules,omitempty"`
	CustomRules   []CustomRule `yaml:"custom_rules,omitempty" json:"custom_rules,omitempty"`
}

type CustomRule struct {
	ID    string `yaml:"id" json:"id"`
	Regex string `yaml:"regex" json:"regex"`
}

type FieldKind string

const (
	FieldKindResourceAttribute FieldKind = "resource_attribute"
	FieldKindScopeAttribute    FieldKind = "scope_attribute"
	FieldKindSpanAttribute     FieldKind = "span_attribute"
	FieldKindEventAttribute    FieldKind = "event_attribute"
	FieldKindLinkAttribute     FieldKind = "link_attribute"
	FieldKindTraceField        FieldKind = "trace_field"
)

type Match struct {
	RuleID string
}

type Verdict struct {
	Matches []Match
}

func (v Verdict) Matched() bool {
	return len(v.Matches) > 0
}

type CompiledPolicy struct {
	catalog       *compiledCatalog
	disabled      ruleSet
	disabledCount int
	custom        compiledRuleSet
	minWidth      int // lower bound in runes across enabled native and custom rules
}

const batchDetectorValueCacheEntries = 32

type batchDetectorValueCacheEntry struct {
	value   string
	matches []Match
	hash    uint32
	valid   bool
}

type BatchDetector struct {
	policy     *CompiledPolicy
	valueCache [batchDetectorValueCacheEntries]batchDetectorValueCacheEntry
	hits       keywordHits
	hitBuffer  keywordHitBuffer
}

func (p *CompiledPolicy) NewBatchDetector() BatchDetector {
	return BatchDetector{policy: p}
}

func (d *BatchDetector) Detect(value string) Verdict {
	// Every rune consumes at least one byte. This bound includes tenant rules,
	// and becomes zero for nullable rules, so it is safe before cache lookup.
	if len(value) < d.policy.minWidth {
		return Verdict{}
	}
	return d.detect(value)
}

func (d *BatchDetector) detect(value string) Verdict {
	// Re-pointing on every call keeps copies of the detector self-contained.
	d.hits.buffer = &d.hitBuffer
	hash := batchDetectorValueHash(value)
	entry := &d.valueCache[hash&(batchDetectorValueCacheEntries-1)]
	if entry.valid && entry.hash == hash && entry.value == value {
		return Verdict{Matches: append([]Match(nil), entry.matches...)}
	}
	entry.value = value
	entry.matches = d.policy.detectMatches(value, &d.hits)
	entry.hash = hash
	entry.valid = true
	return Verdict{Matches: append([]Match(nil), entry.matches...)}
}

func batchDetectorValueHash(value string) uint32 {
	hash := uint32(len(value)) * 16777619
	if len(value) == 0 {
		return hash
	}
	hash = (hash ^ uint32(value[0])) * 16777619
	hash = (hash ^ uint32(value[len(value)/2])) * 16777619
	return (hash ^ uint32(value[len(value)-1])) * 16777619
}

// CatalogRuleIDs lists every supported native rule in lexical order.
// Listing metadata does not compile the catalog.
func CatalogRuleIDs() ([]string, error) {
	ids := make([]string, len(nativeRuleSpecs))
	for i, spec := range nativeRuleSpecs {
		if i > 0 && nativeRuleSpecs[i-1].ID == spec.ID {
			return nil, errors.New("secret catalog contains duplicate IDs")
		}
		ids[i] = spec.ID
	}
	return ids, nil
}

func (p *CompiledPolicy) Detect(value string) Verdict {
	if len(value) < p.minWidth {
		return Verdict{}
	}
	return p.detect(value)
}

func (p *CompiledPolicy) detect(value string) Verdict {
	var hits keywordHits
	matches := p.detectMatches(value, &hits)
	hits.release()
	return Verdict{Matches: matches}
}

func (p *CompiledPolicy) detectMatches(value string, hits *keywordHits) []Match {
	var matches []Match
	if p.disabledCount == 0 {
		matches = p.catalog.detect(value, hits, nil)
	} else if p.disabledCount < len(p.catalog.rules) {
		matches = p.catalog.detectExcluding(value, hits, &p.disabled, nil)
	}
	return p.custom.detect(value, hits, matches)
}

func (c *PolicyCompiler) compilePolicy(policy Policy) (*CompiledPolicy, error) {
	if err := validatePolicyBounds(policy); err != nil {
		return nil, err
	}
	var excluded ruleSet
	for i, id := range policy.DisabledRules {
		index, supported := supportedNativeRuleIndex(id)
		if len(id) > maxPolicyStringLength || !supported {
			return nil, fmt.Errorf("disabled secrets rule %d is not supported", i+1)
		}
		if excluded.has(index) {
			return nil, fmt.Errorf("disabled secrets rule %d is duplicated", i+1)
		}
		excluded.add(index)
	}
	catalog, err := c.catalog()
	if err != nil {
		return nil, errors.New("secret catalog compilation failed")
	}
	var disabled ruleSet
	disabledCount := 0
	for _, id := range policy.DisabledRules {
		if index, active := catalog.byID[id]; active {
			disabled.add(index)
			disabledCount++
		}
	}
	custom, err := compileCustomRules(policy.CustomRules)
	if err != nil {
		return nil, err
	}
	minWidth := int(^uint(0) >> 1)
	for i := range catalog.rules {
		if !disabled.has(uint16(i)) {
			minWidth = min(minWidth, catalog.rules[i].plan.minWidth)
		}
	}
	for i := range custom.rules {
		minWidth = min(minWidth, custom.rules[i].plan.minWidth)
	}
	return &CompiledPolicy{
		catalog:       catalog,
		disabled:      disabled,
		disabledCount: disabledCount,
		custom:        custom,
		minWidth:      minWidth,
	}, nil
}

func validatePolicyBounds(policy Policy) error {
	if len(policy.DisabledRules) > len(nativeRuleSpecs) {
		return errors.New("secrets policy has too many disabled rules")
	}
	if len(policy.CustomRules) > maxPolicyCustomRules {
		return errors.New("secrets policy has too many custom rules")
	}
	for i, rule := range policy.CustomRules {
		if len(rule.ID) > maxPolicyStringLength {
			return fmt.Errorf("custom secrets rule %d ID exceeds the byte limit", i+1)
		}
		if !validCustomRuleID(rule.ID) {
			return fmt.Errorf("custom secrets rule %d ID contains unsupported characters", i+1)
		}
		if len(rule.Regex) > maxCustomRegexLength {
			return fmt.Errorf("custom secrets rule %d regex exceeds the byte limit", i+1)
		}
	}
	return nil
}

func validCustomRuleID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func compileCustomRules(custom []CustomRule) (compiledRuleSet, error) {
	var set compiledRuleSet
	if len(custom) == 0 {
		return set, nil
	}
	seen := make(map[string]struct{}, len(custom))
	instructions := 0
	// Complete the bounded parse/accounting pass before compiling any rule. A
	// small source expression can otherwise expand into a large exact program.
	for i, rule := range custom {
		if rule.ID == "" || rule.Regex == "" {
			return set, fmt.Errorf("custom secrets rule %d requires id and regex", i+1)
		}
		if _, exists := supportedNativeRuleIndex(rule.ID); exists {
			return set, fmt.Errorf("custom secrets rule %d conflicts with the production catalog", i+1)
		}
		if _, exists := seen[rule.ID]; exists {
			return set, fmt.Errorf("custom secrets rule %d has a duplicate ID", i+1)
		}
		seen[rule.ID] = struct{}{}
		parsed, err := syntax.Parse(rule.Regex, syntax.Perl)
		if err != nil {
			return set, fmt.Errorf("custom secrets rule %d has an invalid regex", i+1)
		}
		// Every program also has the fail and match instructions.
		instructions = addRegexpInstructions(instructions, addRegexpInstructions(regexpInstructionEstimate(parsed), 2))
		if instructions > maxCustomPolicyInstructions {
			return set, errors.New("custom secrets policy exceeds the regex instruction limit")
		}
	}
	// Identical execution plans may share immutable regexes and filters while
	// each configured rule keeps its own ID and matcher output. This cache is
	// compilation-local: it retains neither policies nor historical revisions.
	type compiledPlan struct {
		index    int
		keywords []string
	}
	var plans map[string]compiledPlan
	if len(custom) > 1 {
		plans = make(map[string]compiledPlan, len(custom))
	}
	set.rules = make([]compiledRule, len(custom))
	var matcherKeywords []matcherKeyword
	for i, rule := range custom {
		if previous, ok := plans[rule.Regex]; ok {
			set.rules[i] = set.rules[previous.index]
			set.rules[i].id = rule.ID
			matcherKeywords = set.addRuleKeywords(uint16(i), previous.keywords, matcherKeywords)
			continue
		}
		spec := catalogRuleSpec{ID: rule.ID, Regex: rule.Regex}
		keywords := effectiveKeywords(spec)
		compiled, err := compileRuleSpec(spec, keywords)
		if err != nil {
			// regexp errors include the expression. Neither those errors nor
			// tenant-authored IDs are safe to return to logs or callers.
			return compiledRuleSet{}, fmt.Errorf("custom secrets rule %d could not be compiled", i+1)
		}
		set.rules[i] = compiled
		matcherKeywords = set.addRuleKeywords(uint16(i), keywords, matcherKeywords)
		if plans != nil {
			plans[rule.Regex] = compiledPlan{i, keywords}
		}
	}
	var err error
	set.pairMatcher, err = newCatalogPairMatcher(matcherKeywords)
	if err != nil {
		return compiledRuleSet{}, errors.New("custom secrets policy matcher could not be compiled")
	}
	return set, nil
}

func isBaselinePolicy(policy Policy) bool {
	return len(policy.CustomRules) == 0 && len(policy.DisabledRules) == 0
}
