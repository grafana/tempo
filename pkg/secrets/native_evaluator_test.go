package secrets

import (
	"math/rand"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func catalogMatcherKeywords() []matcherKeyword {
	keywords := make([]matcherKeyword, 0, len(nativeRuleSpecs)*2)
	for index, spec := range nativeRuleSpecs {
		if spec.Regex == "" {
			continue
		}
		for _, keyword := range effectiveKeywords(spec) {
			keywords = append(keywords, matcherKeyword{value: keyword, rule: uint16(index)})
		}
	}
	return keywords
}

func TestCatalogSharedExpressionsPreserveRuleAcceptance(t *testing.T) {
	const expression = `token_([a-z]+):([a-z]+)`
	catalog, err := compileCatalog([]catalogRuleSpec{
		{ID: "left", Regex: expression, Keywords: []string{"token_"}, SecretGroup: 1, Validate: func(s string) bool { return s == "left" }},
		{ID: "right", Regex: expression, Keywords: []string{"token_"}, SecretGroup: 2, Validate: func(s string) bool { return s == "right" }},
	})
	require.NoError(t, err)
	for _, test := range []struct {
		value string
		want  []Match
	}{
		{"token_left:other", []Match{{RuleID: "left"}}},
		{"token_other:right", []Match{{RuleID: "right"}}},
		{"token_left:right", []Match{{RuleID: "left"}, {RuleID: "right"}}},
		{"token_other:other", nil},
	} {
		var hits keywordHits
		require.Equal(t, test.want, catalog.detect(test.value, &hits, nil))
	}
}

func TestCatalogSharedPlansPreserveKeywordSemantics(t *testing.T) {
	const expression = `(?i:tok_)([a-z]{1,8}):(secret|public)`
	catalog, err := compileCatalog([]catalogRuleSpec{
		{ID: "capture", Regex: expression, Keywords: []string{"tok_"}, SecretGroup: 1, Validate: func(s string) bool { return s == "good" }},
		{ID: "window", Regex: expression, Keywords: []string{"secret", "public"}, SecretGroup: 2, Validate: func(s string) bool { return s == "secret" }},
		{ID: "context", Regex: expression, Keywords: []string{"tok_"}, SecretGroup: 1, ValidateContext: func(value string, start, _ int, secret string) contextValidation {
			return contextValidation{accepted: secret == "good" && strings.HasSuffix(value[:start], "allow ")}
		}},
	})
	require.NoError(t, err)
	for _, test := range []struct {
		name  string
		value string
		want  []Match
	}{
		{"capture-only", "tok_good:public", []Match{{RuleID: "capture"}}},
		{"window-only", "tok_bad:secret", []Match{{RuleID: "window"}}},
		{"shared-acceptance", "allow tok_good:secret", []Match{{RuleID: "capture"}, {RuleID: "window"}, {RuleID: "context"}}},
		{"later-accepted", "tok_bad:public allow tok_good:secret", []Match{{RuleID: "capture"}, {RuleID: "window"}, {RuleID: "context"}}},
		{"unicode-fold", "allow toK_good:secret", []Match{{RuleID: "capture"}, {RuleID: "window"}, {RuleID: "context"}}},
		{"overflow", strings.Repeat("tok_ ", maxKeywordHits+20) + "allow tok_good:secret", []Match{{RuleID: "capture"}, {RuleID: "window"}, {RuleID: "context"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			reference := assertKeywordGuidedMatchesFullScan(t, catalog, test.value)
			require.Equal(t, test.want, uniqueFindingVerdict(reference).Matches)
		})
	}
}

func TestAnchoredExecutionPreservesLeadingContext(t *testing.T) {
	for _, test := range []struct {
		name       string
		expression string
		good       string
		bad        string
	}{
		{"suffix-only", `(?i:tok_)([A-Z]{4})\b`, "TOK_GOOD", "TOK_BADX"},
		{"internal-boundary", `(?i:tok)\b:([A-Z]{4})\b`, "TOK:GOOD", "TOK:BADX"},
		{"word-boundary", `\b(?i:tok_)([A-Z]{4})\b`, "TOK_GOOD", "TOK_BADX"},
		{"non-boundary", `\B(?i:tok_)([A-Z]{4})\b`, "TOK_GOOD", "TOK_BADX"},
		{"optional-boundary", `(?:\b)?(?i:tok_)([A-Z]{4})\b`, "TOK_GOOD", "TOK_BADX"},
		{"alternate-boundary", `(?:\b|\B)(?i:tok_)([A-Z]{4})\b`, "TOK_GOOD", "TOK_BADX"},
		{"line-anchor", `(?m)^(?i:tok_)([A-Z]{4})$`, "TOK_GOOD", "TOK_BADX"},
		{"text-anchor", `\A(?i:tok_)([A-Z]{4})\z`, "TOK_GOOD", "TOK_BADX"},
	} {
		t.Run(test.name, func(t *testing.T) {
			catalog, err := compileCatalog([]catalogRuleSpec{
				{ID: "existence", Regex: test.expression, Keywords: []string{"tok"}},
				{ID: "capture", Regex: test.expression, Keywords: []string{"tok"}, SecretGroup: 1, Validate: func(s string) bool { return s == "GOOD" }},
				{ID: "context", Regex: test.expression, Keywords: []string{"tok"}, SecretGroup: 1, ValidateContext: func(value string, start, end int, secret string) contextValidation {
					return contextValidation{accepted: secret == "GOOD" && value[start:end] == test.good && !strings.HasPrefix(value, "deny")}
				}},
			})
			require.NoError(t, err)
			for _, value := range []string{
				test.good,
				"x" + test.good,
				"\n" + test.good,
				"界" + test.good,
				"\xff" + test.good,
				test.bad + " " + test.good,
				test.good + "\n" + test.good,
				"deny " + test.good,
				strings.Repeat("tok ", maxKeywordHits+20) + test.good,
			} {
				assertKeywordGuidedMatchesFullScan(t, catalog, value)
			}
		})
	}
}

func TestLiteralRunMatcherPreservesRegexpCaptures(t *testing.T) {
	for _, test := range []struct {
		name       string
		expression string
	}{
		{"delimiter", `\b(TOK_[A-Z]{2,4})(?:$|[^A-Z_])`},
		{"word-boundary", `\b(TOK_[A-Z]{2,4})\b`},
		{"end-of-text", `\b(TOK_[A-Z]{2,4})$`},
		{"no-tail", `\b(TOK_[A-Z]{2,4})`},
		{"unbounded", `\b(TOK_[A-Z]+)(?:$|[^A-Z_])`},
		{"fixed-width", `\b(TOK_[A-Z]{4})(?:$|[^A-Z_])`},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := syntax.Parse(test.expression, syntax.Perl)
			require.NoError(t, err)
			matcher := newLiteralRunMatcher(parsed)
			require.NotNil(t, matcher)
			reference := regexp.MustCompile(test.expression)
			catalog, err := compileCatalog([]catalogRuleSpec{
				{ID: "existence", Regex: test.expression, Keywords: []string{"TOK_"}},
				{ID: "capture", Regex: test.expression, Keywords: []string{"TOK_"}, SecretGroup: 1, Validate: func(s string) bool { return s == "TOK_AB" }},
				{ID: "context", Regex: test.expression, Keywords: []string{"TOK_"}, SecretGroup: 1, ValidateContext: func(value string, start, end int, secret string) contextValidation {
					return contextValidation{accepted: secret == "TOK_AB" && strings.HasSuffix(value[:start], "allow ") && strings.HasPrefix(value[start:end], secret)}
				}},
			})
			require.NoError(t, err)
			for _, value := range []string{
				"TOK_A",
				"TOK_AB",
				"TOK_ABCD",
				"TOK_ABCDE",
				"xTOK_AB",
				"界TOK_AB界TOK_CD",
				"\xffTOK_AB\xffTOK_CD",
				"TOK_AB\nTOK_CD",
				"TOK_CD allow TOK_AB",
				"TOK_AB,TOK_CD,TOK_EF",
			} {
				var actual [][]int
				for scan := 0; scan < len(value); {
					offset := strings.Index(value[scan:], matcher.prefix)
					if offset < 0 {
						break
					}
					start := scan + offset
					end, captureEnd, matched := matcher.match(value, start)
					if matched {
						actual = append(actual, []int{start, end, start, captureEnd})
						scan = end
					} else {
						scan = start + 1
					}
				}
				require.Equal(t, reference.FindAllStringSubmatchIndex(value, -1), actual)
				assertKeywordGuidedMatchesFullScan(t, catalog, value)
			}
			assertKeywordGuidedMatchesFullScan(t, catalog, strings.Repeat("TOK_ ", maxKeywordHits+20)+"allow TOK_AB")
		})
	}
}

