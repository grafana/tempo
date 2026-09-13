package secrets

import (
	"math/rand"
	"regexp"
	"regexp/syntax"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestASCIIRunSkipsPreserveOverlappingCandidates(t *testing.T) {
	var set asciiSet
	set.add('a')
	for width := 1; width <= 8; width++ {
		run := asciiRun{set: set, width: width}
		for n := 0; n <= 10; n++ {
			for mask := 0; mask < 1<<n; mask++ {
				var value strings.Builder
				for i := 0; i < n; i++ {
					if mask&(1<<i) != 0 {
						value.WriteByte('a')
					} else {
						value.WriteByte('!')
					}
				}
				s := value.String()
				require.Equal(t, strings.Contains(s, strings.Repeat("a", width)), run.exists(s), "width=%d value=%q", width, s)
			}
		}
	}
}

func TestRequiredASCIIRunIsNecessary(t *testing.T) {
	expressions := []string{
		`\b[0-9]{15,16}[|%][a-z_-]{27,40}`,
		`(?:sgp_)?[a-fA-F0-9]{40}`,
		`(a{8})?b{10}`,
		`(?:a{8}|[0-9]{10})`,
		`(?:a{8}|xyz)`,
		`(?:a{8})*`,
		`(?:a{8})+`,
		`(?i)k{8}`,
		`[^x]{8}`,
	}
	for _, expression := range expressions {
		t.Run(expression, func(t *testing.T) {
			parsed, err := syntax.Parse(expression, syntax.Perl)
			require.NoError(t, err)
			run := requiredASCIIRun(parsed)
			re := regexp.MustCompile(expression)
			rng := rand.New(rand.NewSource(1))
			for range 100 {
				probe := generateProbe(parsed, rng)
				for _, value := range []string{probe, "~" + probe + "~", strings.ToUpper(probe), probe + probe, "東京 " + probe + " \xff"} {
					if run.width > 0 && (!run.allowsUnicode || isASCIIKeyword(value)) && re.MatchString(value) {
						require.True(t, run.exists(value), "%q", value)
					}
				}
			}
		})
	}
}

func TestCatalogRequiredRunsPreserveWitnesses(t *testing.T) {
	for _, spec := range nativeRuleSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			parsed, err := syntax.Parse(spec.Regex, syntax.Perl)
			require.NoError(t, err)
			run := requiredASCIIRun(parsed)
			if run.width == 0 {
				return
			}
			for _, witness := range nativeCatalogFixtures(t)[spec.ID].Positive {
				if !run.allowsUnicode || isASCIIKeyword(witness) {
					require.True(t, run.exists(witness), "required run rejected independent positive")
				}
			}
			re := regexp.MustCompile(spec.Regex)
			rng := rand.New(rand.NewSource(catalogWitnessSeed(spec.ID)))
			for range 40 {
				probe := generateProbe(parsed, rng)
				for _, value := range []string{probe, "東京 " + probe + " \xff"} {
					if (!run.allowsUnicode || isASCIIKeyword(value)) && re.MatchString(value) {
						require.True(t, run.exists(value), "%q", value)
					}
				}
			}
		})
	}
}

func TestRequiredRunPreservesUnicodeFoldMatches(t *testing.T) {
	rule, err := compileRuleSpec(catalogRuleSpec{ID: "folded", Regex: `token_(?i)[a-z]{8}`}, nil)
	require.NoError(t, err)
	var candidates ruleSet
	candidates.add(0)
	for _, value := range []string{
		"token_" + strings.Repeat("K", 8),
		"token_" + strings.Repeat("K", 8),
		"token_" + strings.Repeat("ſ", 8),
		strings.Repeat("~", 64) + "token_" + strings.Repeat("K", 8),
	} {
		hits := keywordHits{invalid: !isASCIIKeyword(value)}
		want := []Match{{RuleID: "folded"}}
		require.Equal(t, want, evaluateRuleSet([]compiledRule{rule}, candidates, value, &hits, allRuleMatches, nil))
	}
}

