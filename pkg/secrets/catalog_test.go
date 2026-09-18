package secrets

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type nativeCatalogFixture struct {
	ID         string                `json:"id"`
	Positive   []string              `json:"positive"`
	Negative   []string              `json:"negative"`
	Diagnostic []string              `json:"diagnostic,omitempty"`
	Evidence   nativeCatalogEvidence `json:"evidence"`
}

type nativeCatalogEvidence struct {
	Sources       []string `json:"sources"`
	Level         string   `json:"level"`
	Documented    []string `json:"documented"`
	Assumptions   []string `json:"assumptions"`
	FixtureMethod string   `json:"fixture_method"`
}

// The loaded map and its slices are shared read-only by tests, fuzzers and benchmarks.
var loadNativeCatalogFixtures = sync.OnceValues(func() (map[string]nativeCatalogFixture, error) {
	paths, err := filepath.Glob("testdata/catalog/*.json")
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no native catalog fixture shards")
	}
	known := make(map[string]bool, len(nativeRuleSpecs))
	for _, spec := range nativeRuleSpecs {
		if known[spec.ID] {
			return nil, fmt.Errorf("duplicate native catalog ID %q", spec.ID)
		}
		known[spec.ID] = true
	}
	fixtures := make(map[string]nativeCatalogFixture, len(known))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var shard []nativeCatalogFixture
		if err := json.Unmarshal(data, &shard); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		for _, fixture := range shard {
			if !known[fixture.ID] {
				return nil, fmt.Errorf("%s: unknown fixture ID %q", path, fixture.ID)
			}
			if _, exists := fixtures[fixture.ID]; exists {
				return nil, fmt.Errorf("%s: duplicate fixture ID %q", path, fixture.ID)
			}
			if len(fixture.Positive) == 0 || len(fixture.Negative) == 0 {
				return nil, fmt.Errorf("%s: %q requires positive and negative fixtures", path, fixture.ID)
			}
			evidence := fixture.Evidence
			if len(evidence.Sources) == 0 || len(evidence.Documented) == 0 ||
				evidence.Assumptions == nil || strings.TrimSpace(evidence.FixtureMethod) == "" {
				return nil, fmt.Errorf("%s: %q requires an explicit evidence record", path, fixture.ID)
			}
			switch evidence.Level {
			case "generator", "specification", "documented-prefix", "example-only":
			default:
				return nil, fmt.Errorf("%s: %q has an invalid evidence level", path, fixture.ID)
			}
			for _, reference := range evidence.Sources {
				source, err := url.Parse(reference)
				if err != nil || source.Scheme != "https" || source.Host == "" {
					return nil, fmt.Errorf("%s: %q has an invalid evidence source", path, fixture.ID)
				}
			}
			for _, statements := range [][]string{evidence.Documented, evidence.Assumptions} {
				for _, fact := range statements {
					if strings.TrimSpace(fact) == "" {
						return nil, fmt.Errorf("%s: %q has an empty evidence statement", path, fixture.ID)
					}
				}
			}
			fixtures[fixture.ID] = fixture
		}
	}
	for _, spec := range nativeRuleSpecs {
		if _, exists := fixtures[spec.ID]; !exists {
			return nil, fmt.Errorf("missing fixture for native catalog ID %q", spec.ID)
		}
	}
	return fixtures, nil
})

func nativeCatalogFixtures(t testing.TB) map[string]nativeCatalogFixture {
	t.Helper()
	fixtures, err := loadNativeCatalogFixtures()
	require.NoError(t, err)
	return fixtures
}

func TestNativeCatalogNegativeFixtures(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	fixtures := nativeCatalogFixtures(t)
	for index, spec := range nativeRuleSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			for i, value := range fixtures[spec.ID].Negative {
				t.Run(fmt.Sprintf("negative-%d", i), func(t *testing.T) {
					findings := assertCatalogRuleMatchesFullScan(t, catalog, index, value)
					require.False(t, hasRuleFinding(findings, spec.ID), "malformed or public fixture was detected")
				})
			}
		})
	}
}

