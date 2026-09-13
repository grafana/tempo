package secrets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	holdoutGitHubPAT      = "github-pat"
	holdoutGitHubApp      = "github-app-token"
	holdoutPyPI           = "pypi-upload-token"
	holdoutConfluent      = "confluent-secret-key"
	holdoutAge            = "age-secret-key"
	holdoutJWT            = "jwt"
	holdoutPrivateKey     = "private-key"
	holdoutBedrock        = "aws-amazon-bedrock-api-key-short-lived"
	holdoutGrafana        = "grafana-service-account-token"
	holdoutGitLabRoutable = "gitlab-pat-routable"
)

// These cases are independent of regex witness generation and optimization plan
// fields. Expected IDs include every occurrence, in catalog order; the public
// contract is their ordered unique projection. Failure messages never print values.
func assertOptimizerHoldout(t testing.TB, policy *CompiledPolicy, value string, ids []string) {
	t.Helper()
	var want []Match
	for _, id := range ids {
		want = append(want, Match{RuleID: id})
	}
	require.Equal(t, want, catalogFindings(policy.catalog, value, false), "unfiltered finding sequence")
	require.Equal(t, want, catalogFindings(policy.catalog, value, true), "optimized finding sequence")
	verdict := uniqueFindingVerdict(want)
	require.Equal(t, verdict, policy.Detect(value), "direct verdict")
	batch := policy.NewBatchDetector()
	require.Equal(t, verdict, batch.Detect(value), "batch miss verdict")
	require.Equal(t, verdict, batch.Detect(value), "batch hit verdict")
}

func TestOptimizerHoldoutStrictRunBoundaries(t *testing.T) {
	// The same credential appears in anchored, bounded-context and unbounded-
	// context expressions. Tests deliberately do not prescribe their chosen plans.
	catalog := compileTestCatalog(
		t,
		catalogRuleSpec{ID: "z-anchored", Regex: `\bHLD:([A-Za-z0-9]{16})!`, Keywords: []string{"hld:"}, SecretGroup: 1},
		catalogRuleSpec{ID: "a-bounded", Regex: `[A-Z]{2}/HLD:([A-Za-z0-9]{16})!`, Keywords: []string{"hld:"}, SecretGroup: 1},
		catalogRuleSpec{ID: "m-unbounded", Regex: `[A-Z]+/HLD:([A-Za-z0-9]{16})!`, Keywords: []string{"hld:"}, SecretGroup: 1},
	)
	policy := &CompiledPolicy{catalog: catalog}
	const body = "A1b2C3d4E5f6G7h8"
	const credential = "AB/HLD:" + body + "!"
	one := []string{"z-anchored", "a-bounded", "m-unbounded"}
	cases := []struct {
		name  string
		value string
		ids   []string
	}{
		{"one-short", "AB/HLD:" + body[:15] + "!", nil},
		{"exact", credential, one},
		{"one-long", "AB/HLD:" + body + "9!", nil},
		{"punctuation-splits-run", "AB/HLD:" + body[:8] + "." + body[8:] + "!", nil},
		{"two-byte-rune-splits-run", "AB/HLD:" + body[:8] + "é" + body[8:] + "!", nil},
		{"four-byte-rune-replaces-byte", "AB/HLD:" + body[:7] + "\U00010400" + body[8:] + "!", nil},
		{"invalid-byte-splits-run", "AB/HLD:" + body[:8] + "\xff" + body[8:] + "!", nil},
		{"strict-class-rejects-kelvin", "AB/HLD:" + body[:15] + "K!", nil},
		{"strict-class-rejects-long-s", "AB/HLD:" + body[:15] + "ſ!", nil},
		{"unrelated-long-run-does-not-match", strings.Repeat("x", 64) + " AB/HLD:" + body[:15] + "!", nil},
		{"unicode-around-exact-run", "界é/" + credential + "/λ中", one},
		{"invalid-utf8-around-exact-run", "\xff\xc0\xaf/" + credential + "/\xed\xa0\x80", one},
		{"rejected-first-valid-later", "AB/HLD:" + body[:15] + "! " + credential, one},
		{"adjacent-valid-occurrences", credential + credential, []string{"z-anchored", "z-anchored", "a-bounded", "a-bounded", "m-unbounded", "m-unbounded"}},
		{"keyword-flood-then-exact", strings.Repeat("HLD:? ", 300) + credential, one},
		{"unicode-keyword-flood-then-exact", strings.Repeat("HLD:é ", 250) + credential, one},
	}
	// Place the mandatory body across common byte/chunk boundaries, including
	// values just below 2 KiB. These offsets are not read from implementation constants.
	for _, boundary := range []int{32, 64, 128, 256, 512, 1024} {
		cases = append(cases, struct {
			name  string
			value string
			ids   []string
		}{fmt.Sprintf("body-crosses-%d", boundary), strings.Repeat("~", boundary-9) + credential, one})
	}
	cases = append(cases, struct {
		name  string
		value string
		ids   []string
	}{"exact-at-2047-bytes", strings.Repeat("~", 2047-len(credential)) + credential, one})
	// Measurement sizes must never become a runtime truncation boundary.
	cases = append(cases, struct {
		name  string
		value string
		ids   []string
	}{"valid-after-128KiB", strings.Repeat("~", 128<<10) + credential, one})
	for _, probe := range cases {
		t.Run(probe.name, func(t *testing.T) {
			assertOptimizerHoldout(t, policy, probe.value, probe.ids)
		})
	}
}

