package secrets

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeywordHitOrderingPreservesRuleAndPosition(t *testing.T) {
	rng := rand.New(rand.NewSource(29))
	for _, count := range []int{0, 1, 31, 32, maxKeywordHits} {
		var buffer keywordHitBuffer
		for i := range count {
			buffer[i] = keywordHit{
				rule:  uint16(rng.Intn(catalogRuleWords * 64)),
				start: uint32(rng.Intn(128)),
				end:   uint32(128 + rng.Intn(128)),
			}
		}
		want := append([]keywordHit(nil), buffer[:count]...)
		slices.SortFunc(want, compareKeywordHits)
		hits := keywordHits{buffer: &buffer, count: count}
		hits.sort()
		require.Equal(t, want, append([]keywordHit(nil), buffer[:count]...))
	}
}

func TestMinRuneWidth(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		want       int
	}{
		{
			name:       "repeated groups",
			expression: `\d{15,16}(\||%)[0-9a-z\-_]{27,40}`,
			want:       43,
		},
		{
			name:       "fixed character class",
			expression: `[a-fA-F0-9]{40}`,
			want:       40,
		},
		{
			name:       "optional prefix",
			expression: `(?:https?://)?hooks.slack.com/x`,
			want:       17,
		},
		{
			name:       "alternation",
			expression: `a|bcd`,
			want:       1,
		},
		{
			name:       "case-folded literal",
			expression: `(?i)xoxe.xox[bp]-\d-[A-Z0-9]{163,166}`,
			want:       175,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			re, err := syntax.Parse(test.expression, syntax.Perl)
			require.NoError(t, err)
			assert.Equal(t, test.want, minRuneWidth(re))
		})
	}
}

func TestCatalogEvaluationPreservesFixtureContexts(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	fixtures := nativeCatalogFixtures(t)
	for index, spec := range nativeRuleSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			witness := findCatalogRuleWitness(t, catalog, index)
			for _, value := range []string{
				witness,
				" " + witness + "\n",
				"\x00" + witness + "\xff",
				witness + "\n" + witness,
				strings.Repeat("~", 8192) + witness,
				fixtures[spec.ID].Negative[0] + "\n" + witness,
			} {
				assertCatalogRuleMatchesFullScan(t, catalog, index, value)
			}
		})
	}
}

// catalogFindings evaluates every catalog rule against value, either with keyword-guided
// evaluation or with the reference whole-value scan.
func catalogFindings(catalog *compiledCatalog, value string, guided bool) []Match {
	return selectedCatalogFindings(catalog, value, guided, catalog.all, allRuleMatches)
}

// Keep the complete matcher, global hit budget, and original rule indices even
// when checking one rule. Generated per-rule probes must not rescan every
// unrelated regexp quadratically as the catalog grows.
func selectedCatalogFindings(catalog *compiledCatalog, value string, guided bool, selected ruleSet, mode evaluationMode) []Match {
	if !guided {
		return evaluateRuleSet(catalog.rules, selected, value, nil, mode, nil)
	}
	var hits keywordHits
	defer hits.release()
	hits.reset(&catalog.guided, &catalog.boundary)
	var candidates ruleSet
	catalog.pairMatcher.matchHits(value, &candidates, &hits)
	hits.sort()
	candidates.merge(catalog.keywordless)
	return evaluateRuleSet(catalog.rules, candidates.intersect(selected), value, &hits, mode, nil)
}

func assertCatalogRuleMatchesFullScan(t testing.TB, catalog *compiledCatalog, index int, value string) []Match {
	t.Helper()
	var selected ruleSet
	selected.add(uint16(index))
	reference := selectedCatalogFindings(catalog, value, false, selected, allRuleMatches)
	guided := selectedCatalogFindings(catalog, value, true, selected, allRuleMatches)
	require.Equal(t, reference, guided, "per-rule multiplicity for %q", value)
	unique := selectedCatalogFindings(catalog, value, true, selected, uniqueRuleMatches)
	require.Equal(t, uniqueFindingVerdict(reference).Matches, unique, "per-rule existence projection for %q", value)
	return reference
}