func TestLiteralRunFallbackPreservesBacktracking(t *testing.T) {
	for _, expression := range []string{
		`\b(TOK_[A-Z]{2,4})(?:$|[A-Z])`,
		`\b(TOK_[A-Z]{2,4}?)`,
		`\b(TOK_[A-Z-]{2,4})\b`,
		`\b((?i:TOK_)[A-Z]{2,4})(?:$|[^A-Z_])`,
	} {
		catalog, err := compileCatalog([]catalogRuleSpec{
			{ID: "capture", Regex: expression, Keywords: []string{"TOK_"}, SecretGroup: 1, Validate: func(s string) bool { return s == "TOK_AB" || s == "TOK_AB-" }},
		})
		require.NoError(t, err)
		for _, value := range []string{"TOK_ABC", "TOK_AB--", "TOK_AB--X", "TOK_ABC", "TOK_BAD TOK_ABC"} {
			assertKeywordGuidedMatchesFullScan(t, catalog, value)
		}
	}
}

func TestDerivedKeywordsAreRequiredByGeneratedMatches(t *testing.T) {
	for _, spec := range nativeRuleSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			parsed, err := syntax.Parse(spec.Regex, syntax.Perl)
			require.NoError(t, err)

			configured := make([]string, 0, len(spec.Keywords))
			for _, keyword := range spec.Keywords {
				configured = append(configured, strings.ToLower(keyword))
			}
			keywords := effectiveKeywords(spec)
			if consumesKeyword(parsed, configured) || slices.Equal(keywords, spec.Keywords) {
				return
			}

			re, err := regexp.Compile(spec.Regex)
			require.NoError(t, err)
			rng := rand.New(rand.NewSource(catalogWitnessSeed(spec.ID)))
			matches := 0
			for range 40 {
				probe := generateProbe(parsed, rng)
				if !re.MatchString(probe) {
					continue
				}
				matches++
				assert.Truef(
					t,
					containsAnyKeyword(strings.ToLower(probe), keywords),
					"matching probe %q does not contain a derived keyword %q",
					probe,
					keywords,
				)
			}
			assert.NotZero(t, matches, "generated probes did not exercise the derived keyword")
		})
	}
}

