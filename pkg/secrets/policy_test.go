package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

// Existing all-native workloads share the same factory-created compiler as a
// standalone detector; construction alone does not compile the native catalog.
var testPolicyCompiler = func() *PolicyCompiler {
	compiler, err := NewPolicyCompiler(nil)
	if err != nil {
		panic(err)
	}
	return compiler
}()

func TestCatalogContainsEveryValueRule(t *testing.T) {
	ids, err := CatalogRuleIDs()
	require.NoError(t, err)
	fixtures := nativeCatalogFixtures(t)
	expected := make([]string, 0, len(fixtures))
	for id := range fixtures {
		expected = append(expected, id)
	}
	slices.Sort(expected)
	assert.Equal(t, expected, ids)
}

func TestPolicyDetectsEveryNativeFixtureByDefault(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	fixtures := nativeCatalogFixtures(t)

	for _, spec := range nativeRuleSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			batch := policy.NewBatchDetector()
			for index, value := range fixtures[spec.ID].Positive {
				t.Run(fmt.Sprintf("positive-%d", index), func(t *testing.T) {
					require.Contains(t, policy.Detect(value).Matches, Match{RuleID: spec.ID})
					require.Contains(t, batch.Detect(value).Matches, Match{RuleID: spec.ID})
					require.Contains(t, batch.Detect(value).Matches, Match{RuleID: spec.ID})
				})
			}
			for index, value := range fixtures[spec.ID].Negative {
				t.Run(fmt.Sprintf("negative-%d", index), func(t *testing.T) {
					require.False(t, hasRuleFinding(policy.Detect(value).Matches, spec.ID))
					require.False(t, hasRuleFinding(batch.Detect(value).Matches, spec.ID))
				})
			}
		})
	}
}

func TestDisabledRulesPreserveOtherNativeAndCustomFindings(t *testing.T) {
	fixture := nativeCatalogFixtures(t)["github-pat"].Positive[0]
	assigned := `api_key="r9Q2m7V4x1Z8c6B3n0H5j2L9p4T7w8Y1"`
	custom := CustomRule{ID: "acme-key", Regex: `ACME-[A-Z0-9]{10}`}
	value := fixture + "\n" + assigned + "\nACME-AB12CD34EF"
	for _, test := range []struct {
		name     string
		disabled []string
		want     []Match
	}{
		{"defaults", nil, []Match{{RuleID: "generic-api-key"}, {RuleID: "github-pat"}, {RuleID: "acme-key"}}},
		{"generic-disabled", []string{"generic-api-key"}, []Match{{RuleID: "github-pat"}, {RuleID: "acme-key"}}},
		{"specific-disabled", []string{"github-pat"}, []Match{{RuleID: "generic-api-key"}, {RuleID: "acme-key"}}},
		{"both-disabled", []string{"github-pat", "generic-api-key"}, []Match{{RuleID: "acme-key"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			policy, err := testPolicyCompiler.CompilePolicy(Policy{DisabledRules: test.disabled, CustomRules: []CustomRule{custom}})
			require.NoError(t, err)
			want := Verdict{Matches: test.want}
			require.Equal(t, want, fullScanPolicyVerdict(policy, value))
			require.Equal(t, want, policy.Detect(value))
			batch := policy.NewBatchDetector()
			require.Equal(t, want, batch.Detect(value))
			require.Equal(t, want, batch.Detect(value))
		})
	}
}

func TestExclusionOrderPreservesNativeLexicalAndCustomConfiguredOrder(t *testing.T) {
	fixtures := nativeCatalogFixtures(t)
	value := fixtures["gcp-api-key"].Positive[0] + "\n" + fixtures["github-pat"].Positive[0] + "\n" +
		`api_key="r9Q2m7V4x1Z8c6B3n0H5j2L9p4T7w8Y1"` + "\nCUSTOM"
	custom := []CustomRule{{ID: "z-custom", Regex: "CUSTOM"}, {ID: "a-custom", Regex: "CUSTOM"}}
	for _, ids := range [][]string{
		{"aws-secret-access-key", "github-pat"},
		{"github-pat", "aws-secret-access-key"},
	} {
		policy, err := testPolicyCompiler.CompilePolicy(Policy{DisabledRules: ids, CustomRules: custom})
		require.NoError(t, err)
		want := Verdict{Matches: []Match{{RuleID: "gcp-api-key"}, {RuleID: "generic-api-key"}, {RuleID: "z-custom"}, {RuleID: "a-custom"}}}
		require.Equal(t, want, fullScanPolicyVerdict(policy, value))
		require.Equal(t, want, policy.Detect(value))
		batch := policy.NewBatchDetector()
		require.Equal(t, want, batch.Detect(value))
	}
}

func TestRequestExclusionsPreserveSharedRegexValidatorsAndDirectCarriers(t *testing.T) {
	fixtures := nativeCatalogFixtures(t)
	defaults, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	canny := fixtures["canny-api-request"].Positive[0]
	calorie := fixtures["calorieninjas-api-request"].Positive[0]
	require.Contains(t, defaults.Detect(canny).Matches, Match{RuleID: "canny-api-request"})
	require.NotContains(t, defaults.Detect(canny).Matches, Match{RuleID: "calorieninjas-api-request"})
	require.Contains(t, defaults.Detect(calorie).Matches, Match{RuleID: "calorieninjas-api-request"})
	require.NotContains(t, defaults.Detect(calorie).Matches, Match{RuleID: "canny-api-request"})
	probes := []string{
		canny, calorie, fixtures["madkudu-api-basic-credential"].Positive[0],
		fixtures["madkudu-api-basic-credential"].Positive[0] + "\n" + canny + "\n" + calorie,
	}
	cases := []struct {
		ids    []string
		policy *CompiledPolicy
	}{
		{ids: []string{"canny-api-request", "calorieninjas-api-request"}},
		{ids: []string{"calorieninjas-api-request"}},
		{ids: []string{"canny-api-request"}},
		{ids: []string{"calorieninjas-api-request", "canny-api-request"}},
		{},
	}
	// Publish every selection first: a shared regex must not share its rule's
	// validator or mutate an earlier policy's exclusions.
	for i := range cases {
		cases[i].policy, err = testPolicyCompiler.CompilePolicy(Policy{DisabledRules: cases[i].ids})
		require.NoError(t, err)
	}
	for _, test := range cases {
		batch := test.policy.NewBatchDetector()
		for _, value := range probes {
			want := fullScanPolicyVerdict(defaults, value)
			want.Matches = slices.DeleteFunc(want.Matches, func(match Match) bool {
				return slices.Contains(test.ids, match.RuleID)
			})
			if len(want.Matches) == 0 {
				want.Matches = nil
			}
			require.Equal(t, want, fullScanPolicyVerdict(test.policy, value))
			require.Equal(t, want, test.policy.Detect(value))
			require.Equal(t, want, batch.Detect(value))
			require.Equal(t, want, batch.Detect(value))
		}
	}
}

func TestNativeExclusionsAcrossBitsetWordBoundaries(t *testing.T) {
	defaults, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	ids, err := CatalogRuleIDs()
	require.NoError(t, err)
	indexes := []int{0, 63, 64, 127, 128, 1023, 1024, len(ids) - 1}
	disabled := make([]string, len(indexes))
	for i, index := range indexes {
		disabled[i] = ids[index]
	}
	policy, err := testPolicyCompiler.CompilePolicy(Policy{DisabledRules: disabled})
	require.NoError(t, err)
	batch := policy.NewBatchDetector()
	fixtures := nativeCatalogFixtures(t)
	for _, index := range indexes {
		// Adjacent IDs, including those in other words, must remain enabled.
		for _, probeIndex := range []int{index, min(index+1, len(ids)-1)} {
			id := ids[probeIndex]
			value := fixtures[id].Positive[0]
			want := fullScanPolicyVerdict(defaults, value)
			require.Contains(t, want.Matches, Match{RuleID: id})
			want.Matches = slices.DeleteFunc(want.Matches, func(match Match) bool {
				return slices.Contains(disabled, match.RuleID)
			})
			if len(want.Matches) == 0 {
				want.Matches = nil
			}
			require.Equal(t, want, policy.Detect(value), "catalog index %d", probeIndex)
			require.Equal(t, want, batch.Detect(value), "batch index %d", probeIndex)
		}
	}
}

func TestNoNativeRulesPreserveNullableCustomRules(t *testing.T) {
	allIDs, err := CatalogRuleIDs()
	require.NoError(t, err)
	for _, test := range []struct {
		name     string
		selected *[]string
		disabled []string
	}{
		{"tenant-excludes-all", nil, allIDs},
		{"global-empty", &[]string{}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			compiler, err := NewPolicyCompiler(test.selected)
			require.NoError(t, err)
			disabled, err := compiler.CompilePolicy(Policy{DisabledRules: test.disabled})
			require.NoError(t, err)
			custom, err := compiler.CompilePolicy(Policy{
				DisabledRules: test.disabled,
				CustomRules:   []CustomRule{{ID: "empty-or-custom", Regex: `^$|CUSTOM`}},
			})
			require.NoError(t, err)
			native := nativeCatalogFixtures(t)["github-pat"].Positive[0]
			batch := custom.NewBatchDetector()
			for _, probe := range []struct {
				value string
				want  Verdict
			}{
				{"", Verdict{Matches: []Match{{RuleID: "empty-or-custom"}}}},
				{native, Verdict{}},
				{native + "\nCUSTOM", Verdict{Matches: []Match{{RuleID: "empty-or-custom"}}}},
			} {
				require.Equal(t, Verdict{}, disabled.Detect(probe.value))
				require.Equal(t, probe.want, custom.Detect(probe.value))
				require.Equal(t, probe.want, batch.Detect(probe.value))
				require.Equal(t, probe.want, batch.Detect(probe.value))
			}
		})
	}
}

