package secrets

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// These seeds select synthetic payloads, tenant update order, invalid-policy
// variants and placements. Neither partition uses catalog regex witnesses.
const (
	syntheticDynamicDevelopmentSeed uint64 = 0x5344594e01
	syntheticDynamicHoldoutSeed     uint64 = 0x5344594e02
)

type syntheticDynamicProbe struct {
	policyVerdictProbe
	bank int
	rule int
}

type syntheticDynamicModel struct {
	bank  int
	count int
}

type syntheticDynamicStep struct {
	input     *Policy
	inherited bool
	model     syntheticDynamicModel
}

type syntheticDynamicReader struct {
	original      *CompiledPolicy
	originalBatch BatchDetector
	previous      *CompiledPolicy
	previousBatch BatchDetector
}

type syntheticDynamicTenant struct {
	source   *dynamicPolicySource
	provider *compiledPolicyProvider
	steps    []syntheticDynamicStep
	probes   []syntheticDynamicProbe
	combined string
	foreign  string
}

func syntheticDynamicRuleID(index int) string {
	return fmt.Sprintf("synthetic-dynamic-%02d", index)
}

func (m syntheticDynamicModel) combinedVerdict() Verdict {
	var findings []Match
	for index := range m.count {
		findings = append(findings, Match{RuleID: syntheticDynamicRuleID(index)})
	}
	return uniqueFindingVerdict(findings)
}

func (m syntheticDynamicModel) probeVerdict(probe syntheticDynamicProbe) Verdict {
	if probe.bank != m.bank || probe.rule >= m.count {
		return Verdict{}
	}
	var findings []Match
	for _, id := range probe.ruleIDs {
		findings = append(findings, Match{RuleID: id})
	}
	return uniqueFindingVerdict(findings)
}