func TestRequiredPunctuationDoesNotRejectMatches(t *testing.T) {
	for _, expression := range []string{
		`(?:https://example.test/)?[0-9]{5,20}:[A-Za-z0-9_-]{30,64}`,
		`(?:a:b|K)`, `(a:)?b`, `(?i:k)-[a-z]{8}`, `(?:x:y)+`,
		`(?:x:y){0,3}`, `(?:x:y){1,3}`, `[;λ]`, `\pL+[:.]xyz`,
		`https://[a-z]+:[a-z]+@[a-z]+\.[a-z]+`,
		`(?:a:b@|c:d#)`, `(?:a:b|c@d)`, `(?:x@y)*a:b`,
		`(?i:K)@`, `(?:x@y){0}`, `(?:x@y|)`,
		`key[:=][a-z]{4}`, `@(?:x:y|x=z)`, `@[:@=]`,
		`(?:a:b|word)`, `(?:a:b){0,2}`, `(?:a:b){1,2}`,
		`(?i:key)[:=](?:[a-z]{4}|ſ)`, `[^a]`, `[!-/]`, `[A!]`,
	} {
		t.Run(expression, func(t *testing.T) {
			rule, err := compileRuleSpec(catalogRuleSpec{ID: "punctuation", Regex: expression}, nil)
			require.NoError(t, err)
			parsed, err := syntax.Parse(expression, syntax.Perl)
			require.NoError(t, err)
			var candidates ruleSet
			candidates.add(0)
			rng := rand.New(rand.NewSource(17))
			for range 100 {
				probe := generateProbe(parsed, rng)
				for _, value := range []string{probe, "東京 " + probe + " \xff", strings.ReplaceAll(probe, "K", "K")} {
					hits := keywordHits{invalid: !isASCIIKeyword(value)}
					require.Equal(t, rule.detect(value, nil), evaluateRuleSet([]compiledRule{rule}, candidates, value, &hits, allRuleMatches, nil))
				}
			}
		})
	}
}

func TestRequiredPunctuationNecessaryConditions(t *testing.T) {
	for _, tc := range []struct {
		expression string
		allowed    []string
		rejected   []string
	}{
		{
			expression: `https://[^ /]+:[^ @]+@example\.test`,
			allowed:    []string{"https://user:pass@example.test", "東京 https://user:pass@example.test \xff"},
			rejected:   []string{"https://example.test/path", "https:user:pass@example.test", "https//userpass@example.test", "https://user:pass@exampletest"},
		},
		{
			expression: `(?:a:b@|c:d#)`,
			allowed:    []string{"a:b@", "c:d#"},
			rejected:   []string{"ab@cd#"},
		},
		{
			expression: `(?:a:b|c@d)`,
			allowed:    []string{"a:b", "c@d", ""},
		},
		{
			expression: `(?:x@y)?a:b`,
			allowed:    []string{"a:b"},
			rejected:   []string{"x@yab"},
		},
		{
			expression: `(?:x@y){0,3}a:b`,
			allowed:    []string{"a:b"},
			rejected:   []string{"x@yab"},
		},
		{
			expression: `(?:x@y){1,3}a:b`,
			allowed:    []string{"x@ya:b"},
			rejected:   []string{"a:b"},
		},
		{
			expression: `(?:x@y)*a:b`,
			allowed:    []string{"a:b"},
			rejected:   []string{"x@yab"},
		},
		{
			expression: `(?:x@y)+a:b`,
			allowed:    []string{"x@ya:b"},
			rejected:   []string{"a:b"},
		},
		{
			expression: `[;λ]`,
			allowed:    []string{";", "λ", ""},
		},
		{
			expression: `(?i:K)@`,
			allowed:    []string{"k@", "K@"},
			rejected:   []string{"k", "K"},
		},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			parsed, err := syntax.Parse(tc.expression, syntax.Perl)
			require.NoError(t, err)
			required := requiredPunctuation(parsed)
			for _, value := range tc.allowed {
				var presence punctuationPresence
				require.True(t, presence.contains(value, required), "value=%q", value)
			}
			for _, value := range tc.rejected {
				var presence punctuationPresence
				require.False(t, presence.contains(value, required), "value=%q", value)
			}
		})
	}
}

