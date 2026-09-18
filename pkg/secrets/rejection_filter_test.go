package secrets

import (
	"math/rand"
	"regexp"
	"regexp/syntax"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseFilterRegexp(t testing.TB, expression string) *syntax.Regexp {
	t.Helper()
	re, err := syntax.Parse(expression, syntax.Perl)
	require.NoError(t, err)
	return re
}

func TestRejectionFilter(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		anchored   bool
		values     map[string]bool
	}{
		{
			name:       "literal unanchored",
			expression: `abc`,
			values:     map[string]bool{"": false, "xxabcxx": true, "abx": false},
		},
		{
			name:       "literal anchored",
			expression: `abc`,
			anchored:   true,
			values:     map[string]bool{"abc": true, "abcx": true, "xabc": false},
		},
		{
			name:       "class",
			expression: `[a-c][0-9]`,
			values:     map[string]bool{"b4": true, "--c9--": true, "d4": false},
		},
		{
			name:       "alternation",
			expression: `cat|dog`,
			values:     map[string]bool{"xxdog": true, "catnap": true, "cow": false},
		},
		{
			name:       "end assertion waits for end of input",
			expression: `abc$`,
			values:     map[string]bool{"abc": true, "xxabc": true, "abcx": false, "abc\n": false},
		},
		{
			name:       "end assertion or explicit delimiter",
			expression: `abc(?:$|[ ;])`,
			values:     map[string]bool{"abc": true, "abc;next": true, "abc next": true, "abcx": false, "abc.": false},
		},
		{
			name:       "end-only paths cannot consume later runes",
			expression: `abc$$|ab$c`,
			values:     map[string]bool{"abc": true, "abcx": false, "ab": false},
		},
		{
			name:       "word boundary relaxed",
			expression: `\bapi\b`,
			values:     map[string]bool{"xapiy": true, "zzz": false},
		},
		{
			name:       "wide repeat relaxed",
			expression: `a{2,40}b`,
			anchored:   true,
			values:     map[string]bool{strings.Repeat("a", 50) + "b": true, "ab": false},
		},
		{
			name:       "fold case",
			expression: `(?i:AbC)`,
			values:     map[string]bool{"--aBc--": true, "abd": false},
		},
		{
			name:       "non ASCII literal",
			expression: `é+`,
			values:     map[string]bool{"eeee": false, "é": true},
		},
		{
			name:       "empty expression",
			expression: ``,
			values:     map[string]bool{"": true, "anything": true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter, ok := newRejectionFilter(parseFilterRegexp(t, test.expression), test.anchored)
			require.True(t, ok)
			require.NotNil(t, filter)
			for value, want := range test.values {
				if test.anchored {
					assert.Equal(t, want, filter.mayMatchPrefix(value), "%q", value)
				} else {
					assert.Equal(t, want, filter.mayMatch(value), "%q", value)
				}
			}
		})
	}
}

func TestRejectionFilterUnicodeTransitions(t *testing.T) {
	for _, test := range []struct {
		name, expression string
		values           map[string]bool
	}{
		{"ascii candidates in unicode", `\btoken:[A-Z0-9]{8}\b`, map[string]bool{
			"東京 token:AB12CD34 終": true, "東京 token:short 終": false, "\xff token:AB12CD34": true,
		}},
		{"one rune consumes multibyte input", `A.B`, map[string]bool{
			"AéB": true, "A🙂B": true, "A\xffB": true, "AééB": false,
		}},
		{"two runes and malformed bytes", `A..B`, map[string]bool{
			"Aé🙂B": true, "A\xff\xfeB": true, "AéB": false,
		}},
		{"folded ASCII aliases", `(?i)KS`, map[string]bool{
			"Kſ": true, "KS": true, "kſ": true, "%%": false,
		}},
		{"unicode classes", `\p{Greek}+=\d{2}`, map[string]bool{
			"Ωβ=42": true, "not-a-match": false,
		}},
		{"replacement rune", `\x{FFFD}z`, map[string]bool{
			"\xffz": true, "\uFFFDz": true, "xz": false,
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			re := regexp.MustCompile(test.expression)
			filter, ok := newRejectionFilter(parseFilterRegexp(t, test.expression), false)
			require.True(t, ok)
			for value, want := range test.values {
				require.Equal(t, want, re.MatchString(value), "test witness")
				require.Equal(t, want, filter.mayMatch(value))
			}
		})
	}
}