func TestNativeCatalogMatcherMatchesReference(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	reference, err := newKeywordMatcher(catalogMatcherKeywords())
	require.NoError(t, err)
	for index, spec := range nativeRuleSpecs {
		if spec.Regex == "" {
			continue
		}
		for _, keyword := range effectiveKeywords(spec) {
			var ahoCandidates, pairCandidates ruleSet
			value := strings.ToUpper(keyword)
			reference.match(value, &ahoCandidates)
			catalog.pairMatcher.match(value, &pairCandidates)
			assert.Truef(t, ahoCandidates.has(uint16(index)), "Aho-Corasick did not select rule %q keyword %q", spec.ID, keyword)
			assert.Equalf(t, ahoCandidates, pairCandidates, "pair matcher differed for rule %q keyword %q", spec.ID, keyword)
		}
	}
}

func TestCatalogPairMatcherSingleAndUnicodeKeywords(t *testing.T) {
	keywords := []matcherKeyword{
		{value: "a", rule: 0},
		{value: "bc", rule: 1},
		{value: "σ", rule: 2},
	}
	reference, err := newKeywordMatcher(keywords)
	require.NoError(t, err)
	pair, err := newCatalogPairMatcher(keywords)
	require.NoError(t, err)

	for _, value := range []string{"", "A", "xxBCxx", "éA", "Σ", "prefix σ suffix", string([]byte{0xff, 'A'})} {
		var expected, actual ruleSet
		reference.match(value, &expected)
		pair.match(value, &actual)
		assert.Equal(t, expected, actual, value)
	}
}

func TestCatalogPairMatcherDuplicateKeywordOutputs(t *testing.T) {
	keywords := []matcherKeyword{
		{value: "token", rule: 0},
		{value: "token", rule: 1},
		{value: "token", rule: 1},
		{value: "other", rule: 2},
	}
	reference, err := newKeywordMatcher(keywords)
	require.NoError(t, err)
	pair, err := newCatalogPairMatcher(keywords)
	require.NoError(t, err)

	for _, value := range []string{"no match", "prefix TOKEN suffix", "token and other"} {
		var expected, actual ruleSet
		reference.match(value, &expected)
		pair.match(value, &actual)
		assert.Equal(t, expected, actual, value)
	}
}

func TestCatalogPairMatcherLargeCollisionInventory(t *testing.T) {
	var keywords []matcherKeyword
	for i := range 160 {
		word := string([]byte{byte('a' + i/26), byte('a' + i%26)})
		keywords = append(
			keywords,
			matcherKeyword{value: word, rule: uint16(i)},
			matcherKeyword{value: word, rule: uint16(i + 256)},
		)
	}
	reference, err := newKeywordMatcher(keywords)
	require.NoError(t, err)
	pair, err := newCatalogPairMatcher(keywords)
	require.NoError(t, err)
	var values []string
	for i := range 160 {
		word := string([]byte{byte('a' + i/26), byte('a' + i%26)})
		values = append(values, word, strings.ToUpper(word))
	}
	values = append(values, "fK", "aſ", "no-match", "\xfffk")
	for _, value := range values {
		var expected, actual ruleSet
		reference.match(value, &expected)
		pair.match(value, &actual)
		require.Equal(t, expected, actual, "large inventory candidate selection")
	}
}

func TestCatalogPairMatcherRejectsAliasedNonASCII(t *testing.T) {
	keywords := []matcherKeyword{{value: "bc", rule: 0}}
	reference, err := newKeywordMatcher(keywords)
	require.NoError(t, err)
	pair, err := newCatalogPairMatcher(keywords)
	require.NoError(t, err)

	for _, value := range []string{
		string([]byte{0xe2, 0xe3}),
		string([]byte{0xff, 'B', 'C', 0xfe}),
		"πBCλ",
	} {
		var expected, actual ruleSet
		reference.match(value, &expected)
		pair.match(value, &actual)
		assert.Equal(t, expected, actual, value)
	}
}

func TestCatalogPairMatcherLookupBoundary(t *testing.T) {
	for _, test := range []struct {
		name  string
		count int
	}{
		{name: "compact", count: 16},
		{name: "dense", count: 17},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Length order opposes pair order, so direct certificate indices
			// cannot double as sorted lookup positions. A duplicate output also
			// moves its pair behind the direct certificates.
			keywords := make([]matcherKeyword, 0, test.count+1)
			for i := range test.count {
				word := strings.Repeat(string(rune('z'-i)), i+2)
				keywords = append(keywords, matcherKeyword{value: word, rule: uint16(i)})
			}
			keywords = append(keywords, matcherKeyword{value: keywords[0].value, rule: uint16(test.count)})
			reference, err := newKeywordMatcher(keywords)
			require.NoError(t, err)
			matcher, err := newCatalogPairMatcher(keywords)
			require.NoError(t, err)

			for _, keyword := range keywords[:test.count] {
				for _, value := range []string{
					keyword.value,
					"prefix " + strings.ToUpper(keyword.value) + " suffix",
					"\xff" + keyword.value + "\xfe",
					keyword.value[:len(keyword.value)-1] + "\x80",
					strings.ReplaceAll(strings.ReplaceAll(keyword.value, "k", "K"), "s", "ſ"),
				} {
					var expected, actual ruleSet
					reference.match(value, &expected)
					matcher.match(value, &actual)
					require.Equal(t, expected, actual, "candidate selection across compact lookup boundary")
				}
			}
		})
	}
}

func TestNativeEvaluatorUsesCaptureGroups(t *testing.T) {
	tests := []struct {
		name  string
		spec  catalogRuleSpec
		value string
	}{
		{
			name:  "first nonempty capture",
			spec:  catalogRuleSpec{ID: "capture", Regex: `key=(?:([A-Z]+)|([0-9]+))`},
			value: "key=1234",
		},
		{
			name:  "explicit capture",
			spec:  catalogRuleSpec{ID: "capture", Regex: `key=([A-Z]+):([0-9]+)`, SecretGroup: 2},
			value: "key=ABCD:1234",
		},
		{
			name:  "gitleaks allow annotation is telemetry",
			spec:  catalogRuleSpec{ID: "annotated", Regex: `secret`},
			value: "secret gitleaks:allow",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rule, err := compileRuleSpec(test.spec, nil)
			require.NoError(t, err)
			assert.Equal(t, []Match{{RuleID: test.spec.ID}}, rule.detect(test.value, nil))
		})
	}
}