func TestPolicyRejectsInvalidDisabledRulesSafely(t *testing.T) {
	const private = "tenant-private-value"
	for _, test := range []struct {
		name string
		ids  []string
	}{
		{"unknown", []string{private}},
		{"invalid", []string{private + "/"}},
		{"empty", []string{""}},
		{"custom-ID", []string{"tenant-custom"}},
		{"duplicate", []string{"generic-api-key", "generic-api-key"}},
		{"partial-valid", []string{"generic-api-key", private}},
		{"oversized-ID", []string{strings.Repeat(private, maxPolicyStringLength)}},
		{"too-many", make([]string, len(nativeRuleSpecs)+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			compiled, err := testPolicyCompiler.CompilePolicy(Policy{
				DisabledRules: test.ids,
				CustomRules:   []CustomRule{{ID: "tenant-custom", Regex: "CUSTOM"}},
			})
			require.Error(t, err)
			require.Nil(t, compiled)
			require.NotContains(t, err.Error(), private)
			require.NotContains(t, err.Error(), "generic-api-key")
			require.NotContains(t, err.Error(), "github-pat")
		})
	}
}

func TestBatchDetectorMatchesCompiledPolicyAndProtectsCache(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{{ID: "custom", Regex: `CUSTOM-[0-9]{3}`}}})
	require.NoError(t, err)
	detector := policy.NewBatchDetector()
	values := []string{
		"production",
		"CUSTOM-123",
		findCatalogRuleWitness(t, policy.catalog, 0),
		"CUSTOM-123",
	}
	for _, value := range values {
		expected := policy.Detect(value)
		assert.Equal(t, expected, detector.Detect(value))
		assert.Equal(t, expected, detector.Detect(value))
	}

	verdict := detector.Detect("CUSTOM-123")
	require.NotEmpty(t, verdict.Matches)
	verdict.Matches[0].RuleID = "changed"
	assert.Equal(t, policy.Detect("CUSTOM-123"), detector.Detect("CUSTOM-123"))
}

func TestPolicyRejectsInvalidCustomRules(t *testing.T) {
	_, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{{ID: "bad rule ID", Regex: "secret"}}})
	require.Error(t, err)

	_, err = testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{{ID: "broken", Regex: "("}}})
	require.Error(t, err)

	_, err = testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{{ID: nativeRuleSpecs[0].ID, Regex: "secret"}}})
	require.Error(t, err)

	// Excluded native rules still reserve their IDs; a custom rule cannot
	// replace the supported native implementation.
	_, err = testPolicyCompiler.CompilePolicy(Policy{DisabledRules: []string{"generic-api-key"}, CustomRules: []CustomRule{{ID: "generic-api-key", Regex: "secret"}}})
	require.Error(t, err)
}

