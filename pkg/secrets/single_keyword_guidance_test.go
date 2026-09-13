package secrets

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingleKeywordMatcherPreservesOriginalPositions(t *testing.T) {
	keywords := []matcherKeyword{
		{value: "a", rule: 63},
		{value: "k", rule: 64},
		{value: "s", rule: 65},
		{value: "key", rule: 511},
		{value: "k", rule: 971},
		{value: "token", rule: 1023},
	}
	var enabled, boundary ruleSet
	for _, index := range []uint16{63, 64, 65, 511, 1023} {
		enabled.add(index)
	}
	boundary.add(63)

	probes := []struct {
		name     string
		value    string
		hits     []keywordHit
		nonASCII bool
		alias    bool
	}{
		{name: "empty"},
		{name: "one byte has no pair", value: "A", hits: []keywordHit{{rule: 63, start: 0, end: 1}}},
		{
			name:  "ASCII boundary and disabled duplicate output",
			value: "xa A K KEY TOKEN S",
			hits: []keywordHit{
				{rule: 63, start: 3, end: 4},
				{rule: 64, start: 5, end: 6},
				{rule: 64, start: 7, end: 8},
				{rule: 64, start: 13, end: 14},
				{rule: 65, start: 17, end: 18},
				{rule: 511, start: 7, end: 10},
				{rule: 1023, start: 11, end: 16},
			},
		},
		{
			name:  "ordinary Unicode uses original byte offsets",
			value: "éA πK TOKEN",
			hits: []keywordHit{
				{rule: 63, start: 2, end: 3},
				{rule: 64, start: 6, end: 7},
				{rule: 64, start: 10, end: 11},
				{rule: 1023, start: 8, end: 13},
			},
			nonASCII: true,
		},
		{
			name:  "invalid UTF8 retains original byte offsets",
			value: "\xffA\xfeK TOKEN",
			hits: []keywordHit{
				{rule: 63, start: 1, end: 2},
				{rule: 64, start: 3, end: 4},
				{rule: 64, start: 7, end: 8},
				{rule: 1023, start: 5, end: 10},
			},
			nonASCII: true,
		},
		{
			name:  "fold aliases select candidates without normalized positions",
			value: "Key ſ TOKEN K",
			hits: []keywordHit{
				{rule: 64, start: 11, end: 12},
				{rule: 64, start: 15, end: 16},
				{rule: 1023, start: 9, end: 14},
			},
			nonASCII: true,
			alias:    true,
		},
		{name: "high-bit aliases are not ASCII", value: "\xe1\xeb\xf3", nonASCII: true},
		{name: "Unicode keyword fold class", value: "σ Σ ς", nonASCII: true},
	}

	for _, unicodeInventory := range []bool{false, true} {
		inventory := keywords
		name := "ASCII inventory"
		if unicodeInventory {
			inventory = append(slices.Clone(keywords), matcherKeyword{value: "σ", rule: 512})
			name = "mixed Unicode inventory"
		}
		t.Run(name, func(t *testing.T) {
			reference, err := newKeywordMatcher(inventory)
			require.NoError(t, err)
			matcher, err := newCatalogPairMatcher(inventory)
			require.NoError(t, err)
			for _, probe := range probes {
				t.Run(probe.name, func(t *testing.T) {
					var expected, actual ruleSet
					var hits keywordHits
					defer hits.release()
					hits.reset(&enabled, &boundary)
					reference.match(probe.value, &expected)
					matcher.matchHits(probe.value, &actual, &hits)
					require.Equal(t, expected, actual)
					if unicodeInventory && probe.value != "" {
						require.True(t, hits.invalid, "Unicode normalization cannot supply original offsets")
						require.False(t, hits.guides(1023))
						return
					}
					require.False(t, hits.invalid)
					require.Equal(t, probe.nonASCII, hits.nonASCII)
					require.Equal(t, probe.alias, hits.asciiFoldAlias)
					hits.sort()
					var positions []keywordHit
					if hits.count != 0 {
						positions = append(positions, hits.buffer[:hits.count]...)
					}
					require.Equal(t, probe.hits, positions)
				})
			}
		})
	}
}

func TestSingleKeywordGuidancePreservesCatalogEvaluation(t *testing.T) {
	// Retain every native rule, its global index, and the shared hit budget. The
	// added rules exercise single-byte starts, captures, context, and fold aliases
	// without depending on a provider's changing credential syntax.
	specs := append(
		slices.Clone(nativeRuleSpecs),
		catalogRuleSpec{
			ID: "single-guidance-capture", Regex: `\ba:([A-Z0-9]{4})\b`, Keywords: []string{"a"}, SecretGroup: 1, Entropy: 1.5,
			Validate: func(secret string) bool { return secret != "QW12" },
		},
		catalogRuleSpec{ID: "single-guidance-fold", Regex: `(?i)k:([a-z]{4})`, Keywords: []string{"k"}, SecretGroup: 1},
		catalogRuleSpec{ID: "single-guidance-window", Regex: `\b[0-9]{2}=s:([A-Z0-9]{4})\b`, Keywords: []string{"s"}, SecretGroup: 1},
		catalogRuleSpec{
			ID: "single-guidance-context", Regex: `\btoken:([A-Z0-9]{4})\b`, Keywords: []string{"token"}, SecretGroup: 1,
			ValidateContext: func(value string, start, end int, secret string) contextValidation {
				return contextValidation{accepted: strings.HasPrefix(value, "allow ") && value[start:end] == "token:"+secret}
			},
		},
	)
	catalog, err := compileCatalog(specs)
	require.NoError(t, err)
	for _, value := range []string{
		"allow a:AAAA a:QW12 a:AB12 token:CD34 12=s:EF56 k:abcd",
		"allow éa:AB12 πtoken:CD34 12=s:EF56 λk:abcd",
		"allow K:abcd token:CD34 ſ a:AB12",
		"allow \xffa:AB12 \xfetoken:CD34 k:abcd",
		"allow xa:AB12 xtoken:CD34 12=s:EF56",
		"deny a:AB12 token:CD34 k:abcd",
	} {
		assertKeywordGuidedMatchesFullScan(t, catalog, value)
		hits, _ := recordHits(t, catalog, value)
		require.False(t, hits.invalid, "single-byte rules must not disable catalog-wide guidance")
		require.True(t, hits.guides(catalog.byID["single-guidance-context"]))
	}

	// Singleton flooding must fall back for incomplete rules, without losing
	// complete pair-keyword positions or later accepted singleton matches.
	value := "allow token:CD34 " + strings.Repeat("a ", maxKeywordHits+32) + "a:AB12 k:abcd"
	hits, _ := recordHits(t, catalog, value)
	require.False(t, hits.invalid)
	require.False(t, hits.guides(catalog.byID["single-guidance-capture"]))
	require.True(t, hits.guides(catalog.byID["single-guidance-context"]))
	assertKeywordGuidedMatchesFullScan(t, catalog, value)
}