func TestNativeEvaluatorValidatesCapturedCandidates(t *testing.T) {
	for _, test := range []struct {
		name    string
		entropy float64
	}{
		{name: "zero-entropy"},
		{name: "with-entropy", entropy: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			validate := func(secret string) bool { return secret == "key:GOOD" }
			catalog := compileTestCatalog(
				t,
				catalogRuleSpec{ID: "full", Regex: `(key:[A-Z]{3,4})(?:$|[^A-Za-z0-9])`, Keywords: []string{"key"}, SecretGroup: 1, Entropy: test.entropy, Validate: validate},
				catalogRuleSpec{ID: "anchored", Regex: `\b(key:[A-Z]{3,4})(?:$|[^A-Za-z0-9])`, Keywords: []string{"key"}, SecretGroup: 1, Entropy: test.entropy, Validate: validate},
				catalogRuleSpec{ID: "window", Regex: `\b[a-z]{2}=(key:[A-Z]{3,4})(?:$|[^A-Za-z0-9])`, Keywords: []string{"key"}, SecretGroup: 1, Entropy: test.entropy, Validate: validate},
			)
			policy := &CompiledPolicy{catalog: catalog}
			batch := policy.NewBatchDetector()
			for _, probe := range []struct {
				value string
				valid bool
			}{
				{value: "aa=key:BAD"},
				{value: "aa=key:GOODX"},
				{value: "aa=key:GOOD", valid: true},
				{value: "aa=key:BAD aa=key:GOOD", valid: true},
				{value: "aa=key:GOOD aa=key:BAD", valid: true},
				{value: "aa=key:GOOD aa=key:GOOD", valid: true},
				{value: "\xff aa=key:BAD aa=key:GOOD", valid: true},
				{value: strings.Repeat("key ", maxKeywordHits+20) + "aa=key:BAD"},
				{value: strings.Repeat("key ", maxKeywordHits+20) + "aa=key:BAD aa=key:GOOD", valid: true},
			} {
				var want Verdict
				if probe.valid {
					want.Matches = []Match{{RuleID: "full"}, {RuleID: "anchored"}, {RuleID: "window"}}
				}
				findings := assertKeywordGuidedMatchesFullScan(t, catalog, probe.value)
				require.Equal(t, want, uniqueFindingVerdict(findings), "unfiltered capture validation")
				require.Equal(t, want, policy.Detect(probe.value), "direct")
				require.Equal(t, want, batch.Detect(probe.value), "batch miss")
				require.Equal(t, want, batch.Detect(probe.value), "batch hit")
			}
		})
	}
}

func TestNativeEvaluatorEntropyThresholdIsStrict(t *testing.T) {
	entropy := shannonEntropy("aabb")
	atThreshold, err := compileRuleSpec(catalogRuleSpec{ID: "entropy", Regex: `(aabb)`, Entropy: entropy}, nil)
	require.NoError(t, err)
	assert.Empty(t, atThreshold.detect("aabb", nil))

	belowThreshold, err := compileRuleSpec(catalogRuleSpec{ID: "entropy", Regex: `(aabb)`, Entropy: entropy - 0.001}, nil)
	require.NoError(t, err)
	assert.Len(t, belowThreshold.detect("aabb", nil), 1)
}

func TestNativeEvaluatorEntropyPreservesUnicodeSemantics(t *testing.T) {
	below, err := compileRuleSpec(catalogRuleSpec{ID: "entropy", Regex: `(éa)`, Entropy: 1.1}, nil)
	require.NoError(t, err)
	assert.Empty(t, below.detect("éa", nil))

	above, err := compileRuleSpec(catalogRuleSpec{ID: "entropy", Regex: `(éa)`, Entropy: 1}, nil)
	require.NoError(t, err)
	assert.Len(t, above.detect("éa", nil), 1)
}