func TestCustomRuleCountLimit(t *testing.T) {
	makePolicy := func(count int) (Policy, string, []Match) {
		policy := Policy{CustomRules: make([]CustomRule, count)}
		values := make([]string, count)
		matches := make([]Match, count)
		for i := range count {
			id := fmt.Sprintf("limit-%02d", i)
			policy.CustomRules[i] = CustomRule{ID: id, Regex: fmt.Sprintf(`LIMIT_%02d_[A-Z]{8}`, i)}
			values[i] = fmt.Sprintf("LIMIT_%02d_ABCDEFGH", i)
			matches[i] = Match{RuleID: id}
		}
		return policy, strings.Join(values, " "), matches
	}
	t.Run("sixteen-valid-rules", func(t *testing.T) {
		input, value, expected := makePolicy(16)
		policy, err := testPolicyCompiler.CompilePolicy(input)
		require.NoError(t, err)
		require.Equal(t, Verdict{Matches: expected}, policy.Detect(value))
		batch := policy.NewBatchDetector()
		require.Equal(t, Verdict{Matches: expected}, batch.Detect(value))
	})
	t.Run("disabled-native-rules-do-not-disable-custom-rules", func(t *testing.T) {
		input, value, expected := makePolicy(16)
		var err error
		input.DisabledRules, err = CatalogRuleIDs()
		require.NoError(t, err)
		value += "\n" + `api_key="r9Q2m7V4x1Z8c6B3n0H5j2L9p4T7w8Y1"`
		policy, err := testPolicyCompiler.CompilePolicy(input)
		require.NoError(t, err)
		require.Equal(t, Verdict{Matches: expected}, policy.Detect(value))
		batch := policy.NewBatchDetector()
		require.Equal(t, Verdict{Matches: expected}, batch.Detect(value))
	})
	t.Run("seventeen-valid-rules-rejected-not-truncated", func(t *testing.T) {
		input, _, _ := makePolicy(17)
		policy, err := testPolicyCompiler.CompilePolicy(input)
		require.Error(t, err)
		require.Nil(t, policy)
	})
}

func TestPolicyCompileErrorsDoNotExposeConfiguration(t *testing.T) {
	const private = "tenant-private-value"
	tests := []struct {
		name  string
		rules []CustomRule
	}{
		{"invalid-regex", []CustomRule{{ID: private, Regex: "(" + private}}},
		{"invalid-id", []CustomRule{{ID: private + "/", Regex: private}}},
		{"long-id", []CustomRule{{ID: strings.Repeat(private, 40), Regex: private}}},
		{"long-regex", []CustomRule{{ID: private, Regex: strings.Repeat(private, 300)}}},
		{"missing-regex", []CustomRule{{ID: private}}},
		{"duplicate-id", []CustomRule{{ID: private, Regex: private}, {ID: private, Regex: private}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiled, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: test.rules})
			require.Error(t, err)
			require.Nil(t, compiled)
			require.NotContains(t, err.Error(), private)
		})
	}
}

func TestPolicyBoundsExpandedRegexPrograms(t *testing.T) {
	t.Run("single-expression", func(t *testing.T) {
		// Under the source-byte limit, but more than a million instructions
		// after counted repetition expansion.
		compiled, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{{
			ID: "wide", Regex: "(?:" + strings.Repeat("ab", 600) + "){1000}",
		}}})
		require.Error(t, err)
		require.Nil(t, compiled)
	})
	t.Run("aggregate", func(t *testing.T) {
		// Each expression fits; only the policy-wide expansion exceeds the cap.
		copies := maxCustomPolicyInstructions/(1000*maxPolicyCustomRules) + 1
		compiled, err := testPolicyCompiler.CompilePolicy(evaluationCustomPolicy(maxPolicyCustomRules, strings.Repeat(`[ab]{1000}`, copies)))
		require.Error(t, err)
		require.Nil(t, compiled)
	})
	t.Run("optimization-caps-still-accepted", func(t *testing.T) {
		// The NFA fallback benchmark is a supported policy, not a resource
		// rejection. Keep this independent of whether a DFA can be built.
		_, err := testPolicyCompiler.CompilePolicy(evaluationCustomPolicy(maxPolicyCustomRules, strings.Repeat(`[ab]{1000}`, 5)))
		require.NoError(t, err)
	})
}

func TestCompiledPolicyProviderAcceptsPolicyChanges(t *testing.T) {
	input := &Policy{CustomRules: []CustomRule{{ID: "tenant", Regex: `TENANT-[0-9]+`}}}
	provider := testPolicyCompiler.NewCompiledPolicyProvider("tenant-a", func(string) (*Policy, bool) { return input, false }, log.NewNopLogger())

	compiled, ok := provider(context.Background())
	require.True(t, ok)
	assert.True(t, compiled.Detect("TENANT-1").Matched())

	input = &Policy{CustomRules: []CustomRule{{ID: "other", Regex: `OTHER-[0-9]+`}}}
	compiled, ok = provider(context.Background())
	require.True(t, ok)
	assert.False(t, compiled.Detect("TENANT-1").Matched())
	assert.True(t, compiled.Detect("OTHER-1").Matched())
}

func TestCompiledPolicyProviderAcceptsInheritedPolicyAfterOverlayRemoval(t *testing.T) {
	input := &Policy{CustomRules: []CustomRule{{ID: "tenant", Regex: `TENANT-[0-9]+`}}}
	inherited := false
	provider := testPolicyCompiler.NewCompiledPolicyProvider("tenant-a", func(string) (*Policy, bool) { return input, inherited }, log.NewNopLogger())

	compiled, ok := provider(context.Background())
	require.True(t, ok)
	assert.True(t, compiled.Detect("TENANT-1").Matched())

	input = &Policy{CustomRules: []CustomRule{{ID: "default", Regex: `DEFAULT-[0-9]+`}}}
	inherited = true
	compiled, ok = provider(context.Background())
	require.True(t, ok)
	assert.False(t, compiled.Detect("TENANT-1").Matched())
	assert.True(t, compiled.Detect("DEFAULT-1").Matched())
}

func BenchmarkCompiledPolicyProviderUnchanged(b *testing.B) {
	policy := &Policy{}
	provider := testPolicyCompiler.NewCompiledPolicyProvider("tenant-a", func(string) (*Policy, bool) { return policy, false }, log.NewNopLogger())
	_, ok := provider(context.Background())
	require.True(b, ok)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = provider(context.Background())
	}
}

func BenchmarkCompiledPolicyProviderPolicyChurn(b *testing.B) {
	policy := &Policy{}
	provider := testPolicyCompiler.NewCompiledPolicyProvider("tenant-a", func(string) (*Policy, bool) { return policy, false }, log.NewNopLogger())

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		policy = &Policy{
			CustomRules: []CustomRule{{
				ID:    fmt.Sprintf("secret.%d", i),
				Regex: fmt.Sprintf("SECRET-%d", i),
			}},
		}
		_, _ = provider(context.Background())
	}
}