func TestFallbackFiltersPreserveLateAndUnicodeMatches(t *testing.T) {
	catalog := compileTestCatalog(
		t,
		catalogRuleSpec{ID: "folded", Regex: `\b(?i:token):([A-Z0-9]{8})`, Keywords: []string{"token"}, SecretGroup: 1},
		catalogRuleSpec{ID: "entropy", Regex: `\b(?i:token):([A-Z0-9]{8})`, Keywords: []string{"token"}, SecretGroup: 1, Entropy: 2},
	)
	policy := &CompiledPolicy{catalog: catalog}
	for _, prefix := range []string{
		// Many complete starts with guidance still available, before overflow.
		strings.Repeat("token ", 32),
		strings.Repeat("token ", maxKeywordHits+32),
		strings.Repeat("λ 東京 token ", maxKeywordHits+32),
		"\xff" + strings.Repeat("token ", maxKeywordHits+32),
	} {
		for _, test := range []struct {
			suffix string
			want   Verdict
		}{
			{"token:short", Verdict{}},
			{"token:AB12CD34", Verdict{Matches: []Match{{RuleID: "folded"}, {RuleID: "entropy"}}}},
			{"toKen:AB12CD34", Verdict{Matches: []Match{{RuleID: "folded"}, {RuleID: "entropy"}}}},
			{"token:AAAAAAAA token:AB12CD34", Verdict{Matches: []Match{{RuleID: "folded"}, {RuleID: "entropy"}}}},
		} {
			value := prefix + test.suffix
			require.Equal(t, test.want, fullScanPolicyVerdict(policy, value))
			assertKeywordGuidedMatchesFullScan(t, catalog, value)
			require.Equal(t, test.want, policy.Detect(value))
			batch := policy.NewBatchDetector()
			require.Equal(t, test.want, batch.Detect(value))
			require.Equal(t, test.want, batch.Detect(value))
		}
	}
}

func TestCompiledRuleFilterPreservesLongValueMatches(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		value      string
		matches    int
	}{
		{"late literal", `tok_[0-9]{2}`, strings.Repeat("~", 4096) + "tok_42", 1},
		{"failed prefix before match", `tok_[0-9]{2}`, "tok_xx" + strings.Repeat("~", 4096) + "tok_42", 1},
		{"multiple matches", `tok_[0-9]{2}`, "tok_12" + strings.Repeat("~", 4096) + "tok_42", 2},
		{"literal absent", `tok_[0-9]{2}`, strings.Repeat("~", 4096), 0},
		{"wrong case", `tok_[0-9]{2}`, strings.Repeat("~", 4096) + "TOK_42", 0},
		{"folded literal", `(?i)tok_[0-9]{2}`, strings.Repeat("~", 4096) + "TOK_42", 1},
		{"invalid UTF-8 before literal", `tok_[0-9]{2}`, "\xff" + strings.Repeat("~", 4096) + "tok_42", 1},
		{"invalid UTF-8 after literal", `tok_.`, strings.Repeat("~", 4096) + "tok_\xff", 1},
		{"replacement rune matches invalid UTF-8", "\uFFFD[0-9]", strings.Repeat("~", 4096) + "\xff7", 1},
		{"literal followed by boundary", `tok_\b[0-9]{2}`, strings.Repeat("~", 4096) + "tok_42", 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rule, err := compileRuleSpec(catalogRuleSpec{ID: "test", Regex: test.expression}, nil)
			require.NoError(t, err)
			var candidates ruleSet
			candidates.add(0)
			rules := []compiledRule{rule}
			want := evaluateRuleSet(rules, candidates, test.value, nil, allRuleMatches, nil)
			require.Len(t, want, test.matches)
			hits := keywordHits{invalid: !isASCIIKeyword(test.value)}
			assert.Equal(t, want, evaluateRuleSet(rules, candidates, test.value, &hits, allRuleMatches, nil))
		})
	}
}

func TestRejectionFilterConstructionCaps(t *testing.T) {
	re := parseFilterRegexp(t, `abcdef`)

	filter, ok := newRejectionFilterWithLimits(re, false, 4, 1024)
	assert.False(t, ok)
	assert.Nil(t, filter)

	filter, ok = newRejectionFilterWithLimits(re, false, 4096, 3)
	assert.False(t, ok)
	assert.Nil(t, filter)

	filter, ok = newRejectionFilter(parseFilterRegexp(t, strings.Repeat("a", rejectionNFACap)), false)
	assert.False(t, ok)
	assert.Nil(t, filter)

	filter, ok = newRejectionFilter(parseFilterRegexp(t, `[ab]*a[ab]{10}`), false)
	assert.False(t, ok)
	assert.Nil(t, filter)
}