func TestOptimizerHoldoutAlternationOptionalAndUnicode(t *testing.T) {
	catalog := compileTestCatalog(
		t,
		catalogRuleSpec{ID: "branch", Regex: `\b(?:long_[A-Za-z0-9]{24}|ok_[0-9]{2})!`, Keywords: []string{"long_"}},
		catalogRuleSpec{ID: "optional", Regex: `\b(?:pad_[A-Z]{24};)?done_[0-9]{2}!`, Keywords: []string{"pad_"}},
		catalogRuleSpec{ID: "repeat", Regex: `\b(?:pad_[A-Z]{24};){0,2}stop_[0-9]{2}!`, Keywords: []string{"pad_"}},
		catalogRuleSpec{ID: "folded-class", Regex: `\bfold_((?i:[a-z]{8}))!`, Keywords: []string{"fold_"}, SecretGroup: 1},
		catalogRuleSpec{ID: "mixed-scope", Regex: `\bscope_(?i:[ks]{8})[A-Z0-9]{12}!`, Keywords: []string{"scope_"}},
		catalogRuleSpec{ID: "replacement-rune", Regex: `\bbad_\x{FFFD}[A-F0-9]{12}!`, Keywords: []string{"bad_"}},
		catalogRuleSpec{ID: "punctuated-run", Regex: `\bpunct_([A-Za-z0-9_/-]{16})!`, Keywords: []string{"punct_"}, SecretGroup: 1},
		catalogRuleSpec{ID: "entropy", Regex: `\bentropy_([A-Za-z0-9]{16})!`, Keywords: []string{"entropy_"}, SecretGroup: 1, Entropy: 2.5},
	)
	policy := &CompiledPolicy{catalog: catalog}
	const pad = "pad_ABCDEFGHIJKLMNOPQRSTUVWX;"
	const high = "entropy_A1b2C3d4E5f6G7h8!"
	const low = "entropy_aaaaaaaaaaaaaaaa!"
	for _, probe := range []struct {
		name  string
		value string
		ids   []string
	}{
		{"short-alternate-without-long-run", "ok_42!", []string{"branch"}},
		{"long-alternate-exact", "long_Ab12Cd34Ef56Gh78Ij90Kl12!", []string{"branch"}},
		{"short-alternate-after-failed-long", "long_Ab12! ok_42!", []string{"branch"}},
		{"short-alternate-after-unicode", "界/ok_42!", []string{"branch"}},
		{"both-alternates-in-one-value", "ok_42! long_Ab12Cd34Ef56Gh78Ij90Kl12!", []string{"branch", "branch"}},
		{"optional-long-run-absent", "done_42!", []string{"optional"}},
		{"optional-long-run-present", pad + "done_42!", []string{"optional"}},
		{"optional-long-run-separated-by-unicode", "é" + pad + "done_42!", []string{"optional"}},
		{"zero-optional-repeats", "stop_42!", []string{"repeat"}},
		{"two-optional-repeats", pad + pad + "stop_42!", []string{"repeat"}},
		{"folded-ascii-exact", "fold_kSkSkSkS!", []string{"folded-class"}},
		{"kelvin-class-aliases", "fold_KKKKKKKK!", []string{"folded-class"}},
		{"long-s-class-aliases", "fold_ſſſſſſſſ!", []string{"folded-class"}},
		{"mixed-class-aliases", "fold_kKsſKKSſ!", []string{"folded-class"}},
		{"alias-byte-length-is-not-rune-length", "fold_KſKſKſK!", nil},
		{"folded-class-does-not-accept-arbitrary-unicode", "fold_KſKſKſKé!", nil},
		{"folded-class-invalid-byte", "fold_kSkS\xffSkS!", nil},
		{"strict-run-after-unicode-folds", "scope_KſKſKſKſAB12CD34EF56!", []string{"mixed-scope"}},
		{"strict-run-after-folds-one-short", "scope_KſKſKſKſAB12CD34EF5!", nil},
		{"strict-scope-must-not-fold", "scope_KſKſKſKſAB12CD34EF5ſ!", nil},
		{"invalid-utf8-matches-replacement-rune", "bad_\xffAB12CD34EF56!", []string{"replacement-rune"}},
		{"literal-replacement-rune", "bad_�AB12CD34EF56!", []string{"replacement-rune"}},
		{"two-invalid-bytes-are-two-runes", "bad_\xc0\xafAB12CD34EF56!", nil},
		{"punctuation-is-part-of-required-class", "punct_A1_b2/C3-d4_E5/f!", []string{"punctuated-run"}},
		{"punctuation-class-rejects-dot", "punct_A1_b2.C3-d4_E5/f!", nil},
		{"punctuation-class-rejects-multibyte", "punct_A1_b2éC3-d4_E5/f!", nil},
		{"low-entropy-only", low, nil},
		{"rejected-entropy-candidate-then-valid", low + " " + high, []string{"entropy"}},
		{"rejected-between-valid-candidates", high + " " + low + " " + high, []string{"entropy", "entropy"}},
		{"keyword-flood-then-short-branch", strings.Repeat("long_? ", 270) + "ok_42!", []string{"branch"}},
		{"catalog-order-not-position-or-name", high + " fold_KſKſKſKſ! ok_42! " + high, []string{"branch", "folded-class", "entropy", "entropy"}},
	} {
		t.Run(probe.name, func(t *testing.T) {
			assertOptimizerHoldout(t, policy, probe.value, probe.ids)
		})
	}
}