func TestPolicyMinimumWidthPreservesCustomRules(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		values     []string
		matched    []bool
	}{
		{"short literal", `ab`, []string{"ab", "a", "ab"}, []bool{true, false, true}},
		{"nullable", `a?`, []string{"", "b", "a"}, []bool{true, true, true}},
		{"empty only", `^$`, []string{"", "a", ""}, []bool{true, false, true}},
		{"unicode literal", `é`, []string{"e", "é"}, []bool{false, true}},
		{"invalid UTF-8", `\x{FFFD}`, []string{"\xff", "\uFFFD", "x"}, []bool{true, true, false}},
		{"rune width", `.{2}`, []string{"é", "éx"}, []bool{false, true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{{ID: "tenant", Regex: test.expression}}})
			require.NoError(t, err)
			batch := policy.NewBatchDetector()
			for i, value := range test.values {
				want := Verdict{}
				if test.matched[i] {
					want.Matches = []Match{{RuleID: "tenant"}}
				}
				require.Equal(t, want, policy.Detect(value))
				require.Equal(t, want, batch.Detect(value))
				require.Equal(t, want, batch.Detect(value))
			}
		})
	}
}

func uniqueFindingVerdict(findings []Match) Verdict {
	var verdict Verdict
	seen := make(map[string]bool)
	for _, finding := range findings {
		if !seen[finding.RuleID] {
			seen[finding.RuleID] = true
			verdict.Matches = append(verdict.Matches, finding)
		}
	}
	return verdict
}

func fullScanPolicyVerdict(policy *CompiledPolicy, value string) Verdict {
	var findings []Match
	for i := range policy.catalog.rules {
		if !policy.disabled.has(uint16(i)) {
			findings = policy.catalog.rules[i].detect(value, findings)
		}
	}
	for i := range policy.custom.rules {
		findings = policy.custom.rules[i].detect(value, findings)
	}
	return uniqueFindingVerdict(findings)
}

func TestPolicyVerdictMatchesFullScan(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{
		{ID: "z-regex", Regex: `S:([A-Za-z0-9]{8})`},
		{ID: "a-capture", Regex: `S:([A-Za-z0-9]{8})`},
		{ID: "empty", Regex: `^$`},
		{ID: "line", Regex: `(?m)^LINE:([a-z]+)$`},
		{ID: "fold", Regex: `(?i)K{2}`},
		{ID: "invalid", Regex: `\x{FFFD}`},
	}})
	require.NoError(t, err)
	batch := policy.NewBatchDetector()
	values := []string{
		"", "GET", "S:aaaaaaaa", "S:aaaaaaaa S:Ab12Cd34", "S:Ab12Cd34 S:aaaaaaaa",
		strings.Repeat("S:Ab12Cd34 ", 8), "before\nLINE:hello\nafter", "xLINE:hello", "KK", "\xff",
		findCatalogRuleWitness(t, policy.catalog, 0) + " S:Ab12Cd34",
		strings.Repeat("ghp_~~~ ", maxKeywordHits+20) + "S:Ab12Cd34",
	}
	for i, value := range values {
		want := fullScanPolicyVerdict(policy, value)
		require.Equal(t, want, policy.Detect(value), "probe %d direct", i)
		require.Equal(t, want, batch.Detect(value), "probe %d batch miss", i)
		require.Equal(t, want, batch.Detect(value), "probe %d batch hit", i)
	}
	// Custom rule order is configuration order, not lexical order or match position.
	require.Equal(t, Verdict{Matches: []Match{{RuleID: "z-regex"}, {RuleID: "a-capture"}}}, policy.Detect("S:aaaaaaaa S:Ab12Cd34"))
	// Each rule reports once even when the value contains multiple matches.
	require.Equal(t, Verdict{Matches: []Match{{RuleID: "z-regex"}, {RuleID: "a-capture"}}},
		policy.Detect("S:aaaaaaaa S:Ab12Cd34 S:Cd34Ef56"))
	baseline, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	witness := findCatalogRuleWitness(t, baseline.catalog, 0)
	nativeFirst := baseline.Detect(witness)
	nativeFirst.Matches = append(nativeFirst.Matches, Match{RuleID: "z-regex"}, Match{RuleID: "a-capture"})
	require.Equal(t, nativeFirst, policy.Detect(witness+" S:Ab12Cd34"))
}

type policyVerdictProbe struct {
	name    string
	value   string
	ruleIDs []string
}

type policyVerdictCase struct {
	name   string
	rules  []CustomRule
	probes []policyVerdictProbe
}