func assertKeywordGuidedMatchesFullScan(t testing.TB, catalog *compiledCatalog, value string) []Match {
	reference := catalogFindings(catalog, value, false)
	guided := catalogFindings(catalog, value, true)
	if !assert.Equal(t, reference, guided, "value %q", value) {
		t.FailNow()
	}
	// Keep the raw multiplicity comparison above and independently check the
	// production consumer's ordered unique-ID projection, including existence mode.
	policy := CompiledPolicy{catalog: catalog}
	if !assert.Equal(t, uniqueFindingVerdict(reference), policy.Detect(value), "verdict for %q", value) {
		t.FailNow()
	}
	return reference
}

// generateASCII writes a random ASCII string that follows the structure of the expression.
// Zero-width assertions are ignored, so the result is a probe rather than a guaranteed match.
func generateASCII(builder *strings.Builder, re *syntax.Regexp, rng *rand.Rand) {
	switch re.Op {
	case syntax.OpLiteral:
		for _, r := range re.Rune {
			if r >= 0x80 {
				continue
			}
			value := byte(r)
			if re.Flags&syntax.FoldCase != 0 && rng.Intn(2) == 0 {
				switch {
				case value >= 'a' && value <= 'z':
					value -= 'a' - 'A'
				case value >= 'A' && value <= 'Z':
					value += 'a' - 'A'
				}
			}
			builder.WriteByte(value)
		}
	case syntax.OpCharClass:
		var members []byte
		for i := 0; i+1 < len(re.Rune); i += 2 {
			for r := re.Rune[i]; r <= re.Rune[i+1] && r < 0x80; r++ {
				members = append(members, byte(r))
			}
		}
		if len(members) == 0 {
			builder.WriteByte('x')
			return
		}
		builder.WriteByte(members[rng.Intn(len(members))])
	case syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		builder.WriteByte(byte(' ' + rng.Intn(95)))
	case syntax.OpCapture:
		generateASCII(builder, re.Sub[0], rng)
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			generateASCII(builder, sub, rng)
		}
	case syntax.OpAlternate:
		generateASCII(builder, re.Sub[rng.Intn(len(re.Sub))], rng)
	case syntax.OpQuest:
		if rng.Intn(2) == 0 {
			generateASCII(builder, re.Sub[0], rng)
		}
	case syntax.OpStar, syntax.OpPlus:
		count := rng.Intn(4)
		if re.Op == syntax.OpPlus {
			count++
		}
		for range count {
			generateASCII(builder, re.Sub[0], rng)
		}
	case syntax.OpRepeat:
		count := re.Min
		switch {
		case re.Max < 0:
			count += rng.Intn(4)
		case re.Max > re.Min && rng.Intn(8) == 0:
			count = re.Max
		case re.Max > re.Min:
			count += rng.Intn(min(re.Max-re.Min, 3) + 1)
		}
		for range count {
			generateASCII(builder, re.Sub[0], rng)
		}
	}
}

func generateProbe(re *syntax.Regexp, rng *rand.Rand) string {
	var builder strings.Builder
	generateASCII(&builder, re, rng)
	return builder.String()
}

func catalogWitnessSeed(id string) int64 {
	const (
		offset = uint64(1469598103934665603)
		prime  = uint64(1099511628211)
	)
	hash := offset
	for i := range len(id) {
		hash = (hash ^ uint64(id[i])) * prime
	}
	return int64(hash)
}

func hasRuleFinding(findings []Match, id string) bool {
	for _, finding := range findings {
		if finding.RuleID == id {
			return true
		}
	}
	return false
}

func findCatalogRuleWitness(t testing.TB, catalog *compiledCatalog, index int) string {
	t.Helper()
	fixture, ok := nativeCatalogFixtures(t)[catalog.rules[index].id]
	require.True(t, ok, "missing independent fixture for %q", catalog.rules[index].id)
	require.NotEmpty(t, fixture.Positive)
	return fixture.Positive[0]
}

func closestCatalogRuleNegative(t testing.TB, rule *compiledRule, witness string) string {
	t.Helper()
	for removed := 1; removed <= len(witness); removed++ {
		for _, candidate := range []string{witness[:len(witness)-removed], witness[removed:]} {
			if !hasRuleFinding(rule.detect(candidate, nil), rule.id) {
				return candidate
			}
		}
	}
	t.Fatalf("failed to derive negative witness for catalog rule %q", rule.id)
	return ""
}