func TestNativeCatalogRejectsPublicAndMalformedIndicators(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	for _, test := range []struct{ name, value string }{
		{"build-digest", "8f3a12b6c9d047e5a1f2b3c4d5e60718293abcde"},
		{"aws-key-id", "AKIA" + "QWERTYUIOPASDFGH"},
		{"twilio-key-sid", "SK" + "0123456789abcdef0123456789abcdef"},
		{"oauth-client-id", "adobe_client_id=" + "0123456789abcdef0123456789abcdef"},
		{"publishable-key", "FLWPUBK_TEST-" + "0123456789abcdef0123456789abcdef-X"},
		{"bedrock-marker", "bedrock-api-key-" + "YmVkcm9jay5hbWF6b25hd3MuY29t"},
		{"slack-lookalike-host", "https://hooksXslackYcom/services/" + "T12345678/B12345678/Ab12Cd34Ef56Gh78Ij90Kl12"},
		{"key-armor-prose", "-----BEGIN RSA PRIVATE KEY-----\n" + strings.Repeat("not-a-key!", 10) + "\nKEY-----"},
		{"encoded-jwt-marker", "ZXlKaGJHY2lPaU" + strings.Repeat("A", 80) + "=="},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, Verdict{}, policy.Detect(test.value))
			batch := policy.NewBatchDetector()
			require.Equal(t, Verdict{}, batch.Detect(test.value))
			require.Equal(t, Verdict{}, batch.Detect(test.value))
		})
	}
}

func TestNativeKeywordFoldingPreservesUnicodeRegexMatches(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	for _, value := range []string{
		"AWS_SECRET_ACCESS_KEY=" + "Ab12Cd34Ef56Gh78Ij90" + "Kl12Mn34Op56Qr78St90",
		"aws_ſecret_acceſſ_key=" + "Ab12Cd34Ef56Gh78Ij90" + "Kl12Mn34Op56Qr78St90",
		"AWS_SECRET_ACCESS_KEY=" + "Ab12Cd34Ef56Gh78Ij90" + "Kl12Mn34Op56Qr78St90",
	} {
		want := fullScanPolicyVerdict(policy, value)
		require.Contains(t, want.Matches, Match{RuleID: "aws-secret-access-key"})
		require.Equal(t, want, policy.Detect(value))
		batch := policy.NewBatchDetector()
		require.Equal(t, want, batch.Detect(value))
	}
}

func TestNativeVaultScopeExcludesLegacyPrefixes(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	fixtures := nativeCatalogFixtures(t)
	for _, test := range []struct {
		id, modern, legacy string
	}{
		{"vault-service-token", "hvs.", "s."},
		{"vault-batch-token", "hvb.", "b."},
		{"vault-recovery-token", "hvr.", "r."},
	} {
		t.Run(test.id, func(t *testing.T) {
			value := fixtures[test.id].Positive[0]
			require.True(t, strings.Contains(value, test.modern), "fixture must represent a modern Vault token")
			legacy := strings.Replace(value, test.modern, test.legacy, 1)
			want := Verdict{Matches: []Match{{RuleID: test.id}}}
			require.Equal(t, want, policy.Detect(value))
			// Historic credentials are intentionally outside the native scope;
			// absence of this finding is not a claim that the token is invalid.
			require.False(t, hasRuleFinding(policy.Detect(legacy).Matches, test.id))
			batch := policy.NewBatchDetector()
			require.Equal(t, want, batch.Detect(value))
			require.False(t, hasRuleFinding(batch.Detect(legacy).Matches, test.id))
		})
	}
}

// Synthetic catalogs isolate algorithm contracts from changing provider inventories.
func compileTestCatalog(t testing.TB, specs ...catalogRuleSpec) *compiledCatalog {
	t.Helper()
	catalog := &compiledCatalog{byID: make(map[string]uint16, len(specs))}
	var keywords []matcherKeyword
	for i, spec := range specs {
		index := uint16(i)
		ruleKeywords := effectiveKeywords(spec)
		rule, err := compileRuleSpec(spec, ruleKeywords)
		require.NoError(t, err)
		catalog.rules = append(catalog.rules, rule)
		catalog.byID[spec.ID] = index
		catalog.all.add(index)
		if len(ruleKeywords) == 0 || !rule.plan.keywordRequired {
			catalog.keywordless.add(index)
		}
		if rule.plan.strategy != strategyFullScan {
			catalog.guided.add(index)
		}
		if rule.plan.strategy == strategyAnchored && rule.plan.leadingBoundary {
			catalog.boundary.add(index)
		}
		for _, keyword := range ruleKeywords {
			keywords = append(keywords, matcherKeyword{value: keyword, rule: index})
		}
	}
	var err error
	catalog.pairMatcher, err = newCatalogPairMatcher(keywords)
	require.NoError(t, err)
	return catalog
}