// Each rule has an independently authored positive and negative, including
// rules near the maximum count. The special expressions deliberately exceed
// optimization budgets, not the accepted policy resource limits. Assertions
// concern consumer verdicts, never which optimization the compiler selected.
func newSyntheticDynamicTenant(rng *rand.Rand, invalidKinds []int) syntheticDynamicTenant {
	var tenant syntheticDynamicTenant
	var banks [4]Policy
	var positives [4][]string
	counts := [...]int{6, maxPolicyCustomRules, 3, 6}
	addProbe := func(bank, rule int, name, value string, positive bool) {
		probe := syntheticDynamicProbe{
			policyVerdictProbe: policyVerdictProbe{name: name, value: value},
			bank:               bank,
			rule:               rule,
		}
		if positive {
			probe.ruleIDs = []string{syntheticDynamicRuleID(rule)}
		}
		tenant.probes = append(tenant.probes, probe)
	}
	for bank, count := range counts {
		prefix := fmt.Sprintf("SDYN%016xB%dR", rng.Uint64(), bank)
		for index := range count {
			tag := fmt.Sprintf("%s%02d:", prefix, index)
			body := []byte("ABCDEFGHJKLMNPQR")
			rng.Shuffle(len(body), func(i, j int) { body[i], body[j] = body[j], body[i] })
			positive := tag + string(body) + "!"
			negative := tag + string(body[:15]) + "!"
			rule := CustomRule{ID: syntheticDynamicRuleID(index), Regex: tag + `[A-Z]{16}!`}
			switch {
			case index == 1:
				rule.Regex = `(?m)^` + tag + `(?i:[ks]{4})!$`
				positive = tag + "KſKſ!"
				negative = "x" + positive
				addProbe(bank, index, "unicode-line-context", "before\n"+positive+"\nafter", true)
				addProbe(bank, index, "unicode-wrong-rune-count", tag+"KſK!", false)
			case index == 2:
				rule.Regex = tag + `([A-Z]{16})!`
				lowEntropy := tag + strings.Repeat(string(body[0]), 16) + "!"
				addProbe(bank, index, "low-entropy-capture", lowEntropy, true)
			case bank == 1 && index == 3:
				rule.Regex = tag + `(?:long_[A-Z]{24}|ok_[0-9]{2})!`
				positive = tag + fmt.Sprintf("ok_%02d!", rng.IntN(100))
				negative = tag + "long_" + string(body) + "!"
				addProbe(bank, index, "long-alternative", tag+"long_"+string(body)+string(body[:8])+"!", true)
			case bank == 1 && index == 4:
				rule.Regex = `(?m)^` + tag + `[ab]*a[ab]{12}!$`
				positive = tag + strings.Repeat("b", rng.IntN(32)) + "a" + strings.Repeat("b", 12) + "!"
				negative = tag + strings.Repeat("b", 13) + "!"
				addProbe(bank, index, "dfa-long-near-miss", tag+strings.Repeat("b", 1536)+"a!", false)
			case bank == 1 && index == 5:
				rule.Regex = tag + `(?:qwertyui){600}!`
				positive = tag + strings.Repeat("qwertyui", 600) + "!"
				negative = tag + strings.Repeat("qwertyui", 599) + "!"
			}
			banks[bank].CustomRules = append(banks[bank].CustomRules, rule)
			positives[bank] = append(positives[bank], positive)
			addProbe(bank, index, "positive", positive, true)
			addProbe(bank, index, "negative", negative, false)
		}
	}

	// Most probes are shorter than 2 KiB. Only a small, explicit length set
	// carries padding; no large rule-count x length x placement product is kept.
	credential := positives[0][2]
	for _, size := range []int{64, 127, 128, 255, 256, 511, 512, 1023, 1024, 1535, 2047, 2048, 4096, 16384, 128 << 10} {
		padding := size - len(credential)
		before := []int{0, padding / 2, padding}[rng.IntN(3)]
		value := strings.Repeat("~", before) + credential + strings.Repeat("~", padding-before)
		addProbe(0, 2, fmt.Sprintf("padded-%d", size), value, true)
	}
	addProbe(0, 2, "keyword-flood", strings.Repeat(credential[:len(credential)-17]+"? ", 300)+credential, true)
	addProbe(0, 0, "invalid-utf8-context", "\xff"+positives[0][0]+"\xc0\xaf", true)
	for _, size := range []int{0, 1, 31, 1024, 2047, 128 << 10} {
		addProbe(-1, 0, fmt.Sprintf("benign-%d", size), strings.Repeat("~", size), false)
	}
	for _, bank := range positives {
		tenant.combined += strings.Join(bank, "\n") + "\n"
	}
	tenant.foreign = positives[0][0]

	policy := func(bank, count int) *Policy {
		return &Policy{CustomRules: append([]CustomRule(nil), banks[bank].CustomRules[:count]...)}
	}
	invalid := func(round int) *Policy {
		bad := policy(3, counts[3])
		last := &bad.CustomRules[len(bad.CustomRules)-1]
		switch invalidKinds[round] {
		case 0:
			last.Regex = "("
		case 1:
			last.ID = "invalid rule ID"
		case 2:
			last.ID = strings.Repeat("x", maxPolicyStringLength+1)
		case 3:
			last.Regex = strings.Repeat("x", maxCustomRegexLength+1)
		case 4:
			last.ID = bad.CustomRules[0].ID
		case 5:
			bad.CustomRules = append(bad.CustomRules, banks[1].CustomRules...)
		}
		return bad
	}
	tenant.steps = []syntheticDynamicStep{
		{input: policy(0, 3), inherited: true, model: syntheticDynamicModel{0, 3}},
		{input: policy(0, 6), model: syntheticDynamicModel{0, 6}},
		{input: policy(1, maxPolicyCustomRules), model: syntheticDynamicModel{1, maxPolicyCustomRules}},
		{input: invalid(0), model: syntheticDynamicModel{1, maxPolicyCustomRules}},
		{input: policy(1, 4), model: syntheticDynamicModel{1, 4}},
		{input: invalid(1), model: syntheticDynamicModel{1, 4}},
		{input: policy(2, 3), model: syntheticDynamicModel{2, 3}},
		{input: invalid(2), model: syntheticDynamicModel{2, 3}},
		{input: &Policy{}, model: syntheticDynamicModel{-1, 0}},
		{input: policy(0, 6), inherited: true, model: syntheticDynamicModel{0, 6}},
	}
	tenant.source = &dynamicPolicySource{}
	tenant.provider, _ = newDynamicTestProvider(tenant.source)
	return tenant
}