func TestEveryCatalogRuleHasPositiveWitness(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	require.Len(t, catalog.rules, len(nativeRuleSpecs))
	fixtures := nativeCatalogFixtures(t)

	for index, spec := range nativeRuleSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			for fixtureIndex, witness := range fixtures[spec.ID].Positive {
				t.Run(fmt.Sprintf("positive-%d", fixtureIndex), func(t *testing.T) {
					findings := assertCatalogRuleMatchesFullScan(t, catalog, index, witness)
					require.True(t, hasRuleFinding(findings, spec.ID), "independent witness did not detect its catalog rule")
				})
			}
		})
	}
}

func TestCatalogRuleNearMissesMatchFullScan(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)

	for index, spec := range nativeRuleSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			t.Parallel()
			witness := findCatalogRuleWitness(t, catalog, index)
			nearMiss := closestCatalogRuleNegative(t, &catalog.rules[index], witness)
			require.False(t, hasRuleFinding(assertCatalogRuleMatchesFullScan(t, catalog, index, nearMiss), spec.ID))

			for _, value := range []string{
				nearMiss,
				"x" + nearMiss + "x",
				nearMiss + "\n" + nearMiss,
				"\x00" + nearMiss + "\xff",
				strings.ToUpper(nearMiss),
			} {
				assertCatalogRuleMatchesFullScan(t, catalog, index, value)
			}
		})
	}
}

func TestKeywordGuidedEvaluationMatchesFullScan(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	rng := rand.New(rand.NewSource(1))
	contexts := []string{"", " ", "\n", "\"", "'", "=", "x", "_", "-", ".", "key ", "token=", "ey", "curl ", "api-", "s.", "\t", "ab_", "\\n", ";", "https://", strings.Repeat("a", 60)}
	pick := func() string { return contexts[rng.Intn(len(contexts))] }

	t.Run("rules", func(t *testing.T) {
		for index, spec := range nativeRuleSpecs {
			parsed, err := syntax.Parse(spec.Regex, syntax.Perl)
			require.NoError(t, err, spec.ID)
			values := []string{findCatalogRuleWitness(t, catalog, index)}
			for range 40 {
				probe := generateProbe(parsed, rng)
				values = append(
					values,
					probe,
					pick()+probe+pick(),
					probe+pick()+probe,
					probe[:len(probe)/2]+pick()+probe,
					probe+"\n"+pick()+probe+"\n",
					strings.ToUpper(probe),
				)
			}
			for _, keyword := range spec.Keywords {
				values = append(
					values,
					keyword,
					"x"+keyword+"x",
					" "+keyword+"=abc",
					strings.Repeat("y", 200)+keyword+strings.Repeat("z", 300),
					strings.ToUpper(keyword)+"\n"+keyword+keyword,
				)
			}
			const valueChunkSize = 32
			for start := 0; start < len(values); start += valueChunkSize {
				chunk := values[start:min(start+valueChunkSize, len(values))]
				t.Run(spec.ID, func(t *testing.T) {
					t.Parallel()
					for _, value := range chunk {
						assertCatalogRuleMatchesFullScan(t, catalog, index, value)
					}
				})
			}
		}
	})
}