func TestOptimizerHoldoutStructuralCredentials(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	policy := &CompiledPolicy{catalog: catalog}
	fixtures := nativeCatalogFixtures(t)
	// Fixed indices select independently serialized fixtures, not regex-generated
	// witnesses: checksums, routing frames, JSON, DER, JWT and macaroon framing.
	for _, selected := range []struct {
		id       string
		positive int
		negative int
	}{
		{holdoutGrafana, 0, 0},
		{holdoutConfluent, 0, 0},
		{holdoutGitLabRoutable, 0, 4},
		{holdoutAge, 0, 1},
		{holdoutJWT, 0, 6},
		{holdoutGitHubApp, 2, 1},
		{holdoutPyPI, 3, 5},
		{holdoutPrivateKey, 1, 0},
		{holdoutBedrock, 0, 4},
	} {
		t.Run(selected.id, func(t *testing.T) {
			fixture := fixtures[selected.id]
			positive := fixture.Positive[selected.positive]
			negative := fixture.Negative[selected.negative]
			t.Run("malformed", func(t *testing.T) {
				assertOptimizerHoldout(t, policy, negative, nil)
			})
			t.Run("valid-surrounded-by-unicode", func(t *testing.T) {
				assertOptimizerHoldout(t, policy, "界\n"+positive+"\né", []string{selected.id})
			})
			t.Run("rejected-first-then-two-valid", func(t *testing.T) {
				value := "\xff\n" + negative + "\n" + positive + "\n" + positive
				assertOptimizerHoldout(t, policy, value, []string{selected.id, selected.id})
			})
		})
	}
}

type optimizerHoldoutWorkload struct {
	name  string
	value string
	ids   []string
}

// Fill only complete UTF-8 motifs and use ASCII spaces for the remainder. Input
// sizes are exact bytes without accidentally turning the Unicode cases invalid.
func optimizerHoldoutFill(motif string, size int) string {
	return strings.Repeat(motif, size/len(motif)) + strings.Repeat(" ", size%len(motif))
}