func TestNativeEvaluatorCaptureAndEntropyVerdicts(t *testing.T) {
	for _, test := range []struct {
		name   string
		rules  []catalogRuleSpec
		probes []policyVerdictProbe
	}{
		{
			name: "capture-selection-and-strict-entropy",
			rules: []catalogRuleSpec{
				{ID: "z-zero", Regex: `C:(?:([a-z]{4})|([A-Z0-9]{4}))`},
				{ID: "a-default", Regex: `C:(?:([a-z]{4})|([A-Z0-9]{4}))`, Entropy: 1},
				{ID: "second-group", Regex: `C:(?:([a-z]{4})|([A-Z0-9]{4}))`, Entropy: 1, SecretGroup: 2},
				{ID: "first-group", Regex: `C:(?:([a-z]{4})|([A-Z0-9]{4}))`, Entropy: 1, SecretGroup: 1},
				{ID: "below-threshold", Regex: `C:(?:([a-z]{4})|([A-Z0-9]{4}))`, Entropy: 0.999},
			},
			probes: []policyVerdictProbe{
				{"rejected", "C:aaaa", []string{"z-zero"}},
				{"at-threshold", "C:aabb", []string{"z-zero", "below-threshold"}},
				{"first-group", "C:abcd", []string{"z-zero", "a-default", "first-group", "below-threshold"}},
				{"second-group", "C:AB12", []string{"z-zero", "a-default", "second-group", "below-threshold"}},
				{"rejected-then-accepted", "C:aabb C:AB12", []string{"z-zero", "a-default", "second-group", "below-threshold"}},
				{"order-and-dedup", strings.Repeat("C:AB12 C:abcd ", 8), []string{"z-zero", "a-default", "second-group", "first-group", "below-threshold"}},
			},
		},
		{
			name: "empty-and-nested-captures",
			rules: []catalogRuleSpec{
				{ID: "first-nonempty", Regex: `P:()(ab)(cd)`, Entropy: 1.5},
				{ID: "explicit-group", Regex: `P:()(ab)(cd)`, Entropy: 0.5, SecretGroup: 3},
				{ID: "outer-group", Regex: `N:((aa)(bc))`, Entropy: 1.25},
				{ID: "inner-group", Regex: `N:((aa)(bc))`, Entropy: 0.1, SecretGroup: 2},
				{ID: "empty-falls-back", Regex: `E:()`, Entropy: 0.5},
				{ID: "explicit-empty", Regex: `E:()`, Entropy: 0.5, SecretGroup: 1},
				{ID: "empty-entropy", Regex: `()`, Entropy: 0.5},
			},
			probes: []policyVerdictProbe{
				{"first-nonempty-not-full-match", "P:abcd", []string{"explicit-group"}},
				{"outer-before-inner", "N:aabc", []string{"outer-group"}},
				{"default-empty-versus-selected-empty", "E:", []string{"empty-falls-back"}},
				{"absent", "P:ab", nil},
				{"empty", "", nil},
			},
		},
		{
			name: "alternation-and-greediness",
			rules: []catalogRuleSpec{
				{ID: "short-first", Regex: `A:(a|abcd)`, Entropy: 0.5, SecretGroup: 1},
				{ID: "long-first", Regex: `A:(abcd|a)`, Entropy: 0.5, SecretGroup: 1},
				{ID: "lazy", Regex: `Q:([a-d]+?)`, Entropy: 0.5, SecretGroup: 1},
				{ID: "greedy", Regex: `Q:([a-d]+)`, Entropy: 0.5, SecretGroup: 1},
			},
			probes: []policyVerdictProbe{
				{"alternative-priority", "A:abcd", []string{"long-first"}},
				{"later-alternative-match", "A:aaaa A:abcd", []string{"long-first"}},
				{"short-only", "A:a", nil},
				{"greedy-capture", "Q:abcd", []string{"greedy"}},
				{"low-entropy-greedy-capture", "Q:aaaa", nil},
			},
		},
		{
			name: "unicode-capture-context",
			rules: []catalogRuleSpec{
				{ID: "capture", Regex: `CAP:([A-Z0-9]{8})(?:[^A-Z0-9]|$)`, SecretGroup: 1, Entropy: 2},
			},
			probes: []policyVerdictProbe{
				{"rejected-first-then-accepted", "界CAP:AAAAAAAAéCAP:AB12CD34😀", []string{"capture"}},
				{"accepted-capture-excludes-context", "界CAP:AB12CD34é", []string{"capture"}},
				{"context-must-not-raise-capture-entropy", "界CAP:AAAAAAAAé", nil},
			},
		},
		{
			name: "same-regex-distinct-acceptance",
			rules: []catalogRuleSpec{
				{ID: "same-a", Regex: `(MEM:([A-Z0-9]{8}))`, SecretGroup: 2},
				{ID: "same-b", Regex: `(MEM:([A-Z0-9]{8}))`, SecretGroup: 2},
				{ID: "body", Regex: `(MEM:([A-Z0-9]{8}))`, SecretGroup: 2, Entropy: 1},
				{ID: "whole", Regex: `(MEM:([A-Z0-9]{8}))`, SecretGroup: 1, Entropy: 1},
			},
			probes: []policyVerdictProbe{
				{"low-entropy-body", "界MEM:AAAAAAAAé", []string{"same-a", "same-b", "whole"}},
				{"diverse-body", "界MEM:AB12CD34é", []string{"same-a", "same-b", "body", "whole"}},
				{"short-body", "MEM:AB12CD3", nil},
			},
		},
		{
			name: "long-values-and-late-matches",
			rules: []catalogRuleSpec{
				{ID: "late-entropy", Regex: `LATE:([A-Z0-9]{8})`, Entropy: 2.5, SecretGroup: 1},
				{ID: "late-zero", Regex: `LATE:([A-Z0-9]{8})`},
			},
			probes: []policyVerdictProbe{
				{"long-clean", strings.Repeat("~", 131072), nil},
				{"late-match", strings.Repeat("~", 131072) + "LATE:AB12CD34", []string{"late-entropy", "late-zero"}},
				{"rejected-first-late-match", "LATE:AAAAAAAA" + strings.Repeat("~", 131072) + "LATE:AB12CD34", []string{"late-entropy", "late-zero"}},
				{"late-near-miss", strings.Repeat("~", 131072) + "LATE:AB12CD3", nil},
				{"invalid-before-late-match", "\xff" + strings.Repeat("~", 4096) + "LATE:AB12CD34", []string{"late-entropy", "late-zero"}},
				{"keyword-overflow-rejected-then-two-accepted", strings.Repeat("LATE:AAAAAAAA ", maxKeywordHits+1) + "LATE:AB12CD34 LATE:EF56GH78", []string{"late-entropy", "late-zero"}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			catalog, err := compileCatalog(test.rules)
			require.NoError(t, err)
			policy := &CompiledPolicy{catalog: catalog}
			batch := policy.NewBatchDetector()
			for _, probe := range test.probes {
				t.Run(probe.name, func(t *testing.T) {
					var want Verdict
					for _, id := range probe.ruleIDs {
						want.Matches = append(want.Matches, Match{RuleID: id})
					}
					require.Equal(t, want, fullScanPolicyVerdict(policy, probe.value), "oracle")
					require.Equal(t, want, policy.Detect(probe.value), "direct")
					require.Equal(t, want, batch.Detect(probe.value), "batch miss")
					require.Equal(t, want, batch.Detect(probe.value), "batch hit")
				})
			}
		})
	}
}

func TestNativeEvaluatorValidatesOriginalContext(t *testing.T) {
	validate := func(value string, _, _ int, secret string) contextValidation {
		return contextValidation{accepted: secret == "key:GOOD" && !strings.Contains(value, "deny")}
	}
	catalog := compileTestCatalog(
		t,
		catalogRuleSpec{ID: "full", Regex: `(key:[A-Z]{3,4})(?:$|[^A-Za-z0-9])`, Keywords: []string{"key"}, SecretGroup: 1, ValidateContext: validate},
		catalogRuleSpec{ID: "anchored", Regex: `\b(key:[A-Z]{3,4})(?:$|[^A-Za-z0-9])`, Keywords: []string{"key"}, SecretGroup: 1, ValidateContext: validate},
		catalogRuleSpec{ID: "window", Regex: `\b[a-z]{2}=(key:[A-Z]{3,4})(?:$|[^A-Za-z0-9])`, Keywords: []string{"key"}, SecretGroup: 1, ValidateContext: validate},
	)
	policy := &CompiledPolicy{catalog: catalog}
	for _, probe := range []struct {
		name  string
		value string
		valid bool
	}{
		{name: "accepted", value: "aa=key:GOOD", valid: true},
		{name: "capture-rejected", value: "aa=key:BAD"},
		{name: "distant-context", value: "aa=key:GOOD" + strings.Repeat(" ", 1024) + "deny"},
		{name: "later-accepted", value: "aa=key:BAD aa=key:GOOD", valid: true},
		{name: "overflow-context", value: strings.Repeat("key ", maxKeywordHits+20) + "aa=key:GOOD deny"},
		{name: "unicode-context", value: "Key aa=key:GOOD deny"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			var ids []string
			if probe.valid {
				ids = []string{"full", "anchored", "window"}
			}
			assertOptimizerHoldout(t, policy, probe.value, ids)
		})
	}
}