func TestKeywordGuidedEvaluationEdgeCases(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" + "." + "eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0" + "." + "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	aws := "AKIA" + "QWERTYUIOPASDFGH"
	generic := "secret_key = \"" + strings.Repeat("Zq9", 12) + "\""
	values := []string{
		jwt, "Bearer " + jwt, "x" + jwt, "key" + jwt, jwt + jwt, jwt + " " + jwt, "\n" + jwt + "\n",
		aws, "prefix " + aws + " suffix", aws + "\n" + aws, "AKIA" + aws, strings.Repeat("AKIA", 20),
		generic, strings.Repeat("x", 49) + generic, strings.Repeat("x", 50) + generic, strings.Repeat("x", 51) + generic,
		"passwd=" + strings.Repeat("Ab1", 8) + " " + generic, generic + "\n" + generic,
		"glsa_" + strings.Repeat("A1b2", 8) + "_0a1b2c3d", "xglsa_" + strings.Repeat("A1b2", 8) + "_0a1b2c3d",
		"hvs." + strings.Repeat("Zq9-", 24) + " s." + strings.Repeat("a1", 12) + " status.ok bytes.count",
		"ATATT3" + strings.Repeat("xY", 93) + " confluence=1 atlassian_token=" + strings.Repeat("ab12", 6),
		"the monkey took the key and they left: keyey eyey ey ey",
		strings.Repeat("ey", 100),
		// Windows whose keywords are ordinary text and that lack the rule's other factors.
		"is_public=true test_mode=false live_mode=false",
		strings.Repeat("is_public=true test_mode=false live_mode=false region=us-east-1 ", 14),
		"lob_api_key = test_" + strings.Repeat("a1b2c3d", 5) + " is_public=true",
		"LOB key: live_" + strings.Repeat("0f", 17) + "z live_pub_" + strings.Repeat("9c", 15) + "e",
		"nrii-" + strings.Repeat("aZ", 16) + " New_Relic insert key nrii-" + strings.Repeat("aZ", 16) + " sk-" + strings.Repeat("T3BlbkFJ", 4),
	}
	for _, value := range values {
		assertKeywordGuidedMatchesFullScan(t, catalog, value)
	}
}

// recordHits runs candidate selection and records positions only for guided rules.
func recordHits(t *testing.T, catalog *compiledCatalog, value string) (*keywordHits, ruleSet) {
	hits := &keywordHits{}
	t.Cleanup(hits.release)
	hits.reset(&catalog.guided, &catalog.boundary)
	var candidates ruleSet
	catalog.pairMatcher.matchHits(value, &candidates, hits)
	hits.sort()
	return hits, candidates
}

func findingRuleIDs(findings []Match) []string {
	ids := make([]string, 0, len(findings))
	for _, finding := range findings {
		ids = append(ids, finding.RuleID)
	}
	return ids
}

func TestKeywordHitsOverflowIsPerRule(t *testing.T) {
	catalog := compileTestCatalog(
		t,
		catalogRuleSpec{ID: "flood", Regex: `\bkey:[A-Z]{4}\b`, Keywords: []string{"key"}},
		catalogRuleSpec{ID: "token", Regex: `\btoken:[A-Z]{4}\b`, Keywords: []string{"token"}},
	)
	floodRule := catalog.byID["flood"]
	tokenRule := catalog.byID["token"]
	flood := strings.Repeat("key ", maxKeywordHits+50)
	token := "token:ABCD"

	// The token arrives after the buffer filled: only overflowing rules lose positions.
	hits, _ := recordHits(t, catalog, flood+token)
	assert.Equal(t, maxKeywordHits, hits.count)
	assert.False(t, hits.invalid)
	assert.False(t, hits.guides(floodRule))
	assert.False(t, hits.guides(tokenRule))
	assert.Equal(t, []string{"token"}, findingRuleIDs(assertKeywordGuidedMatchesFullScan(t, catalog, flood+token)))

	// The token arrives first: its single occurrence is complete and stays guided.
	hits, _ = recordHits(t, catalog, token+" "+flood)
	assert.False(t, hits.guides(floodRule))
	assert.True(t, hits.guides(tokenRule))
	from, to := hits.rangeFor(tokenRule)
	assert.Equal(t, 1, to-from)
	assert.Equal(t, []string{"token"}, findingRuleIDs(assertKeywordGuidedMatchesFullScan(t, catalog, token+" "+flood)))
}