func TestCatalogRejectionFiltersAreSound(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	contexts := []string{"", " ", "\n", `"`, "'", "=", "x", "_", "-", ".", "key ", "token=", "https://", strings.Repeat("a", 60)}

	for index, spec := range nativeRuleSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			rule := &catalog.rules[index]
			if rule.filter == nil {
				return // A capped filter is bypassed; the fallback test covers those rules.
			}
			parsed := parseFilterRegexp(t, spec.Regex)
			re := regexp.MustCompile(spec.Regex)
			anchoredRE := regexp.MustCompile(`^(?:` + spec.Regex + `)`)
			rng := rand.New(rand.NewSource(catalogWitnessSeed(spec.ID)))
			probes := append([]string(nil), nativeCatalogFixtures(t)[spec.ID].Positive...)
			for range 200 {
				probes = append(probes, generateProbe(parsed, rng))
			}
			for iteration, probe := range probes {
				left := contexts[(iteration*2)%len(contexts)]
				right := contexts[(iteration*2+1)%len(contexts)]
				values := []string{
					probe,
					left + probe + right,
					probe + left + probe,
					probe[:len(probe)/2] + right + probe,
					probe + "\n" + left + probe + "\n",
					strings.ToUpper(probe),
				}
				for _, value := range values {
					if rule.fallbackFilter != nil && re.MatchString(value) {
						assert.True(t, rule.fallbackFilter.mayMatch(value), "fallback filter rejected a regex match")
					}
					if rule.plan.strategy == strategyAnchored {
						if anchoredRE.MatchString(value) {
							assert.True(t, rule.filter.mayMatchPrefix(value), "prefix filter rejected regex match %q", value)
						}
					} else if re.MatchString(value) {
						assert.True(t, rule.filter.mayMatch(value), "filter rejected regex match %q", value)
					}
				}
			}
		})
	}
}

func TestCatalogRejectionFilterBudgetAndFallback(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)

	totalBytes, filtered := 0, 0
	for index := range catalog.rules {
		filter := catalog.rules[index].filter
		if fallback := catalog.rules[index].fallbackFilter; fallback != nil {
			assert.LessOrEqual(t, len(fallback.next), rejectionDFACap*fallback.classes)
			totalBytes += len(fallback.next)*2 + len(fallback.acceptAtEnd)
		}
		if filter == nil {
			witness := findCatalogRuleWitness(t, catalog, index)
			findings := assertCatalogRuleMatchesFullScan(t, catalog, index, witness)
			require.True(t, hasRuleFinding(findings, catalog.rules[index].id), "capped filters must conservatively fall back")
			continue
		}
		filtered++
		assert.LessOrEqual(t, len(filter.next), rejectionDFACap*filter.classes)
		totalBytes += len(filter.next)*2 + len(filter.acceptAtEnd)
	}
	t.Logf("%d primary rejection filters; primary and fallback tables use %d bytes", filtered, totalBytes)
	// Catalog growth may add independent tables. Bound average shared storage
	// per supported rule while the per-filter caps above enforce fallback.
	assert.Less(t, totalBytes, len(catalog.rules)*(16<<10))
}

func TestRejectionFiltersRejectUnrelatedURL(t *testing.T) {
	catalog := compileTestCatalog(
		t,
		catalogRuleSpec{ID: "anchored", Regex: `\btoken:[A-Z]{8}\b`, Keywords: []string{"token"}},
		catalogRuleSpec{ID: "full", Regex: `secret:[A-Z]{8}`, Keywords: []string{"secret"}},
		catalogRuleSpec{ID: "window", Regex: `\b[a-z]{2}=token:[A-Z]{8}\b`, Keywords: []string{"token"}},
	)
	value := "https://api.example.com/v1/orders/123456?region=us-east-1"
	for _, rule := range catalog.rules {
		require.NotNil(t, rule.filter)
		if rule.plan.strategy == strategyAnchored {
			assert.False(t, rule.filter.mayMatchPrefix(value), rule.id)
		} else {
			assert.False(t, rule.filter.mayMatch(value), rule.id)
		}
	}
	assert.Empty(t, assertKeywordGuidedMatchesFullScan(t, catalog, value))
}
