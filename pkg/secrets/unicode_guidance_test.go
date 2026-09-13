package secrets

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestKeywordGuidancePreservesMixedByteContexts(t *testing.T) {
	tests := []struct {
		name   string
		rule   CustomRule
		values []struct {
			name, value string
			matched     bool
		}
	}{
		{name: "word boundaries", rule: CustomRule{Regex: `\bTOK_[A-Z]{4}\b`}, values: []struct {
			name, value string
			matched     bool
		}{
			{"multibyte neighbors", "界TOK_ABCDé", true},
			{"four-byte neighbors", "😀TOK_ABCD𐐀", true},
			{"invalid neighbors", "\xff\xc0TOK_ABCD\xaf\x80", true},
			{"ascii word before", "界xTOK_ABCDé", false},
			{"ascii word after", "界TOK_ABCDxé", false},
		}},
		{name: "line anchors", rule: CustomRule{Regex: `(?m)^TOK_[A-Z]{4}$`}, values: []struct {
			name, value string
			matched     bool
		}{
			{"line after unicode", "界\nTOK_ABCD\né", true},
			{"not a line start", "界TOK_ABCD\n", false},
			{"not a line end", "\nTOK_ABCD界", false},
			{"unicode line separator is not LF", "\u2028TOK_ABCD\u2028", false},
		}},
		{name: "absolute anchors", rule: CustomRule{Regex: `\ATOK_[A-Z]{4}\z`}, values: []struct {
			name, value string
			matched     bool
		}{
			{"exact input", "TOK_ABCD", true},
			{"unicode prefix", "界TOK_ABCD", false},
			{"unicode suffix", "TOK_ABCD界", false},
			{"invalid prefix", "\xffTOK_ABCD", false},
		}},
		{name: "nonword assertions", rule: CustomRule{Regex: `\BTOK_[A-Z]{4}\B`}, values: []struct {
			name, value string
			matched     bool
		}{
			{"ascii word context", "界xTOK_ABCDyé", true},
			{"unicode is not ASCII word context", "界TOK_ABCDé", false},
		}},
		{name: "bounded window", rule: CustomRule{Regex: `[A-Z]{2}:TOK:[A-Z]{4}`}, values: []struct {
			name, value string
			matched     bool
		}{
			{"multibyte window edges", "界AB:TOK:CDEF😀", true},
			{"invalid window edges", "\xffAB:TOK:CDEF\xc0", true},
			{"multibyte inside ASCII match", "AB:TOK:CD界F", false},
		}},
		{name: "optional prefix", rule: CustomRule{Regex: `[A-Z._]{0,4}TOK_[A-Z]{4}`}, values: []struct {
			name, value string
			matched     bool
		}{
			{"prefix after multibyte rune", "界AB._TOK_CDEFé", true},
			{"zero prefix after invalid byte", "\xffTOK_CDEF", true},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.rule.ID = "mixed-context-rule"
			policy, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{test.rule}})
			if err != nil {
				t.Fatal("policy compilation failed")
			}
			batch := policy.NewBatchDetector()
			for _, probe := range test.values {
				var want Verdict
				if probe.matched {
					want.Matches = []Match{{RuleID: test.rule.ID}}
				}
				if !slices.Equal(want.Matches, fullScanPolicyVerdict(policy, probe.value).Matches) {
					t.Fatalf("probe %s: independent expectation disagrees with reference", probe.name)
				}
				if !slices.Equal(want.Matches, policy.Detect(probe.value).Matches) || !slices.Equal(want.Matches, batch.Detect(probe.value).Matches) || !slices.Equal(want.Matches, batch.Detect(probe.value).Matches) {
					t.Fatalf("probe %s: optimized verdict differs", probe.name)
				}
			}
		})
	}
}