func optimizerHoldoutWorkloads(t testing.TB) []optimizerHoldoutWorkload {
	t.Helper()
	fixtures := nativeCatalogFixtures(t)
	var cases []optimizerHoldoutWorkload
	addNoise := func(name, motif string, sizes ...int) {
		for _, size := range sizes {
			cases = append(cases, optimizerHoldoutWorkload{
				name: name + fmt.Sprintf("/%dB", size), value: optimizerHoldoutFill(motif, size),
			})
		}
	}
	addNoise("alternating-ascii-unicode", "r7é.p2界/", 8, 24, 64, 192, 768, 1536, 32768)
	addNoise("near-complete-github-starts", "ghp_"+strings.Repeat("Ab7d", 8)+"Ab7! ", 48, 128, 384, 1024, 2048, 131072)
	addNoise("multiple-provider-prefixes", "ghp_! glsa_! pypi-! xoxb-! Bearer ! glpat-! ", 96, 256, 512, 1792, 32768)
	addNoise("non-ascii-only", "éλ", 16, 32, 128, 512, 1984, 131072)
	addNoise("invalid-utf8-and-false-starts", "\xffghp_?\xc0\xafglsa_?\xed\xa0\x80pypi-? ", 1024)

	// Positive placement and provider choice vary together deliberately, rather
	// than multiplying every family by every size and offset.
	for _, placement := range []struct {
		id       string
		fixture  int
		size     int
		position string
	}{
		{holdoutGitHubPAT, 0, 64, "early"},
		{holdoutGrafana, 0, 192, "middle"},
		{holdoutPyPI, 0, 256, "end"},
		{holdoutJWT, 0, 384, "early"},
		{holdoutConfluent, 0, 768, "middle"},
		{holdoutGitLabRoutable, 0, 1024, "end"},
		{holdoutAge, 0, 1536, "early"},
		{holdoutGitHubApp, 2, 2048, "middle"},
		{holdoutPyPI, 3, 1920, "end"},
		{holdoutPrivateKey, 1, 32768, "end"},
		{holdoutBedrock, 0, 131072, "middle"},
	} {
		positive := fixtures[placement.id].Positive[placement.fixture]
		space := placement.size - len(positive) - 2
		require.GreaterOrEqual(t, space, 0, "fixture exceeds held-out workload size")
		offset := 0
		switch placement.position {
		case "middle":
			offset = space / 2
		case "end":
			offset = space
		}
		const request = "GET /v2/items?id=42; peer=ab7d; region=eu-west-1; status=204\n"
		value := optimizerHoldoutFill(request, offset) + "\n" + positive + "\n" + optimizerHoldoutFill(request, space-offset)
		cases = append(cases, optimizerHoldoutWorkload{
			name:  fmt.Sprintf("positive-%s-%s/%dB", placement.position, placement.id, placement.size),
			value: value, ids: []string{placement.id},
		})
	}

	// Encounter order differs from catalog order and the repeated GitHub token
	// must produce two full findings but only one public rule ID.
	multiple := strings.Join([]string{
		fixtures[holdoutPyPI].Positive[0],
		fixtures[holdoutGitHubPAT].Positive[0],
		fixtures[holdoutGrafana].Positive[0],
		fixtures[holdoutConfluent].Positive[0],
		fixtures[holdoutGitHubPAT].Positive[0],
	}, "\n")
	for _, size := range []int{512, 1536} {
		cases = append(cases, optimizerHoldoutWorkload{
			name:  fmt.Sprintf("several-valid-credentials/%dB", size),
			value: multiple + "\n" + optimizerHoldoutFill("status=204 peer=é; ", size-len(multiple)-1),
			ids:   []string{holdoutConfluent, holdoutGitHubPAT, holdoutGitHubPAT, holdoutGrafana, holdoutPyPI},
		})
	}
	flood := strings.Repeat("ghp_? ", 300) + fixtures[holdoutGitHubPAT].Positive[0]
	cases = append(cases, optimizerHoldoutWorkload{
		name: "keyword-flood-then-short-valid/1920B", value: flood + "\n" + strings.Repeat(" ", 1920-len(flood)-1), ids: []string{holdoutGitHubPAT},
	})
	return cases
}

func TestOptimizerHoldoutWorkloads(t *testing.T) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(t, err)
	policy := &CompiledPolicy{catalog: catalog}
	for _, workload := range optimizerHoldoutWorkloads(t) {
		t.Run(workload.name, func(t *testing.T) {
			t.Parallel()
			assertOptimizerHoldout(t, policy, workload.value, workload.ids)
		})
	}
}

// BenchmarkOptimizerHoldoutWorkloads measures uncached full-catalog detection and
// the unfiltered oracle over held-out shapes, not the default production workload.
// Compilation and fixture setup are outside each timed workload.
func BenchmarkOptimizerHoldoutWorkloads(b *testing.B) {
	catalog, err := testPolicyCompiler.catalog()
	require.NoError(b, err)
	policy := &CompiledPolicy{catalog: catalog}
	for _, workload := range optimizerHoldoutWorkloads(b) {
		b.Run(workload.name, func(b *testing.B) {
			assertOptimizerHoldout(b, policy, workload.value, workload.ids)
			b.Run("optimized", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(workload.value)))
				for range b.N {
					benchmarkVerdict = policy.Detect(workload.value)
				}
			})
			b.Run("unfiltered", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(workload.value)))
				for range b.N {
					benchmarkVerdict = fullScanPolicyVerdict(policy, workload.value)
				}
			})
		})
	}
}