func TestGenericAPIKeyCandidateFilters(t *testing.T) {
	policy := &CompiledPolicy{catalog: compileTestCatalog(t, genericRuleSpecs...)}
	const body = "r9Q2m7V4x1Z8c6B3n0H5j2L9p4T7w8Y1"
	const assigned = `api_key="` + body + `"`
	for _, probe := range []struct {
		name  string
		value string
		valid bool
	}{
		{name: "unknown-assigned", value: assigned, valid: true},
		{name: "misleading-keyword-in-optional-prefix", value: "api" + strings.Repeat("z", 30) + assigned, valid: true},
		{name: "valid-candidate-beyond-filter-lookahead", value: `api_key="` + strings.Repeat(body, 16) + `"`, valid: true},
		{name: "lob-shaped-generic-only", value: `api_key="live_` + `7d42b1e960acf8352ed4a697bc0138fe592"`, valid: true},
		{name: "bare-candidate", value: body},
		{name: "low-entropy", value: `api_key="` + `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`},
		{name: "secret-identifier", value: `api_key="` + `QwErTyUiOpAsDfGhJkLzXcVbNm"`},
		{name: "match-identifier", value: `primary_key="` + body + `"`},
		{name: "secret-stopword", value: `api_key="` + body + `example"`},
		{name: "global-secret-regex", value: `api_key="true` + body + `"`},
		{name: "global-stopword", value: `api_key="` + `014df517-39d1-4453-b7b3-9930c563627c"`},
		{name: "distant-line-context", value: assigned + strings.Repeat(" ", 1024) + "--mount=type=secret,"},
		{name: "import-line-context", value: `import { publicName } from 'module'; ` + assigned},
		{name: "later-candidate", value: `primary_key="` + body + `" ` + assigned, valid: true},
		{name: "following-line-not-context", value: assigned + "\n--mount=type=secret,", valid: true},
		{name: "terminated-line-not-prefix", value: "--mount=type=secret,\n" + assigned + "\n", valid: true},
		{name: "unterminated-line-retains-prefix", value: "--mount=type=secret,\n" + assigned},
		{name: "multiline-final-line-truncation", value: "api_key=\n\"" + body + "\" --mount=type=secret,", valid: true},
		{name: "unrelated-unicode-around-ascii-match", value: "café " + assigned + " 東京", valid: true},
		{name: "folded-unicode-keyword", value: "ſecret=\"" + body + "\"", valid: true},
		{name: "folded-unicode-identifier-prefix", value: "K" + assigned, valid: true},
		{name: "filtered-line-followed-by-valid", value: strings.Repeat(assigned+" ", 8) + "--mount=type=secret,\n" + assigned + "\n", valid: true},
		{name: "multiline-rejection-does-not-hide-later-line", value: "api_key=\n\"" + body + "\" --mount=type=secret,\n" + assigned + "\n", valid: true},
		{name: "rejected-range-still-consumes-cross-line-match", value: assigned + " --mount=type=secret, api_key=\napi_key=" + body + "\n"},
		{name: "overflowed-keywords-find-final-candidate", value: strings.Repeat("API KEY ", 256) + assigned, valid: true},
		{name: "overflowed-unicode-keywords-find-final-candidate", value: strings.Repeat("café API KEY 東京 ", 256) + assigned, valid: true},
		{name: "overflowed-filtered-line-followed-by-valid", value: strings.Repeat(assigned+" ", 160) + "--mount=type=secret,\n" + assigned + "\n", valid: true},
		{name: "overflowed-rejected-range-consumes-cross-line-match", value: strings.Repeat(assigned+" ", 160) + "--mount=type=secret, api_key=\napi_key=" + body + "\n"},
		{name: "path-evidence-absent", value: `LICENSE = "` + assigned, valid: true},
		{name: "inline-directive-is-telemetry", value: assigned + " # gitleaks:allow", valid: true},
	} {
		t.Run(probe.name, func(t *testing.T) {
			var ids []string
			if probe.valid {
				ids = []string{"generic-api-key"}
			}
			assertOptimizerHoldout(t, policy, probe.value, ids)
		})
	}
}