func TestRequiredPunctuationSharesPresenceAcrossCandidates(t *testing.T) {
	const value = "https://example.test"
	var presence punctuationPresence
	for _, tc := range []struct {
		punctuation string
		want        bool
	}{
		{"/:@", false},
		{"/:.", true},
		{"@", false},
		{"#", false},
		{"@#", false},
		{"", true},
		{"/:.", true},
	} {
		var required asciiSet
		for i := range len(tc.punctuation) {
			required.add(tc.punctuation[i])
		}
		require.Equal(t, tc.want, presence.contains(value, required), "required=%q", tc.punctuation)
	}
}

func TestPunctuationChoiceSharesExactPresenceCache(t *testing.T) {
	const value = "https://example.test"
	var presence punctuationPresence
	var exact asciiSet
	exact.add(':')
	exact.add('@')
	require.False(t, presence.contains(value, exact))
	for _, tc := range []struct {
		choice string
		want   bool
	}{
		{":@", true},
		{"#@", false},
		{"#/", true},
		{".", true},
		{"$", false},
		{"", false},
	} {
		var choice asciiSet
		for i := range len(tc.choice) {
			choice.add(tc.choice[i])
		}
		require.Equal(t, tc.want, presence.containsAny(value, choice), tc.choice)
	}
}

func TestRequiredASCIINecessaryCondition(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
		value      string
		possible   bool
		matches    bool
	}{
		{name: "Unicode noise", expression: `token`, value: strings.Repeat("界", 682)},
		{name: "mixed literal", expression: `λa`, value: "λλa界", possible: true, matches: true},
		{name: "strict class", expression: `[0-9]{2}`, value: "界界"},
		{name: "Unicode alternate", expression: `(?:[0-9]{2}|λ+)`, value: "λλ", possible: true, matches: true},
		{name: "strict alternates", expression: `(?:aλ|b界)`, value: "λ界"},
		{name: "optional ASCII", expression: `a?λ+`, value: "λλ", possible: true, matches: true},
		{name: "optional repeat", expression: `(?:ab){0,2}λ+`, value: "λλ", possible: true, matches: true},
		{name: "required repeat", expression: `(?:aλ){1,2}`, value: "λλ"},
		{name: "required capture", expression: `(aλ)+`, value: "λλ"},
		{name: "star", expression: `(?:aλ)*λ+`, value: "λλ", possible: true, matches: true},
		{name: "empty alternate", expression: `(?:aλ|)`, value: "λ", possible: true, matches: true},
		{name: "Kelvin fold", expression: `(?i:k+)`, value: "KK", possible: true, matches: true},
		{name: "long s fold", expression: `(?i:s+)`, value: "ſſ", possible: true, matches: true},
		{name: "Unicode literal fold", expression: `(?i:K+)`, value: "KK", possible: true, matches: true},
		{name: "strict folded suffix", expression: `(?i:ka)`, value: "Kλ"},
		{name: "mixed folded match", expression: `(?i:ka)`, value: "界Kaλ", possible: true, matches: true},
		{name: "folded class aliases", expression: `(?i:[a-z]+)`, value: "Kſ", possible: true, matches: true},
		{name: "ASCII class boundary", expression: `[\x00-\x7f]`, value: "界"},
		{name: "NUL is ASCII", expression: `[\x00-\x7f]`, value: "\x00界", possible: true, matches: true},
		{name: "Unicode class boundary", expression: `[\x7f-\x{100}]`, value: "\u0080", possible: true, matches: true},
		{name: "negated class", expression: `[^a]+`, value: "λ", possible: true, matches: true},
		{name: "wildcard", expression: `.+`, value: "λ", possible: true, matches: true},
		{name: "zero width", expression: `\b`, value: "λ", possible: true},
		{name: "ASCII input stays eligible", expression: `a`, value: "b", possible: true},
		{name: "malformed bytes lack ASCII", expression: `a`, value: "\xff\xfe"},
		{name: "malformed bytes with ASCII", expression: `.a`, value: "\xffa", possible: true, matches: true},
		{name: "malformed bytes decode RuneError", expression: `\x{FFFD}`, value: "\xff", possible: true, matches: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := syntax.Parse(tc.expression, syntax.Perl)
			require.NoError(t, err)
			required := requiredASCII(parsed)
			hits := keywordHits{invalid: true, nonASCII: !isASCIIKeyword(tc.value)}
			require.Equal(t, tc.possible, required.possible(tc.value, &hits))
			require.Equal(t, tc.matches, regexp.MustCompile(tc.expression).MatchString(tc.value))

			rule, err := compileRuleSpec(catalogRuleSpec{ID: "ascii-required", Regex: tc.expression}, nil)
			require.NoError(t, err)
			var candidates ruleSet
			candidates.add(0)
			require.Equal(t, rule.detect(tc.value, nil), evaluateRuleSet([]compiledRule{rule}, candidates, tc.value, &hits, allRuleMatches, nil))
		})
	}
}