// These bounded expressions are tenant shapes, not catalog-derived expressions.
// Each probe belongs to one policy rather than a Cartesian product of every
// expression and value. The same policies also form the fuzz selector domain.
func customPolicyVerdictCases() []policyVerdictCase {
	return []policyVerdictCase{
		{
			name: "mandatory-literals-and-overlap",
			rules: []CustomRule{
				{ID: "z-window", Regex: `[0-9]{2}CUST:[A-Z]{4}:END`},
				{ID: "a-contained", Regex: `CUST:[A-Z]{4}`},
				{ID: "overlapping-starts", Regex: `aba[0-9]{2}Z`},
			},
			probes: []policyVerdictProbe{
				{"same-keyword-two-rules", "12CUST:ABCD:END", []string{"z-window", "a-contained"}},
				{"configured-not-match-order", "CUST:EFGH 12CUST:ABCD:END", []string{"z-window", "a-contained"}},
				{"keyword-without-required-factor", "12CUST:ABCD:BAD", []string{"a-contained"}},
				{"missing-keyword", "12NONE:ABCD:END", nil},
				{"missing-leading-run", "CUST:ABCD:END", []string{"a-contained"}},
				{"overlapping-start-after-failure", "ababa12Z", []string{"overlapping-starts"}},
				{"near-miss", "ababa1Z", nil},
			},
		},
		{
			name: "mandatory-literal-unicode-fallback",
			rules: []CustomRule{
				{ID: "folded-keyword", Regex: `(?i)secret:[a-z]{4}`},
				{ID: "unicode-alternative", Regex: `(?:MARK:[A-Z]{4}|ΩΩ)`},
			},
			probes: []policyVerdictProbe{
				{"ascii", "SECRET:ABCD", []string{"folded-keyword"}},
				{"keyword-and-run-aliases", "ſecret:abKſ", []string{"folded-keyword"}},
				{"malformed-before-aliases", "\xffſecret:abKſ", []string{"folded-keyword"}},
				{"malformed-in-run", "secret:\xffabc", nil},
				{"different-unicode-letter", "sɘcret:abcd", nil},
				{"ascii-alternative", "MARK:ABCD", []string{"unicode-alternative"}},
				{"keywordless-unicode-alternative", "ΩΩ", []string{"unicode-alternative"}},
				{"malformed-before-unicode-alternative", "\xffΩΩ", []string{"unicode-alternative"}},
			},
		},
		{
			name: "mixed-folding",
			rules: []CustomRule{
				{ID: "scoped", Regex: `(?i:ks)(?-i:X)(?i:ab)y`},
				{ID: "inline", Regex: `(?i)ab(?-i)CD(?i)ks`},
			},
			probes: []policyVerdictProbe{
				{"scoped-ascii", "kSXaBy", []string{"scoped"}},
				{"scoped-unicode-folds", "KſXaBy", []string{"scoped"}},
				{"sensitive-scope", "KſxaBy", nil},
				{"scope-restored", "KſXaBY", nil},
				{"inline-unicode-folds", "aBCDKſ", []string{"inline"}},
				{"inline-sensitive", "aBcdKſ", nil},
				{"invalid-before-match", "\xffKſXaBy", []string{"scoped"}},
			},
		},
		{
			name: "unicode-and-invalid-utf8",
			rules: []CustomRule{
				{ID: "unicode", Regex: `\A\p{Greek}{2}\p{Nd}+\z`},
				{ID: "replacement", Regex: `\A\x{FFFD}{2}\p{L}\z`},
			},
			probes: []policyVerdictProbe{
				{"unicode-digits", "Ωβ١٢", []string{"unicode"}},
				{"ascii-digits", "Ωβ12", []string{"unicode"}},
				{"wrong-letter-class", "Aβ١٢", nil},
				{"missing-digits", "Ωβ", nil},
				{"invalid-bytes", "\xff\xfeé", []string{"replacement"}},
				{"truncated-sequence", "\xe2\x82é", []string{"replacement"}},
				{"literal-replacement", "\uFFFD\uFFFDé", []string{"replacement"}},
				{"one-invalid-byte", "\xffé", nil},
			},
		},
		{
			name: "assertions",
			rules: []CustomRule{
				{ID: "line", Regex: `(?m)^EDGE:[a-z]{2}$`},
				{ID: "text", Regex: `\AEDGE:[a-z]{2}\z`},
				{ID: "word", Regex: `\bBOUND\b`},
				{ID: "nonword", Regex: `\BINNER\B`},
			},
			probes: []policyVerdictProbe{
				{"whole-text", "EDGE:ab", []string{"line", "text"}},
				{"interior-line", "before\nEDGE:ab\nafter", []string{"line"}},
				{"trailing-newline", "EDGE:ab\n", []string{"line"}},
				{"carriage-return", "EDGE:ab\r\n", nil},
				{"not-line-start", "xEDGE:ab", nil},
				{"ascii-word", "!BOUND!", []string{"word"}},
				{"unicode-word-context", "éBOUNDé", []string{"word"}},
				{"invalid-word-context", "\xffBOUND\xff", []string{"word"}},
				{"underscore-context", "_BOUND_", nil},
				{"inside-word", "xINNERy", []string{"nonword"}},
				{"outside-word", "!INNER!", nil},
			},
		},
		{
			name: "nullable-and-empty",
			rules: []CustomRule{
				{ID: "z-empty", Regex: `(?:)`},
				{ID: "a-nullable", Regex: `(?i:k*)`},
				{ID: "empty-text", Regex: `\A\z`},
				{ID: "word-assertion", Regex: `\b`},
				{ID: "impossible-assertion", Regex: `\A\b\z`},
			},
			probes: []policyVerdictProbe{
				{"empty", "", []string{"z-empty", "a-nullable", "empty-text"}},
				{"nonword", "☃", []string{"z-empty", "a-nullable"}},
				{"folded-repeat", "KK", []string{"z-empty", "a-nullable"}},
				{"word", "a a", []string{"z-empty", "a-nullable", "word-assertion"}},
				{"invalid", "\xff", []string{"z-empty", "a-nullable"}},
			},
		},
		{
			name: "regex-matches-ignore-capture-content",
			rules: []CustomRule{
				{ID: "low-entropy", Regex: `C:([a-z]{4})`},
				{ID: "unmatched-capture", Regex: `A:(?:(a+)|b+)`},
				{ID: "optional-capture", Regex: `O:(a)?b`},
				{ID: "empty-capture", Regex: `E:()`},
				{ID: "nested-capture", Regex: `N:((aa)(bc))`},
			},
			probes: []policyVerdictProbe{
				{"low-entropy-match", "C:aaaa", []string{"low-entropy"}},
				{"unmatched-alternative-capture", "A:bbbb", []string{"unmatched-capture"}},
				{"absent-optional-capture", "O:b", []string{"optional-capture"}},
				{"empty-capture-match", "E:", []string{"empty-capture"}},
				{"nested-low-entropy-capture", "N:aabc", []string{"nested-capture"}},
				{"near-miss", "C:aaa", nil},
			},
		},
		{
			name: "wide-and-nested-repetitions",
			rules: []CustomRule{
				{ID: "wide", Regex: `\AW:(ab){2,80}!\z`},
				{ID: "nested", Regex: `\AN:(?:[ab]{2,5}c){2,4}!\z`},
			},
			probes: []policyVerdictProbe{
				{"lower-bound", "W:abab!", []string{"wide"}},
				{"below-lower-bound", "W:ab!", nil},
				{"upper-bound", "W:" + strings.Repeat("ab", 80) + "!", []string{"wide"}},
				{"above-upper-bound", "W:" + strings.Repeat("ab", 81) + "!", nil},
				{"nested-match", "N:" + strings.Repeat("aabc", 3) + "!", []string{"nested"}},
				{"nested-too-many", "N:" + strings.Repeat("aabc", 5) + "!", nil},
				{"inner-too-short", "N:acac!", nil},
			},
		},
		{
			name: "literal-analysis-cap-fallback",
			rules: []CustomRule{{
				ID:    "long-alternatives",
				Regex: "(?:FIRST" + strings.Repeat("x", 256) + "TAIL|SECOND" + strings.Repeat("y", 256) + "TAIL)",
			}},
			probes: []policyVerdictProbe{
				{"first-alternative", "FIRST" + strings.Repeat("x", 256) + "TAIL", []string{"long-alternatives"}},
				{"second-alternative", "SECOND" + strings.Repeat("y", 256) + "TAIL", []string{"long-alternatives"}},
				{"wrong-suffix", "FIRST" + strings.Repeat("x", 256) + "FAIL", nil},
			},
		},
		{
			name: "automaton-cap-fallback",
			// The exact repeat expands beyond the NFA cap. Remembering which of
			// the previous thirteen positions contained 'a' exceeds the DFA cap.
			// Only the resulting consumer verdict is part of the contract.
			rules: []CustomRule{
				{ID: "large-program", Regex: `\AN:(?:abcdefgh){600}\z`},
				{ID: "many-subsets", Regex: `\AD:[ab]*a[ab]{12}!\z`},
			},
			probes: []policyVerdictProbe{
				{"large-program-match", "N:" + strings.Repeat("abcdefgh", 600), []string{"large-program"}},
				{"large-program-near-miss", "N:" + strings.Repeat("abcdefgh", 599) + "abcdefgi", nil},
				{"many-subsets-match", "D:a" + strings.Repeat("b", 12) + "!", []string{"many-subsets"}},
				{"many-subsets-miss", "D:" + strings.Repeat("b", 13) + "!", nil},
				{"many-subsets-wrong-distance", "D:" + strings.Repeat("b", 12) + "a!", nil},
				{"many-subsets-late", "D:" + strings.Repeat("ab", 256) + "a" + strings.Repeat("b", 12) + "!", []string{"many-subsets"}},
			},
		},
		{
			name: "long-values-and-late-matches",
			rules: []CustomRule{
				{ID: "late", Regex: `LATE:([A-Z0-9]{8})`},
			},
			probes: []policyVerdictProbe{
				{"long-clean", strings.Repeat("~", 131072), nil},
				{"late-match", strings.Repeat("~", 131072) + "LATE:AB12CD34", []string{"late"}},
				{"early-and-late-match", "LATE:AAAAAAAA" + strings.Repeat("~", 131072) + "LATE:AB12CD34", []string{"late"}},
				{"late-near-miss", strings.Repeat("~", 131072) + "LATE:AB12CD3", nil},
				{"invalid-before-late-match", "\xff" + strings.Repeat("~", 4096) + "LATE:AB12CD34", []string{"late"}},
				{"keyword-overflow-many-matches", strings.Repeat("LATE:AAAAAAAA ", maxKeywordHits+1) + "LATE:AB12CD34 LATE:EF56GH78", []string{"late"}},
			},
		},
	}
}