func TestGenericFiltersPreserveSpecificAndCustomFindings(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{
		CustomRules: []CustomRule{{
			ID: "custom-generic", Regex: `(ghp_[A-Za-z0-9]{36})`,
		}},
	})
	require.NoError(t, err)
	const body = "r9Q2m7V4x1Z8c6B3n0H5j2L9p4T7w8Y1"
	ordinary := `api_key="ghp_` + body + body[:4] + `"`
	for _, probe := range []struct {
		name    string
		value   string
		generic bool
	}{
		{name: "additional-heuristic", value: ordinary, generic: true},
		{name: "secret-filter-isolation", value: `api_key="ghp_example` + body[:29] + `"`},
		{name: "line-filter-isolation", value: "--mount=type=secret, " + ordinary},
	} {
		t.Run(probe.name, func(t *testing.T) {
			want := Verdict{}
			if probe.generic {
				want.Matches = []Match{{RuleID: "generic-api-key"}}
			}
			want.Matches = append(want.Matches, Match{RuleID: "github-pat"}, Match{RuleID: "custom-generic"})
			require.Equal(t, want, fullScanPolicyVerdict(policy, probe.value), "unfiltered")
			require.Equal(t, want, policy.Detect(probe.value), "direct")
			batch := policy.NewBatchDetector()
			require.Equal(t, want, batch.Detect(probe.value), "batch miss")
			require.Equal(t, want, batch.Detect(probe.value), "batch hit")
		})
	}
}