func TestRequiredASCIIInputPresenceResetsBetweenValues(t *testing.T) {
	parsed, err := syntax.Parse(`[0-9]`, syntax.Perl)
	require.NoError(t, err)
	required := requiredASCII(parsed)
	var hits keywordHits
	for _, tc := range []struct {
		name     string
		value    string
		possible bool
	}{
		{name: "absent", value: "界"},
		{name: "present after absent", value: "界1", possible: true},
		{name: "absent after present", value: "界"},
		{name: "present before malformed byte", value: "1\xff", possible: true},
		{name: "malformed bytes after present", value: "\xff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hits.reset(nil, nil)
			hits.nonASCII = true
			require.Equal(t, tc.possible, required.possible(tc.value, &hits))
			require.Equal(t, tc.possible, required.possible(tc.value, &hits))
		})
	}
}

func TestRequiredRunUsesOnlyRelevantUnicode(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
		value      string
		invalid    bool
		reject     bool
		matches    bool
	}{
		{name: "unrelated Unicode rejects", expression: `(?i)[a-z]{8}`, value: "界12345678é", reject: true},
		{name: "unrelated Unicode preserves run", expression: `(?i)[a-z]{8}`, value: "界abcdefghé", matches: true},
		{name: "Kelvin participates", expression: `(?i)[a-z]{8}`, value: strings.Repeat("K", 8), matches: true},
		{name: "long s participates", expression: `(?i)[a-z]{8}`, value: strings.Repeat("ſ", 8), matches: true},
		{name: "explicit Unicode class", expression: `[a-zé]{8}`, value: strings.Repeat("é", 8), matches: true},
		{name: "Unicode alternate", expression: `(?i:[a-z]{8})|é{8}`, value: strings.Repeat("é", 8), matches: true},
		{name: "strict run ignores aliases", expression: `[0-9]{8}`, value: "Kabcdefgh", reject: true},
		{name: "malformed bytes lack aliases", expression: `(?i)[a-z]{8}`, value: "\xff12345678", reject: true},
		{name: "unknown alias inventory", expression: `(?i)[a-z]{8}`, value: "界12345678é", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := syntax.Parse(tc.expression, syntax.Perl)
			require.NoError(t, err)
			run := requiredASCIIRun(parsed)
			hits := keywordHits{
				invalid:        tc.invalid,
				nonASCII:       !isASCIIKeyword(tc.value),
				asciiFoldAlias: containsUnicodeASCIIFold(tc.value),
			}
			require.Equal(t, tc.reject, run.width > 0 && run.canReject(&hits) && !run.exists(tc.value))
			require.Equal(t, tc.matches, regexp.MustCompile(tc.expression).MatchString(tc.value))
		})
	}
}