func TestKeywordGuidancePreservesUnicodeConsumingRules(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{
		{ID: "folded", Regex: `FOLD:(?i:[a-z]{4})`},
		{ID: "folded-prefix", Regex: `(?i)KEL_[A-Z]{4}`},
		{ID: "wide", Regex: `WIDE:[^\n]{4}!`},
		{ID: "replacement", Regex: `BAD:\x{FFFD}{2}!`},
		{ID: "literal", Regex: `LIT:界{4}!`},
	}})
	if err != nil {
		t.Fatal("policy compilation failed")
	}
	for _, probe := range []struct{ name, value, id string }{
		{"folded ASCII aliases", "界FOLD:KſKſé", "folded"},
		{"folded prefix requires fallback", "界KEL_ABCDé", "folded-prefix"},
		{"four multibyte runes", "éWIDE:😀😀😀😀!界", "wide"},
		{"two invalid bytes", "界BAD:\xff\xfe!é", "replacement"},
		{"literal Unicode", "éLIT:界界界界!😀", "literal"},
		{"wrong folded alphabet", "FOLD:KſK界", ""},
		{"wrong rune count", "WIDE:😀😀😀!", ""},
		{"one replacement rune", "BAD:�!", ""},
	} {
		var want Verdict
		if probe.id != "" {
			want.Matches = []Match{{RuleID: probe.id}}
		}
		if !slices.Equal(want.Matches, fullScanPolicyVerdict(policy, probe.value).Matches) || !slices.Equal(want.Matches, policy.Detect(probe.value).Matches) {
			t.Fatalf("probe %s: Unicode-consuming rule changed", probe.name)
		}
	}
}

func TestKeywordGuidanceMixedInputOverflowAndReset(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{CustomRules: []CustomRule{{ID: "overflow", Regex: `\bTOK_[A-Z]{4}\b`}}})
	if err != nil {
		t.Fatal("policy compilation failed")
	}
	batch := policy.NewBatchDetector()
	for index, value := range []string{
		"界" + strings.Repeat("TOK_? ", maxKeywordHits+1) + "TOK_ABCDé",
		"\xffTOK_ABCD " + strings.Repeat("TOK_? ", maxKeywordHits+1),
		"TOK_ABCD",
		"TOK_ABCD界",
		"TOK_ABC",
		"éTOK_ABC\xff",
	} {
		want := fullScanPolicyVerdict(policy, value)
		if !slices.Equal(want.Matches, policy.Detect(value).Matches) || !slices.Equal(want.Matches, batch.Detect(value).Matches) {
			t.Fatalf("probe %d: overflow or reused hit state changed the verdict", index)
		}
	}
}