func TestGenericExclusionFiltersPreserveRegexpSemantics(t *testing.T) {
	values := []string{
		"", "true", "true1", "1false2", "1null", "null1", "primary_key=value",
		"access_id=value", "secret_size=32", "keyfile", "MONKEY", "MoNkEy",
		"Keyfile", "ſecret_size", "İD", "\xffkeyfile", "ékeyfile東京",
		"A=\nB=", "A=\nB=\n", "A=\nB=x", "a=\nb=", "A=\nb=",
		"${TOKEN}", "{{ value }}", "$12", "/Users/test/path", "abc.123",
	}
	for _, fixture := range nativeCatalogFixtures(t) {
		values = append(values, fixture.Positive...)
		values = append(values, fixture.Negative...)
	}
	for _, test := range []struct{ name, pattern string }{
		{"secret", genericSecretFilterRegex},
		{"match", genericMatchFilterRegex},
		{"assertions-and-flags", `(?i:^key$)|(?-i:^TOKEN$)|secret\z`},
		{"dfa-limit-fallback", `[ab]*a[ab]{10}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			exact := regexp.MustCompile(test.pattern)
			filter := newGenericExclusionFilter(test.pattern)
			check := func(value string) {
				t.Helper()
				require.Equal(t, exact.MatchString(value), filter.MatchString(value), "exclusion acceptance")
			}
			for _, value := range values {
				check(value)
			}
			parsed, err := syntax.Parse(test.pattern, syntax.Perl)
			require.NoError(t, err)
			branches := []*syntax.Regexp{parsed}
			if parsed.Op == syntax.OpAlternate {
				branches = parsed.Sub
			}
			rng := rand.New(rand.NewSource(71))
			for _, branch := range branches {
				for range 8 {
					value := generateProbe(branch, rng)
					for _, context := range []string{"", "x", "\n", "K", "ſ", "\xff"} {
						check(context + value)
						check(value + context)
					}
				}
			}
		})
	}
}

func TestGenericStopwordsUseLowercaseNotSimpleFold(t *testing.T) {
	for _, probe := range []struct {
		name  string
		value string
	}{
		{name: "ascii-uppercase", value: "EXAMPLE"},
		{name: "long-s-not-ascii-s", value: "ſample"},
		{name: "kelvin-lowercases-to-ascii-k", value: "faKe"},
		{name: "dotted-i-lowercases-to-ascii-i", value: "gİthub"},
		{name: "combining-mark-breaks-substring", value: "sam\u0301ple"},
		{name: "invalid-utf8-keeps-following-match", value: "\xffexAMPle"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			// Independent pinned semantics, not the catalog's simple-fold oracle.
			lower := strings.ToLower(probe.value)
			want := false
			for _, word := range genericStopwords {
				if strings.Contains(lower, word) {
					want = true
					break
				}
			}
			require.Equal(t, want, containsGenericStopword(probe.value), "stopword acceptance")
		})
	}
}

func FuzzCatalogPairMatcherMatchesAhoCorasick(f *testing.F) {
	for _, seed := range []string{
		"clean",
		"AKIA" + "QWERTYUIOPASDFGH",
		"xoxb-" + "1234567890-1234567890123-abcdefghijklmnopqrstuvwx",
		"prefix Key suffix",
		"prefix Σ suffix",
		string([]byte{0xff, 0xfe, 's', 'k'}),
	} {
		f.Add(seed)
	}
	catalog, err := testPolicyCompiler.catalog()
	if err != nil {
		f.Fatal(err)
	}
	reference, err := newKeywordMatcher(catalogMatcherKeywords())
	if err != nil {
		f.Fatal(err)
	}
	for index, spec := range nativeRuleSpecs {
		f.Add(findCatalogRuleWitness(f, catalog, index))
		for _, keyword := range spec.Keywords {
			f.Add("prefix " + strings.ToUpper(keyword) + " suffix")
		}
	}
	f.Fuzz(func(t *testing.T, value string) {
		var ahoCandidates, pairCandidates ruleSet
		reference.match(value, &ahoCandidates)
		catalog.pairMatcher.match(value, &pairCandidates)
		assert.Equal(t, ahoCandidates, pairCandidates)
	})
}

var benchmarkCandidateRules ruleSet

func BenchmarkCatalogCandidateSelection(b *testing.B) {
	catalog, err := testPolicyCompiler.catalog()
	if err != nil {
		b.Fatal(err)
	}
	reference, err := newKeywordMatcher(catalogMatcherKeywords())
	if err != nil {
		b.Fatal(err)
	}
	cases := []struct {
		name  string
		value string
	}{
		{name: "clean-url", value: "https://api.example.com/v1/orders/123456?region=us-east-1"},
		{name: "otel-json", value: `{"service.name":"checkout-api","deployment.environment":"production","http.request.method":"POST","server.address":"api.example.com"}`},
		{name: "kubernetes", value: "k8s.pod.name=checkout-api-7d9f68d4bb-x2klp k8s.namespace.name=production cloud.region=us-east-1"},
		{name: "natural-language", value: "payment authorization failed because the upstream gateway closed the connection before sending a response"},
		{name: "numeric", value: "status=200 duration_ms=17.384 content_length=65536 retry_count=0"},
		{name: "uuid-path", value: "/api/v1/accounts/550e8400-e29b-41d4-a716-446655440000/orders/8f14e45f-ea5e-4d2f-a8f3-c98bc0d81a42"},
		{name: "hex-base64", value: "4bf92f3577b34da6a3ce929d0e0e473600f067aa0ba902b7 SGVsbG8sIFdvcmxkIQ=="},
		{name: "long-clean", value: strings.Repeat("z", 1083)},
		{name: "long-mixed", value: strings.Repeat("Ab9_-zY/", 136)},
		{name: "catalog-match", value: findCatalogRuleWitness(b, catalog, 0)},
	}
	for _, test := range cases {
		b.Run(test.name, func(b *testing.B) {
			for _, strategy := range []struct {
				name  string
				match func(string, *ruleSet)
			}{
				{name: "aho-corasick", match: reference.match},
				{name: "rare-pair", match: catalog.pairMatcher.match},
			} {
				b.Run(strategy.name, func(b *testing.B) {
					var candidates ruleSet
					b.ReportAllocs()
					b.SetBytes(int64(len(test.value)))
					for range b.N {
						candidates = ruleSet{}
						strategy.match(test.value, &candidates)
					}
					benchmarkCandidateRules = candidates
				})
			}
		})
	}
}

func BenchmarkCatalogCandidateStorage(b *testing.B) {
	catalog, err := testPolicyCompiler.catalog()
	if err != nil {
		b.Fatal(err)
	}
	reference, err := newKeywordMatcher(catalogMatcherKeywords())
	if err != nil {
		b.Fatal(err)
	}
	ahoBytes := unsafe.Sizeof(reference) + uintptr(cap(reference.states))*unsafe.Sizeof(matcherState{})
	for _, state := range reference.states {
		ahoBytes += uintptr(cap(state.outputs)) * unsafe.Sizeof(uint16(0))
	}
	ahoBytes += uintptr(cap(reference.unicodeKeywords)) * unsafe.Sizeof(unicodeKeyword{})

	pairBytes := unsafe.Sizeof(*catalog.pairMatcher) + unsafe.Sizeof(asciiFoldTable)
	if catalog.pairMatcher.entryByPair != nil {
		pairBytes += unsafe.Sizeof(*catalog.pairMatcher.entryByPair)
	}
	pairBytes += uintptr(cap(catalog.pairMatcher.compactPairs)) * unsafe.Sizeof(catalogPairEntry{})
	pairBytes += uintptr(cap(catalog.pairMatcher.singleCertificates)) * unsafe.Sizeof(catalogPairCertificate{})
	pairBytes += uintptr(cap(catalog.pairMatcher.collisionGroups)) * unsafe.Sizeof([]catalogPairCertificate{})
	pairBytes += uintptr(cap(catalog.pairMatcher.collisionGuards)) * unsafe.Sizeof(catalogCollisionGuard{})
	for _, group := range catalog.pairMatcher.collisionGroups {
		pairBytes += uintptr(cap(group)) * unsafe.Sizeof(catalogPairCertificate{})
	}
	pairBytes += uintptr(cap(catalog.pairMatcher.duplicateOutputs)) * unsafe.Sizeof(uint16(0))
	if catalog.pairMatcher.singleOutputs != nil {
		pairBytes += unsafe.Sizeof(*catalog.pairMatcher.singleOutputs)
		for _, outputs := range catalog.pairMatcher.singleOutputs {
			pairBytes += uintptr(cap(outputs)) * unsafe.Sizeof(uint16(0))
		}
	}
	pairBytes += uintptr(cap(catalog.pairMatcher.unicodeKeyword)) * unsafe.Sizeof(unicodeKeyword{})

	b.ReportMetric(float64(ahoBytes), "aho-bytes")
	b.ReportMetric(float64(pairBytes), "rare-pair-bytes")
	b.ReportMetric(float64(catalog.pairMatcher.singleCertificateCount), "singleton-pairs")
	b.ReportMetric(float64(len(catalog.pairMatcher.collisionGroups)), "collision-pairs")
	for range b.N {
		benchmarkCandidateRules[0] = uint64(ahoBytes + pairBytes)
	}
}