func TestPolicyCustomExpressionsMatchFullScan(t *testing.T) {
	compiler, err := NewPolicyCompiler(&[]string{})
	require.NoError(t, err)
	for _, test := range customPolicyVerdictCases() {
		t.Run(test.name, func(t *testing.T) {
			policy, err := compiler.CompilePolicy(Policy{CustomRules: test.rules})
			require.NoError(t, err)
			wants := make([]Verdict, len(test.probes))
			batch := policy.NewBatchDetector()
			for i, probe := range test.probes {
				t.Run(probe.name, func(t *testing.T) {
					for _, id := range probe.ruleIDs {
						wants[i].Matches = append(wants[i].Matches, Match{RuleID: id})
					}
					// Explicit positive and negative verdicts keep agreement with
					// the unfiltered oracle from becoming vacuous.
					require.Equal(t, wants[i], fullScanPolicyVerdict(policy, probe.value), "oracle")
					require.Equal(t, wants[i], policy.Detect(probe.value), "direct")
					require.Equal(t, wants[i], batch.Detect(probe.value), "batch miss")
					require.Equal(t, wants[i], batch.Detect(probe.value), "batch hit")
				})
			}
			t.Run("parallel-consumers", func(t *testing.T) {
				for worker := range 4 {
					t.Run(fmt.Sprintf("consumer-%d", worker), func(t *testing.T) {
						t.Parallel()
						// CompiledPolicy is shared; mutable BatchDetector state is not.
						batch := policy.NewBatchDetector()
						for round := range 3 {
							for offset := range test.probes {
								index := (worker + round + offset) % len(test.probes)
								probe := test.probes[index]
								require.Equal(t, wants[index], policy.Detect(probe.value), "%s direct", probe.name)
								require.Equal(t, wants[index], batch.Detect(probe.value), "%s batch miss", probe.name)
								require.Equal(t, wants[index], batch.Detect(probe.value), "%s batch hit", probe.name)
							}
						}
					})
				}
			})
		})
	}
}

func FuzzPolicyVerdictMatchesFullScan(f *testing.F) {
	baseline, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(f, err)
	custom, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{
		{ID: "unanchored", Regex: `T:([A-Za-z0-9]{8})`},
		{ID: "line", Regex: `(?m)^T:[A-Za-z0-9]{8}$`},
		{ID: "nullable", Regex: `(?i)k*`},
		{ID: "binary", Regex: `\x{FFFD}{1,2}`},
	}})
	require.NoError(f, err)
	exclusions, err := testPolicyCompiler.CompilePolicy(Policy{DisabledRules: []string{"generic-api-key", "canny-api-request", "github-pat"}})
	require.NoError(f, err)
	type seed struct {
		value string
		which uint8
	}
	seen := make(map[seed]bool)
	addSeed := func(value string, which uint8) {
		key := seed{value, which}
		if !seen[key] {
			seen[key] = true
			f.Add(value, which)
		}
	}
	policies := []*CompiledPolicy{baseline, custom, exclusions}
	for i := range baseline.catalog.rules {
		addSeed(findCatalogRuleWitness(f, baseline.catalog, i), 0)
	}
	fixtures := nativeCatalogFixtures(f)
	for _, spec := range nativeRuleSpecs {
		fixture := fixtures[spec.ID]
		for _, value := range fixture.Positive {
			addSeed(value, 2)
		}
		for _, value := range fixture.Negative {
			addSeed(value, 2)
		}
	}
	for _, value := range []string{"", "GET", "us-east-1", "T:aaaaaaaa\nT:Ab12Cd34", "KK", "\xff\xfe", "xT:Ab12Cd34"} {
		addSeed(value, 0)
		addSeed(value, 1)
	}
	// Fuzz both the expression selection and its input without compiling
	// unbounded generated regexes or rebuilding the shared production catalog.
	customOnlyCompiler, err := NewPolicyCompiler(&[]string{})
	require.NoError(f, err)
	for _, test := range customPolicyVerdictCases() {
		policy, err := customOnlyCompiler.CompilePolicy(Policy{CustomRules: test.rules})
		require.NoError(f, err)
		policies = append(policies, policy)
		for _, probe := range test.probes {
			addSeed(probe.value, uint8(len(policies)-1))
		}
	}
	f.Fuzz(func(t *testing.T, value string, which uint8) {
		t.Parallel()
		policy := policies[int(which)%len(policies)]
		want := fullScanPolicyVerdict(policy, value)
		require.Equal(t, want, policy.Detect(value))
		batch := policy.NewBatchDetector()
		require.Equal(t, want, batch.Detect(value))
		require.Equal(t, want, batch.Detect(value))
	})
}