func TestFoldedKeywordGuidancePreservesUnicodeContexts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
		keyword    string
		value      string
	}{
		{name: "UTF8 left context and rune widths", expression: `(?i)\btoken:([^ ]{4})`, keyword: "token", value: "étoken:界界界界 token:abcd"},
		{name: "invalid UTF8 left context", expression: `(?i)\btoken:([^ ]{4})`, keyword: "token", value: "\xfftoken:abcd"},
		{name: "Kelvin keyword fallback", expression: `(?i)\btoken:([^ ]{4})`, keyword: "token", value: "étoKen:abcd"},
		{name: "long s keyword fallback", expression: `(?i)\bstart:([^ ]{4})`, keyword: "start", value: "Xſtart:abcd"},
		{name: "long s lacks ASCII word boundary", expression: `(?i)\bstart:([^ ]{4})`, keyword: "start", value: "éſtart:abcd"},
		{name: "Unicode optional prefix fallback", expression: `(?i)[é]{0,2}token:([^ ]{4})`, keyword: "token", value: "éétoken:abcd"},
		{name: "Unicode window fallback", expression: `(?i)[é]{2}token:([^ ]{4})`, keyword: "token", value: "éétoken:abcd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			indices := regexp.MustCompile(tc.expression).FindAllStringSubmatchIndex(tc.value, -1)
			catalog, err := compileCatalog([]catalogRuleSpec{{
				ID:          "folded-guidance",
				Regex:       tc.expression,
				Keywords:    []string{tc.keyword},
				SecretGroup: 1,
				ValidateContext: func(value string, start, end int, secret string) contextValidation {
					// Anchored evaluation must report original byte positions, including
					// the entire Unicode prefix rather than its ASCII keyword alone.
					if value != tc.value {
						return contextValidation{}
					}
					for _, index := range indices {
						if start == index[0] && end == index[1] && secret == value[index[2]:index[3]] {
							return contextValidation{accepted: true}
						}
					}
					return contextValidation{}
				},
			}})
			require.NoError(t, err)
			expected := make([]Match, len(indices))
			for i := range expected {
				expected[i] = Match{RuleID: "folded-guidance"}
			}
			require.ElementsMatch(t, expected, catalogFindings(catalog, tc.value, false))
			require.ElementsMatch(t, expected, catalogFindings(catalog, tc.value, true))
			require.ElementsMatch(t, expected[:min(len(expected), 1)], selectedCatalogFindings(catalog, tc.value, true, catalog.all, uniqueRuleMatches))
		})
	}
}

func TestRequiredLiteralAlternativesPreserveRegexpMatches(t *testing.T) {
	for _, tc := range []struct {
		name, expression, value string
		reject                  bool
	}{
		{"exact request case", `(?i:token_)[a-z]+|GET[ \t]+[^\r\n]+`, "é public get target", true},
		{"folded token", `(?i:token_)[a-z]+|GET[ \t]+[^\r\n]+`, "ToKeN_ab", false},
		{"Unicode request body", `(?i:token_)[a-z]+|GET[ \t]+[^\r\n]+`, "GET 界", false},
		{"Kelvin alias", `(?i:token_)[a-z]+|GET[ \t]+[^\r\n]+`, "toKen_ab", false},
		{"long s alias", `(?i:secret_)[a-z]+|GET[ \t]+[^\r\n]+`, "ſecret_ab", false},
		{"Unicode alternative", `token_|éé`, "éé", false},
		{"optional literal", `(?:token_){0,3}`, "", false},
		{"required repeated literal", `(?:token_){2,4}`, "public text", true},
		{"nested alternatives", `(?:left_)?middle_(?:right|tail)`, "middle_tail", false},
		{"malformed UTF8 context", `token_|GET[ \t]+[^\r\n]+`, "\xffGET value", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := syntax.Parse(tc.expression, syntax.Perl)
			require.NoError(t, err)
			alternatives := requiredLiteralAlternatives(parsed)
			hits := keywordHits{nonASCII: !isASCIIKeyword(tc.value), asciiFoldAlias: containsUnicodeASCIIFold(tc.value)}
			possible := literalAlternativesPossible(tc.value, alternatives, &hits)
			require.Equal(t, !tc.reject, possible)
			if regexp.MustCompile(tc.expression).MatchString(tc.value) {
				require.True(t, possible, "a necessary-literal gate must preserve every regexp match")
			}
		})
	}

	// Exceeding the proof budget must not silently drop the last alternatives.
	branches := make([]*syntax.Regexp, analysisMaxStrings+1)
	for i := range branches {
		branches[i] = &syntax.Regexp{Op: syntax.OpLiteral, Rune: []rune("term-" + string(rune('!'+i)) + "-end")}
	}
	parsed := &syntax.Regexp{Op: syntax.OpAlternate, Sub: branches}
	alternatives := requiredLiteralAlternatives(parsed)
	reference := regexp.MustCompile(parsed.String())
	for _, branch := range branches {
		value := string(branch.Rune)
		require.True(t, reference.MatchString(value))
		require.True(t, literalAlternativesPossible(value, alternatives, &keywordHits{}))
	}
}