func TestUnicodeKeywordPlansPreserveIndependentCaptureOracle(t *testing.T) {
	tests := []struct {
		name, expression string
		keywords         []string
		values           []string
	}{
		{
			name:       "alternative request starts",
			expression: `(?i:\b(?:GET|POST)) ([^\n]{1,8})(?:\n|$)`,
			keywords:   []string{"get", "post"},
			values: []string{
				"GET deny\nPOST 界😀\n",
				"界GET deny\npost \xff\xfe\n",
				"éPOſT Kſ\n",
				"GET too-long-body\n界POST valid\n",
				"界" + strings.Repeat("GET? ", maxKeywordHits+1) + "\nPOST deny\nGET valid\n",
				"GET deny\n" + strings.Repeat("POST? ", maxKeywordHits+1) + "\nPOST 界\n",
			},
		},
		{
			name:       "exact alternative starts with fold aliases in tail",
			expression: `\b(?:GET|PUT) ((?i:[a-z]){4})(?:\n|$)`,
			keywords:   []string{"get", "put"},
			values:     []string{"界GET KſKſ\nPUT deny\n", "\xffPUT deny\nGET pass\n"},
		},
		{
			name:       "Unicode branch retains its earlier match start",
			expression: `((?:GET|界GET)[^\n]{4})`,
			keywords:   []string{"get"},
			values:     []string{"界GETabcd GETefgh", "\xff界GET😀😀😀😀", "GETabcd界GETefgh"},
		},
		{
			name:       "nullable Unicode prefix",
			expression: `((?:界)?GET[^\n]{4})`,
			keywords:   []string{"get"},
			values:     []string{"界GETabcd GETefgh", "\xffGET😀😀😀😀"},
		},
		{
			name:       "bounded Unicode windows",
			expression: `(?:\A|[^A-Z])((?:GET|POST)[^\n]{4})(?:\n|\z)`,
			keywords:   []string{"get", "post"},
			values: []string{
				strings.Repeat("界", 40) + "GET😀😀😀😀\n" + strings.Repeat("é", 40),
				"\xffGETabcd\n界POSTefgh\n",
				strings.Repeat("é", 30) + "界GETabcdX" + strings.Repeat("界", 40),
				"GETabcd\n" + strings.Repeat("POST? ", maxKeywordHits+1) + "界GET😀😀😀😀\n",
			},
		},
		{
			name:       "replacement rune windows preserve malformed bytes",
			expression: `(?:\x{FFFD}|!)(GET[^\n]{2})(?:\n|\z)`,
			keywords:   []string{"get"},
			values: []string{
				strings.Repeat("é", 20) + "\xffGET😀界\n" + strings.Repeat("界", 20),
				strings.Repeat("😀", 20) + "!GET\xfe\x80\n",
				"界GETab\n", "\xef\xbf\xbdGETab\n",
			},
		},
		{
			name:       "nullable ASCII prefix with Unicode body",
			expression: `[A-Z._]{0,4}(GET[^\n]{4})`,
			keywords:   []string{"get"},
			values:     []string{"界AB._GET😀😀😀😀", "GETabcdGETefgh"},
		},
		{
			name:       "overlapping alternative keywords",
			expression: `\b((?:ABA|BAB)[^\n]{1,4})`,
			keywords:   []string{"aba", "bab"},
			values:     []string{"界ABABABA😀 BABAB\n", "ABABABABAB", "\xffBAB😀ABA界"},
		},
		{
			name:       "line and absolute assertions",
			expression: `(?m)^(GET|POST)[^\n]{1,4}$`,
			keywords:   []string{"get", "post"},
			values:     []string{"界GETabcd\nPOST😀😀😀😀\n", "GETabcdX\nPOST界\n", "\xff\nGET😀😀😀😀"},
		},
	}
	type observation struct {
		start, end int
		secret     string
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oracle := regexp.MustCompile(test.expression)
			var value string
			var observed []observation
			catalog, err := compileCatalog([]catalogRuleSpec{{
				ID: "unicode-oracle", Regex: test.expression, Keywords: test.keywords, SecretGroup: 1,
				ValidateContext: func(original string, start, end int, secret string) contextValidation {
					if original != value {
						t.Fatal("context validator received a sliced value")
					}
					observed = append(observed, observation{start, end, secret})
					return contextValidation{accepted: secret != "deny"}
				},
			}})
			if err != nil {
				t.Fatal("catalog compilation failed")
			}
			policy := CompiledPolicy{catalog: catalog}
			batch := policy.NewBatchDetector()
			for probe, input := range test.values {
				value = input
				var wantObservations []observation
				var want []Match
				for _, indices := range oracle.FindAllStringSubmatchIndex(value, -1) {
					secret := value[indices[2]:indices[3]]
					wantObservations = append(wantObservations, observation{indices[0], indices[1], secret})
					if secret != "deny" {
						want = append(want, Match{RuleID: "unicode-oracle"})
					}
				}
				observed = nil
				got := catalogFindings(catalog, value, true)
				if !slices.Equal(want, got) || !slices.Equal(wantObservations, observed) {
					t.Fatalf("probe %d: complete matches or capture byte offsets differ from standard regexp", probe)
				}
				if len(want) > 1 {
					want = want[:1]
				}
				if !slices.Equal(want, policy.Detect(value).Matches) || !slices.Equal(want, batch.Detect(value).Matches) {
					t.Fatalf("probe %d: public first-accepted finding differs from standard regexp", probe)
				}
			}
		})
	}
}

func TestContextExecutionPreservesExistenceAndCaptureAcceptance(t *testing.T) {
	for _, expression := range []string{
		`\b(?:GET|POST) ([^\n]{1,8})(?:\n|$)`,
		`(?m)^(?:GET|POST) ([^\n]{1,8})$`,
		`\A(?:GET|POST) ([^\n]{1,8})\z`,
	} {
		for _, entropy := range []float64{0, 1} {
			catalog, err := compileCatalog([]catalogRuleSpec{{
				ID: "context-execution", Regex: expression, SecretGroup: 1, Entropy: entropy,
				Keywords: []string{"get", "post"},
			}})
			if err != nil {
				t.Fatal("policy compilation failed")
			}
			policy := CompiledPolicy{catalog: catalog}
			oracle := regexp.MustCompile(expression)
			for probe, value := range []string{
				"GET aaaaaaaa\nPOST abcd1234\n",
				"GET too-long-body\nPOST abcd1234\n",
				"界GET abcd1234\n",
				"GET aaaaaaaa\n",
				"GET abcd1234",
				strings.Repeat("GET? ", maxKeywordHits+1) + "\nPOST abcd1234\n",
			} {
				var want []Match
				for _, indices := range oracle.FindAllStringSubmatchIndex(value, -1) {
					if entropy == 0 || shannonEntropy(value[indices[2]:indices[3]]) > entropy {
						want = []Match{{RuleID: "context-execution"}}
						break
					}
				}
				if !slices.Equal(want, policy.Detect(value).Matches) {
					t.Fatalf("probe %d: context execution differs from standard regexp", probe)
				}
			}
		}
	}
}
