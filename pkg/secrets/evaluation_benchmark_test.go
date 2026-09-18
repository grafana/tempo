package secrets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// BenchmarkDetectionWorkloads measures uncached public detection against the
// unfiltered semantic oracle on the same inputs. The oracle is not a historical
// release baseline. These synthetic workloads deliberately include slow paths;
// they are not a weighted model of production traffic.
func BenchmarkDetectionWorkloads(b *testing.B) {
	policies := []struct {
		name string
		spec Policy
	}{
		{name: "catalog"},
		{name: fmt.Sprintf("custom-%d", maxPolicyCustomRules), spec: evaluationCustomPolicy(maxPolicyCustomRules, `tenant_[A-Za-z0-9]{16,40}`)},
	}
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(b, err)
	positive := findCatalogRuleWitness(b, catalog, 0)
	var curlPositive string
	for _, value := range nativeCatalogFixtures(b)["madkudu-api-basic-credential"].Positive {
		if strings.HasPrefix(value, "curl ") {
			curlPositive = value
			break
		}
	}
	require.NotEmpty(b, curlPositive)
	var overflow string
	for _, spec := range nativeRuleSpecs {
		nativeIndex, ok := catalog.byID[spec.ID]
		if ok && catalog.guided.has(nativeIndex) {
			overflow = strings.Repeat(effectiveKeywords(spec)[0]+" ", maxKeywordHits+32)
			break
		}
	}
	require.NotEmpty(b, overflow, "overflow workload needs a guided catalog rule")
	values := []struct{ name, value string }{
		{"url", "https://api.example.com/v1/orders/123456?region=us-east-1"},
		{"ascii-128KiB", strings.Repeat("~", 128<<10)},
		{"unicode-4KiB", strings.Repeat("~é~", 1024)},
		{"unicode-keywords-4KiB", strings.Repeat("api é token Σ ", 256)},
		{"invalid-utf8-4KiB", strings.Repeat("\xff~", 2048)},
		{"guided-hit-overflow", overflow},
		{"hex-near-misses-4KiB", strings.Repeat("0123456789abcdef0123456789abcdef012345678~", 100)},
		{"positive", positive},
		{"late-positive-128KiB", strings.Repeat("~", (128<<10)-len(positive)) + positive},
		{"contextual-curl", curlPositive},
		{"contextual-curl-start-128KiB", curlPositive + strings.Repeat("~", (128<<10)-len(curlPositive))},
		{"contextual-curl-late-128KiB", strings.Repeat("~", (128<<10)-len(curlPositive)) + curlPositive},
		{"custom-positive", "tenant_Ab12Cd34Ef56Gh78"},
	}
	for _, policyCase := range policies {
		policy, err := testPolicyCompiler.CompilePolicy(policyCase.spec)
		require.NoError(b, err)
		for _, valueCase := range values {
			// Check the observed contract before measuring either path. Cache hits
			// and misses have separate existing BatchDetector/processor benchmarks.
			want := fullScanPolicyVerdict(policy, valueCase.value)
			require.Equal(b, want, policy.Detect(valueCase.value), valueCase.name)
			b.Run(policyCase.name+"/"+valueCase.name+"/optimized", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(valueCase.value)))
				for range b.N {
					benchmarkVerdict = policy.Detect(valueCase.value)
				}
			})
			b.Run(policyCase.name+"/"+valueCase.name+"/unfiltered", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(valueCase.value)))
				for range b.N {
					benchmarkVerdict = fullScanPolicyVerdict(policy, valueCase.value)
				}
			})
		}
	}
}

func evaluationCustomPolicy(count int, expression string) Policy {
	rules := make([]CustomRule, count)
	for i := range rules {
		rules[i] = CustomRule{ID: fmt.Sprintf("evaluation-%02d", i), Regex: expression}
	}
	return Policy{CustomRules: rules}
}

var (
	benchmarkCompiledCatalog *compiledCatalog
	benchmarkCompiledPolicy  *CompiledPolicy
)

// Cold catalog construction is intentionally distinct from tenant policy
// compilation, which reuses the immutable catalog. Cap-exceeding expressions
// measure actual compilation and fallback costs, not just steady-state scans.
func BenchmarkDetectionCompilation(b *testing.B) {
	b.Run("cold-catalog", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			catalog, err := compileNativeCatalog()
			if err != nil {
				b.Fatal(err)
			}
			benchmarkCompiledCatalog = catalog
		}
	})
	_, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(b, err)
	cases := []struct {
		name string
		spec Policy
	}{
		{name: "warm-catalog"},
		{name: "custom-1", spec: evaluationCustomPolicy(1, `tenant_[A-Za-z0-9]{16,40}`)},
		{name: fmt.Sprintf("custom-%d", maxPolicyCustomRules), spec: evaluationCustomPolicy(maxPolicyCustomRules, `tenant_[A-Za-z0-9]{16,40}`)},
		{name: fmt.Sprintf("dfa-fallback-%d", maxPolicyCustomRules), spec: evaluationCustomPolicy(maxPolicyCustomRules, `[ab]*a[ab]{10}`)},
		{name: fmt.Sprintf("nfa-fallback-%d", maxPolicyCustomRules), spec: evaluationCustomPolicy(maxPolicyCustomRules, strings.Repeat(`[ab]{1000}`, 5))},
	}
	for _, test := range cases {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				policy, err := testPolicyCompiler.CompilePolicy(test.spec)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkCompiledPolicy = policy
			}
		})
	}
}