func BenchmarkPolicyEvaluation(b *testing.B) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(b, err)
	cases := []struct{ name, value string }{
		{"method", "GET"},
		{"region", "us-east-1"},
		{"service", "checkout-service"},
		{"clean-url", "https://api.example.com/v1/orders/123456?region=us-east-1"},
	}
	for index, rule := range policy.catalog.rules {
		id := rule.id
		witness := findCatalogRuleWitness(b, policy.catalog, index)
		cases = append(
			cases,
			struct{ name, value string }{id, witness},
			struct{ name, value string }{id + "-repeated", strings.Repeat(witness+"\n", 8)},
			struct{ name, value string }{id + "-late-128KiB", strings.Repeat("~", 131072) + witness},
		)
	}
	for _, test := range cases {
		b.Run(test.name+"/direct", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				benchmarkVerdict = policy.Detect(test.value)
			}
		})
		b.Run(test.name+"/batch-hit", func(b *testing.B) {
			detector := policy.NewBatchDetector()
			detector.Detect(test.value)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				benchmarkVerdict = detector.Detect(test.value)
			}
		})
	}
}

func TestCustomPlanReusePreservesAcceptanceAndIDs(t *testing.T) {
	const expression = `(MEM:([A-Z0-9]{8}))`
	policy, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{
		{ID: "same-a", Regex: expression},
		{ID: "same-b", Regex: expression},
	}})
	require.NoError(t, err)
	for _, probe := range []struct {
		name, value string
		ids         []string
	}{
		{"low-entropy-body", "界MEM:AAAAAAAAé", []string{"same-a", "same-b"}},
		{"diverse-body", "界MEM:AB12CD34é", []string{"same-a", "same-b"}},
		{"short-body", "MEM:AB12CD3", nil},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			var want Verdict
			for _, id := range probe.ids {
				want.Matches = append(want.Matches, Match{RuleID: id})
			}
			batch := policy.NewBatchDetector()
			for range 32 {
				require.Equal(t, want, policy.Detect(probe.value))
				require.Equal(t, want, batch.Detect(probe.value))
			}
			require.Equal(t, want, fullScanPolicyVerdict(policy, probe.value))
		})
	}
}

func TestCustomPlanReusePreservesPublishedSnapshots(t *testing.T) {
	const oldExpression = `OLDMEM_[A-Z]{8}`
	input := &Policy{CustomRules: []CustomRule{
		{ID: "first", Regex: oldExpression},
		{ID: "second", Regex: oldExpression},
	}}
	provider := testPolicyCompiler.NewCompiledPolicyProvider("reuse-tenant", func(string) (*Policy, bool) {
		return input, false
	}, log.NewNopLogger())
	old, ok := provider(context.Background())
	require.True(t, ok)
	input = &Policy{CustomRules: []CustomRule{
		{ID: "first", Regex: `NEWMEM_[A-Z]{8}`},
		{ID: "second", Regex: oldExpression},
	}}
	current, ok := provider(context.Background())
	require.True(t, ok)
	for _, snapshot := range []struct {
		name   string
		policy *CompiledPolicy
		oldIDs []Match
		newIDs []Match
	}{
		{"retained", old, []Match{{RuleID: "first"}, {RuleID: "second"}}, nil},
		{"current", current, []Match{{RuleID: "second"}}, []Match{{RuleID: "first"}}},
	} {
		t.Run(snapshot.name, func(t *testing.T) {
			t.Parallel()
			batch := snapshot.policy.NewBatchDetector()
			for range 32 {
				require.Equal(t, Verdict{Matches: snapshot.oldIDs}, snapshot.policy.Detect("OLDMEM_ABCDEFGH"))
				require.Equal(t, Verdict{Matches: snapshot.newIDs}, snapshot.policy.Detect("NEWMEM_ABCDEFGH"))
				require.Equal(t, Verdict{Matches: snapshot.oldIDs}, batch.Detect("OLDMEM_ABCDEFGH"))
			}
		})
	}
}

func TestFeatureConfigSelectionSerialization(t *testing.T) {
	fixtures := nativeCatalogFixtures(t)
	value := fixtures["github-pat"].Positive[0] + "\n" + fixtures["gcp-api-key"].Positive[0]
	all, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	for _, codec := range []struct {
		name      string
		marshal   func(any) ([]byte, error)
		unmarshal func([]byte, any) error
	}{
		{"yaml", yaml.Marshal, yaml.UnmarshalStrict},
		{"json", json.Marshal, json.Unmarshal},
	} {
		for _, test := range []struct {
			name  string
			input string
			all   bool
			want  Verdict
		}{
			{"omitted", `{}`, true, all.Detect(value)},
			{"null", `{"enabled_rules":null}`, true, all.Detect(value)},
			{"list", `{"enabled_rules":["github-pat"]}`, false, Verdict{Matches: []Match{{RuleID: "github-pat"}}}},
			{"empty", `{"enabled_rules":[]}`, false, Verdict{}},
		} {
			t.Run(codec.name+"/"+test.name, func(t *testing.T) {
				input := []byte(test.input)
				for range 2 {
					var cfg FeatureConfig
					require.NoError(t, codec.unmarshal(input, &cfg))
					require.NoError(t, cfg.Validate())
					require.Equal(t, test.all, cfg.EnabledRules == nil)
					compiler, err := NewPolicyCompiler(cfg.EnabledRules)
					require.NoError(t, err)
					policy, err := compiler.CompilePolicy(Policy{})
					require.NoError(t, err)
					require.Equal(t, test.want, policy.Detect(value))
					input, err = codec.marshal(cfg)
					require.NoError(t, err)
				}
			})
		}
		t.Run(codec.name+"/programmatic-empty", func(t *testing.T) {
			cfg := FeatureConfig{EnabledRules: new([]string)}
			for range 2 {
				compiler, err := NewPolicyCompiler(cfg.EnabledRules)
				require.NoError(t, err)
				policy, err := compiler.CompilePolicy(Policy{})
				require.NoError(t, err)
				require.Equal(t, Verdict{}, policy.Detect(value))
				encoded, err := codec.marshal(cfg)
				require.NoError(t, err)
				var decoded FeatureConfig
				require.NoError(t, codec.unmarshal(encoded, &decoded))
				cfg = decoded
			}
		})
	}
}