func TestKeywordOverflowPreservesOrderedCaptures(t *testing.T) {
	type observation struct {
		start, end int
		capture    [sha256.Size]byte
	}
	for _, test := range []struct {
		name       string
		expression string
		keywords   []string
		flood      string
		value      string
	}{
		{
			name:       "anchored-later-accepted",
			expression: `\b(?i:tok):([A-Z]{4})(?:$|[^A-Z])`,
			keywords:   []string{"tok"},
			flood:      "tok? ",
			value:      "tok:DROP tok:KEEP tok:KEEP",
		},
		{
			name:       "optional-prefix",
			expression: `[a-z]{0,3}(?i:tok):([A-Z]{4})(?:$|[^A-Z])`,
			keywords:   []string{"tok"},
			flood:      "tok? ",
			value:      "abctok:DROP abctok:KEEP tok:KEEP",
		},
		{
			name:       "overlapping-keywords-and-rejected-match",
			expression: `(?i:tok(?:en)?):([A-Z]{4})(?:;tok:KEEP)?`,
			keywords:   []string{"tok", "token", "ok"},
			flood:      "tok? ",
			value:      "token:DROP;tok:KEEP tok:KEEP",
		},
		{
			name:       "anchored-real-eof",
			expression: `\b(?i:tok):([A-Z]{4})$`,
			keywords:   []string{"tok"},
			flood:      "tok? ",
			value:      "tok:KEEP!" + strings.Repeat("~", 40) + "tok:KEEP",
		},
		{
			name:       "anchored-sliced-later-match",
			expression: `\b(?i:tok):([A-Z]{1,4})$`,
			keywords:   []string{"tok"},
			flood:      "tok ",
			value:      "tok tok:KEEP",
		},
		{
			name:       "unicode-and-malformed-context",
			expression: `\bTOK:([A-Z]{4})(?:$|[^A-Z])`,
			keywords:   []string{"TOK"},
			flood:      "TOK? ",
			value:      "\xffTOK:DROP界TOK:KEEP\xffTOK:KEEP",
		},
		{
			name:       "fold-alias-fallback",
			expression: `\b(?i:key|set):([A-Z]{4})(?:$|[^A-Z])`,
			keywords:   []string{"key", "set"},
			flood:      "key? ",
			value:      "Key:DROP ſet:KEEP key:KEEP",
		},
		{
			name:       "windows-different-keyword-end-order",
			expression: `\b[A-Z]{1,3}=(?i:taglong|ag):([A-Z]{4})(?:$|[^A-Z])`,
			keywords:   []string{"taglong", "ag"},
			flood:      "taglong? ",
			value:      "A=taglong:DROP~~B=ag:KEEP" + strings.Repeat("~", 70) + "C=taglong:KEEP",
		},
		{
			name:       "touching-windows",
			expression: `\b[A-Z]{1,2}=tag:([A-Z]{4})(?:$|[^A-Z])`,
			keywords:   []string{"tag"},
			flood:      "tag? ",
			// The two keyword starts are 23 bytes apart: with width 12,
			// the second window begins exactly at the first window's end.
			value: "A=tag:DROP" + strings.Repeat("~", 13) + "B=tag:KEEP",
		},
		{
			name:       "window-line-context",
			expression: `(?m)^[A-Z]{1,3}=(?i:tag|ag):([A-Z]{4})$`,
			keywords:   []string{"tag", "ag"},
			flood:      "tag?\n",
			value:      "\nA=tag:DROP\nB=ag:KEEP\n",
		},
		{
			name:       "window-real-eof",
			expression: `\b[A-Z]{1,3}=(?i:tag|ag):([A-Z]{4})$`,
			keywords:   []string{"tag", "ag"},
			flood:      "tag? ",
			value:      "A=tag:KEEP!" + strings.Repeat("~", 70) + "B=ag:KEEP",
		},
		{
			name:       "window-rune-edges",
			expression: `\b[A-Z]{1,3}=(?i:tag|ag):([A-Z]{4})[界é\x{fffd}]`,
			keywords:   []string{"tag", "ag"},
			flood:      "tag? ",
			value:      "A=tag:DROP界" + strings.Repeat("é", 37) + "\xffB=ag:KEEPé",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			oracle := regexp.MustCompile(test.expression)
			var seen []observation
			var originalValue string
			catalog := compileTestCatalog(
				t,
				catalogRuleSpec{ID: "existence", Regex: test.expression, Keywords: test.keywords, SecretGroup: 1},
				catalogRuleSpec{ID: "capture", Regex: test.expression, Keywords: test.keywords, SecretGroup: 1, Validate: func(secret string) bool {
					return secret == "KEEP"
				}},
				catalogRuleSpec{ID: "context", Regex: test.expression, Keywords: test.keywords, SecretGroup: 1, ValidateContext: func(original string, start, end int, secret string) contextValidation {
					require.True(t, original == originalValue, "context validator must receive the complete original value")
					seen = append(seen, observation{start, end, sha256.Sum256([]byte(secret))})
					return contextValidation{accepted: secret == "KEEP"}
				}},
			)
			for _, placement := range []string{"sparse", "prefix-overflow", "suffix-overflow"} {
				overflow := placement != "sparse"
				value := test.value
				switch placement {
				case "prefix-overflow":
					value = strings.Repeat(test.flood, maxKeywordHits+20) + value
				case "suffix-overflow":
					value += strings.Repeat(test.flood, maxKeywordHits+20)
				}
				originalValue = value
				hits, _ := recordHits(t, catalog, value)
				for index := range catalog.rules {
					require.Equal(t, overflow, hits.overflowed.has(uint16(index)), "probe overflow state")
				}
				indices := oracle.FindAllStringSubmatchIndex(value, -1)
				for _, mode := range []evaluationMode{allRuleMatches, uniqueRuleMatches} {
					var want []Match
					var wantSeen []observation
					for _, id := range []string{"existence", "capture", "context"} {
						for _, match := range indices {
							secret := value[match[2]:match[3]]
							if id == "context" {
								wantSeen = append(wantSeen, observation{match[0], match[1], sha256.Sum256([]byte(secret))})
							}
							if id == "existence" || secret == "KEEP" {
								want = append(want, Match{RuleID: id})
								if mode == uniqueRuleMatches {
									break
								}
							}
						}
					}
					seen = nil
					got := selectedCatalogFindings(catalog, value, true, catalog.all, mode)
					require.Equal(t, want, got, "ordered accepted matches; placement=%s mode=%d", placement, mode)
					require.Equal(t, wantSeen, seen, "original offsets and capture digests; placement=%s mode=%d", placement, mode)
				}
			}
		})
	}
}