// Only a static path label escapes on failure, never input values, expressions,
// policy objects, tenant-provided names, or hashes of generated values.
func syntheticDynamicCheck(policy *CompiledPolicy, batch *BatchDetector, value string, want Verdict) string {
	if !slices.Equal(want.Matches, policy.Detect(value).Matches) {
		return "direct"
	}
	if !slices.Equal(want.Matches, batch.Detect(value).Matches) {
		return "batch-first"
	}
	if !slices.Equal(want.Matches, batch.Detect(value).Matches) {
		return "batch-repeat"
	}
	return ""
}

func TestDynamicPolicySyntheticChurn(t *testing.T) {
	// This model exercises tenant-owned custom rules and update isolation.
	// Native catalog behavior is covered by the catalog and mixed-policy tests.
	compiler, err := NewPolicyCompiler(&[]string{})
	if err != nil {
		t.Fatal(err)
	}
	for _, partition := range []struct {
		name string
		seed uint64
	}{
		{"development", syntheticDynamicDevelopmentSeed},
		{"holdout", syntheticDynamicHoldoutSeed},
	} {
		t.Run(partition.name, func(t *testing.T) {
			t.Logf("synthetic dynamic partition=%s seed=%d", partition.name, partition.seed)
			rng := rand.New(rand.NewPCG(partition.seed, partition.seed^0x9e3779b97f4a7c15))
			invalidKinds := rng.Perm(6)
			// awaitDynamic bounds each lifecycle wait. Slow synchronous reference
			// scans must not cancel already completed updates under -race.
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			const tenantCount, readerCount = 3, 2
			admission := make(chan struct{}, 2)
			var tenants [tenantCount]syntheticDynamicTenant
			var readers [tenantCount][readerCount]syntheticDynamicReader
			var checkedOracles [tenantCount]map[syntheticDynamicModel]bool
			for index := range tenants {
				// Across the first two tenants every rejection variant is
				// exercised. Rotate from accepted states so tenants concurrently
				// add, reject, remove and recover at different lifecycle stages.
				kinds := append(append([]int(nil), invalidKinds[index*3%6:]...), invalidKinds[:index*3%6]...)
				tenants[index] = newSyntheticDynamicTenant(rng, kinds)
				checkedOracles[index] = make(map[syntheticDynamicModel]bool)
				offset := [...]int{0, 2, 6}[index]
				steps := tenants[index].steps
				tenants[index].steps = append(append([]syntheticDynamicStep(nil), steps[offset:]...), steps[:offset]...)
				tenants[index].provider.tenant = fmt.Sprintf("synthetic-tenant-%d", index)
				tenants[index].provider.admission = admission
				tenants[index].provider.compile = compiler.CompilePolicy
			}
			for index := range tenants {
				tenant := &tenants[index]
				for other := range tenants {
					if other != index {
						tenant.combined += tenants[other].foreign + "\n"
						tenant.probes = append(tenant.probes, syntheticDynamicProbe{
							policyVerdictProbe: policyVerdictProbe{name: "foreign-tenant", value: tenants[other].foreign},
							bank:               -1,
						})
					}
				}
				initial := tenant.steps[0]
				tenant.source.set(initial.input, initial.inherited)
				compiled, ok := tenant.provider.current(ctx)
				if !ok || ctx.Err() != nil {
					t.Fatalf("tenant %d initial policy unavailable", index)
				}
				for reader := range readers[index] {
					readers[index][reader] = syntheticDynamicReader{
						original: compiled, originalBatch: compiled.NewBatchDetector(),
						previous: compiled, previousBatch: compiled.NewBatchDetector(),
					}
				}
			}

			for phase := 1; phase < len(tenants[0].steps); phase++ {
				var admissionWaits [tenantCount]<-chan struct{}
				var release [tenantCount]chan struct{}
				var updates [tenantCount]<-chan dynamicPolicyResult
				for _, index := range rng.Perm(tenantCount) {
					tenant := &tenants[index]
					step := tenant.steps[phase]
					release[index] = make(chan struct{})
					tenant.provider.compile = func(input Policy) (*CompiledPolicy, error) {
						select {
						case <-release[index]:
							return compiler.CompilePolicy(input)
						case <-ctx.Done():
							return nil, ctx.Err()
						}
					}
					tenant.source.set(step.input, step.inherited)
					admissionContext := &dynamicWaitContext{Context: ctx, at: 2, waiting: make(chan struct{})}
					admissionWaits[index] = admissionContext.waiting
					updates[index] = refreshDynamicPolicy(admissionContext, tenant.provider)
				}
				for _, barrier := range admissionWaits {
					awaitDynamic(t, barrier)
				}

				// All updates hold their provider gate; one also waits for shared
				// compilation admission. Readers inspect retained snapshots first.
				// Sources stay fixed until every call has completed, so expectations
				// do not depend on scheduler timing or publication order.
				var waiting []<-chan struct{}
				var results []<-chan string
				for index := range tenants {
					tenant := &tenants[index]
					step := tenant.steps[phase]
					value := tenant.combined + strings.Repeat("~", phase)
					for readerIndex := range readers[index] {
						reader := &readers[index][readerIndex]
						waitContext := &dynamicWaitContext{Context: ctx, at: 1, waiting: make(chan struct{})}
						result := make(chan string, 1)
						waiting = append(waiting, waitContext.waiting)
						results = append(results, result)
						go func() {
							failure := ""
							if path := syntheticDynamicCheck(reader.original, &reader.originalBatch, value, tenant.steps[0].model.combinedVerdict()); path != "" {
								failure = "retained-original-" + path
							}
							if path := syntheticDynamicCheck(reader.previous, &reader.previousBatch, value, tenant.steps[phase-1].model.combinedVerdict()); path != "" && failure == "" {
								failure = "retained-previous-" + path
							}
							compiled, ok := tenant.provider.current(waitContext)
							if !ok || ctx.Err() != nil {
								result <- "current-policy-unavailable"
								return
							}
							batch := compiled.NewBatchDetector()
							if path := syntheticDynamicCheck(compiled, &batch, value, step.model.combinedVerdict()); path != "" && failure == "" {
								failure = "current-" + path
							}
							reader.previous, reader.previousBatch = compiled, batch
							result <- failure
						}()
					}
				}
				for _, barrier := range waiting {
					awaitDynamic(t, barrier)
				}
				for _, index := range rng.Perm(tenantCount) {
					close(release[index])
				}
				for reader, pending := range results {
					if failure := awaitDynamic(t, pending); failure != "" {
						t.Fatalf("phase %d reader %d: %s mismatch", phase, reader, failure)
					}
				}
				for index, pending := range updates {
					result := awaitDynamic(t, pending)
					tenant := &tenants[index]
					step := tenant.steps[phase]
					if !result.ok || ctx.Err() != nil {
						t.Fatalf("phase %d tenant %d: updated policy unavailable", phase, index)
					}
					batch := result.policy.NewBatchDetector()
					for probeIndex, probe := range tenant.probes {
						// Check the slow oracle once per policy model, not once
						// per concurrent reader or repeated rejected update.
						// Public direct/batch verdicts still check every probe
						// after every transition, including removed rules.
						if !checkedOracles[index][step.model] && (probe.bank == -1 ||
							(probe.bank == step.model.bank && probe.rule < step.model.count)) &&
							!slices.Equal(step.model.probeVerdict(probe).Matches, fullScanPolicyVerdict(result.policy, probe.value).Matches) {
							t.Fatalf("phase %d tenant %d probe %d: unfiltered mismatch", phase, index, probeIndex)
						}
						if path := syntheticDynamicCheck(result.policy, &batch, probe.value, step.model.probeVerdict(probe)); path != "" {
							t.Fatalf("phase %d tenant %d probe %d (%s): %s mismatch", phase, index, probeIndex, probe.name, path)
						}
					}
					checkedOracles[index][step.model] = true
				}
			}
		})
	}
}