func TestFeatureConfigRejectsInvalidSelectionsWhenDetectionDisabled(t *testing.T) {
	const private = "operator-private-value"
	for _, test := range []struct {
		name string
		ids  []string
	}{
		{"unknown", []string{private}},
		{"empty-ID", []string{""}},
		{"duplicate", []string{"github-pat", "github-pat"}},
		{"partial-valid", []string{"github-pat", private}},
		{"oversized-ID", []string{strings.Repeat(private, maxPolicyStringLength)}},
		{"too-many", make([]string, len(nativeRuleSpecs)+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := FeatureConfig{EnabledRules: &test.ids}
			err := cfg.Validate()
			require.Error(t, err)
			require.NotContains(t, err.Error(), private)
			require.NotContains(t, err.Error(), "github-pat")
			compiler, err := NewPolicyCompiler(cfg.EnabledRules)
			require.Error(t, err)
			require.Nil(t, compiler)
			require.NotContains(t, err.Error(), private)
			require.NotContains(t, err.Error(), "github-pat")
		})
	}
}

func TestPolicyCompilerSnapshotsSelectionAndIsolatesTenants(t *testing.T) {
	selected := []string{"github-pat", "gcp-api-key"}
	compiler, err := NewPolicyCompiler(&selected)
	require.NoError(t, err)
	selected[0] = "generic-api-key"
	selected = nil
	other, err := NewPolicyCompiler(&[]string{"generic-api-key"})
	require.NoError(t, err)
	fixtures := nativeCatalogFixtures(t)
	value := fixtures["github-pat"].Positive[0] + "\n" + fixtures["gcp-api-key"].Positive[0]
	baseline, err := compiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	first, err := compiler.CompilePolicy(Policy{DisabledRules: []string{"gcp-api-key"}})
	require.NoError(t, err)
	second, err := compiler.CompilePolicy(Policy{DisabledRules: []string{"github-pat"}})
	require.NoError(t, err)
	otherPolicy, err := other.CompilePolicy(Policy{})
	require.NoError(t, err)
	defaults, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	otherWant := defaults.Detect(value)
	otherWant.Matches = slices.DeleteFunc(otherWant.Matches, func(match Match) bool {
		return match.RuleID != "generic-api-key"
	})
	if len(otherWant.Matches) == 0 {
		otherWant.Matches = nil
	}
	for _, test := range []struct {
		policy *CompiledPolicy
		want   Verdict
	}{
		{baseline, Verdict{Matches: []Match{{RuleID: "gcp-api-key"}, {RuleID: "github-pat"}}}},
		{first, Verdict{Matches: []Match{{RuleID: "github-pat"}}}},
		{second, Verdict{Matches: []Match{{RuleID: "gcp-api-key"}}}},
		{otherPolicy, otherWant},
	} {
		batch := test.policy.NewBatchDetector()
		require.Equal(t, test.want, test.policy.Detect(value))
		require.Equal(t, test.want, batch.Detect(value))
		require.Equal(t, test.want, batch.Detect(value))
	}
}

func TestPolicyCompilerValidatesInactiveNativeIDs(t *testing.T) {
	compiler, err := NewPolicyCompiler(&[]string{"github-pat"})
	require.NoError(t, err)
	policy, err := compiler.CompilePolicy(Policy{DisabledRules: []string{"gcp-api-key", "generic-api-key"}})
	require.NoError(t, err)
	value := nativeCatalogFixtures(t)["github-pat"].Positive[0]
	want := Verdict{Matches: []Match{{RuleID: "github-pat"}}}
	require.Equal(t, want, policy.Detect(value))
	batch := policy.NewBatchDetector()
	require.Equal(t, want, batch.Detect(value))
	for _, input := range []Policy{
		{DisabledRules: []string{"gcp-api-key", "gcp-api-key"}},
		{CustomRules: []CustomRule{{ID: "gcp-api-key", Regex: "CUSTOM"}}},
	} {
		rejected, err := compiler.CompilePolicy(input)
		require.Error(t, err)
		require.Nil(t, rejected)
		require.NotContains(t, err.Error(), "gcp-api-key")
	}
}

func TestSelectedProviderFallbackAndRecoveryNeverExpandCoverage(t *testing.T) {
	fixtures := nativeCatalogFixtures(t)
	value := fixtures["github-pat"].Positive[0] + "\n" + fixtures["gcp-api-key"].Positive[0] + "\nCUSTOM"
	for _, selected := range [][]string{{"github-pat"}, {}} {
		t.Run(fmt.Sprintf("selected-%d", len(selected)), func(t *testing.T) {
			compiler, err := NewPolicyCompiler(&selected)
			require.NoError(t, err)
			var native []Match
			if len(selected) != 0 {
				native = []Match{{RuleID: "github-pat"}}
			}
			withCustom := append(slices.Clone(native), Match{RuleID: "tenant-custom"})
			input := &Policy{DisabledRules: []string{"unsupported-rule"}}
			provider := compiler.NewCompiledPolicyProvider("selected-tenant", func(string) (*Policy, bool) {
				return input, false
			}, log.NewNopLogger())
			for _, step := range []struct {
				input *Policy
				want  []Match
			}{
				{input, native},
				{&Policy{DisabledRules: []string{"gcp-api-key"}, CustomRules: []CustomRule{{ID: "tenant-custom", Regex: "CUSTOM"}}}, withCustom},
				{&Policy{CustomRules: []CustomRule{{ID: "gcp-api-key", Regex: "CUSTOM"}}}, withCustom},
				{nil, native},
			} {
				input = step.input
				policy, ok := provider(context.Background())
				require.True(t, ok)
				want := Verdict{Matches: step.want}
				require.Equal(t, want, policy.Detect(value))
				batch := policy.NewBatchDetector()
				require.Equal(t, want, batch.Detect(value))
			}
		})
	}
}