func TestFullScanKeywordsDoNotExhaustGuidedHits(t *testing.T) {
	catalog := compileTestCatalog(
		t,
		catalogRuleSpec{ID: "full", Regex: `full:[A-Z]{4}`, Keywords: []string{"full"}},
		catalogRuleSpec{ID: "guided", Regex: `\bkey:[A-Z]{4}\b`, Keywords: []string{"key"}},
	)
	value := "full:ABCD " + strings.Repeat("full:~~~ ", maxKeywordHits+20) + "key:EFGH"
	hits, candidates := recordHits(t, catalog, value)
	require.True(t, candidates.has(catalog.byID["full"]))
	require.True(t, hits.guides(catalog.byID["guided"]))
	require.Equal(t, []Match{{RuleID: "full"}, {RuleID: "guided"}},
		assertKeywordGuidedMatchesFullScan(t, catalog, value))
}

func TestKeywordHitsDropKeywordsGluedToWords(t *testing.T) {
	catalog := compileTestCatalog(
		t,
		catalogRuleSpec{ID: "boundary", Regex: `\bey:[A-Z]{4}\b`, Keywords: []string{"ey"}},
	)
	boundaryRule := catalog.byID["boundary"]

	// "ey" occurs six times but only three occurrences follow a non-word byte.
	value := "the monkey and they said: ey eyJ.ey key"
	hits, candidates := recordHits(t, catalog, value)
	assert.True(t, candidates.has(boundaryRule))
	assert.True(t, hits.guides(boundaryRule))
	from, to := hits.rangeFor(boundaryRule)
	assert.Equal(t, 3, to-from)
	for i := from; i < to; i++ {
		assert.True(t, asciiWordBoundary(value, int(hits.buffer[i].start)))
	}
	assert.Empty(t, assertKeywordGuidedMatchesFullScan(t, catalog, value))

	// Every occurrence glued to a word character leaves the rule with a complete, empty range.
	hits, candidates = recordHits(t, catalog, "monkey turkey")
	assert.True(t, candidates.has(boundaryRule))
	assert.True(t, hits.guides(boundaryRule))
	from, to = hits.rangeFor(boundaryRule)
	assert.Equal(t, from, to)
	assert.Empty(t, assertKeywordGuidedMatchesFullScan(t, catalog, "monkey turkey"))
}

func FuzzKeywordGuidedEvaluationMatchesFullScan(f *testing.F) {
	catalog, err := testPolicyCompiler.catalog()
	if err != nil {
		f.Fatal(err)
	}
	seen := make(map[string]bool)
	addSeed := func(value string) {
		if !seen[value] {
			seen[value] = true
			f.Add(value)
		}
	}
	for _, seed := range []string{
		"clean",
		"AKIA" + "QWERTYUIOPASDFGH",
		"xoxb-" + "1234567890-1234567890123-abcdefghijklmnopqrstuvwx",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" + "." + "eyJzdWIiOiIxMjM0NTY3ODkwIn0" + "." + "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		"api_key = \"" + strings.Repeat("Zq9", 12) + "\"",
		"glsa_" + strings.Repeat("A1b2", 8) + "_0a1b2c3d",
		"prefix Σ suffix",
	} {
		addSeed(seed)
	}
	// Keep mutations near every catalog expression, not only the handful of
	// common token formats above. Deterministic tests don't replace fuzz seeds:
	// only seeded or discovered inputs become starting points for new mutations.
	rng := rand.New(rand.NewSource(2))
	for index, spec := range nativeRuleSpecs {
		addSeed(findCatalogRuleWitness(f, catalog, index))
		parsed, err := syntax.Parse(spec.Regex, syntax.Perl)
		require.NoError(f, err, spec.ID)
		addSeed(" " + generateProbe(parsed, rng) + "\n")
		for _, keyword := range spec.Keywords {
			addSeed("x" + strings.ToUpper(keyword) + "=" + keyword)
		}
		for _, keyword := range effectiveKeywords(spec) {
			addSeed(keyword + "\n" + strings.ToUpper(keyword))
		}
	}
	f.Fuzz(func(t *testing.T, value string) {
		t.Parallel()
		assertKeywordGuidedMatchesFullScan(t, catalog, value)
	})
}

func BenchmarkDetectKeywordCollisions(b *testing.B) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{})
	if err != nil {
		b.Fatal(err)
	}
	sql := strings.Repeat("SELECT id, key, value FROM settings WHERE key = ? AND tenant = ? ORDER BY key; ", 13)
	cases := []struct {
		name   string
		policy *CompiledPolicy
		value  string
	}{
		{name: "clean-url", policy: policy, value: "https://api.example.com/v1/orders/123456?region=us-east-1"},
		{name: "sql-1k-key", policy: policy, value: sql},
		{name: "user-agent-curl", policy: policy, value: "curl/7.68.0 (x86_64-pc-linux-gnu) libcurl/7.68.0 OpenSSL/1.1.1f zlib/1.2.11 " + strings.Repeat("token ", 40)},
		{name: "pod-name-api", policy: policy, value: "k8s.pod.name=checkout-api-7d9f68d4bb-x2klp k8s.namespace.name=production cloud.region=asia-east1"},
		{name: "json", policy: policy, value: `{"service.name":"checkout-api","auth.mode":"token","http.request.method":"POST","server.address":"api.example.com","secret_ref":"vault://kv/data/keys/checkout"}`},
		{name: "sql-1k", policy: policy, value: sql},
		{name: "catalog-match", policy: policy, value: findCatalogRuleWitness(b, policy.catalog, 0)},
		{name: "decimal-64", policy: policy, value: strings.Repeat("1234567890", 6) + "1234"},
		{name: "hex-2k", policy: policy, value: strings.Repeat("abc012345def6789", 128)},
		{name: "near-runs-2k", policy: policy, value: strings.Repeat("01234567890123~", 140)},
		{name: "repeated-token-prefix-64", policy: policy, value: strings.Repeat("ghp_~~~ ", 8)},
		{name: "repeated-token-prefix-2k", policy: policy, value: strings.Repeat("ghp_~~~ ", 256)},
		{name: "repeated-token-prefix-128k", policy: policy, value: strings.Repeat("ghp_~~~ ", 16384)},
		{name: "mixed-keyword-pressure", policy: policy, value: strings.Repeat("ghp_~~~ ", 256) + findCatalogRuleWitness(b, policy.catalog, 0)},
	}
	for _, test := range cases {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(test.value)))
			for range b.N {
				benchmarkVerdict = test.policy.Detect(test.value)
			}
		})
	}
}

var benchmarkVerdict Verdict
