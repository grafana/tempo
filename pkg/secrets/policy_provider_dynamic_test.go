package secrets

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dynamicPolicyInput struct {
	policy    *Policy
	inherited bool
}

type dynamicPolicySource struct {
	input atomic.Pointer[dynamicPolicyInput]
}

func (s *dynamicPolicySource) set(policy *Policy, inherited bool) {
	s.input.Store(&dynamicPolicyInput{policy: policy, inherited: inherited})
}

func (s *dynamicPolicySource) get(string) (*Policy, bool) {
	input := s.input.Load()
	if input == nil {
		return nil, true
	}
	return input.policy, input.inherited
}

func dynamicPolicy(regex string) *Policy {
	return &Policy{CustomRules: []CustomRule{{ID: "dynamic-token", Regex: regex}}}
}

func newDynamicTestProvider(source *dynamicPolicySource) (*compiledPolicyProvider, *prometheus.Registry) {
	registry := prometheus.NewRegistry()
	return &compiledPolicyProvider{
		tenant:    "dynamic-tenant",
		policy:    source.get,
		logger:    log.NewNopLogger(),
		updating:  make(chan struct{}, 1),
		compile:   testPolicyCompiler.CompilePolicy,
		admission: make(chan struct{}, 2),
		metrics:   newPolicyUpdateMetrics(registry),
	}, registry
}

type dynamicPolicyResult struct {
	policy *CompiledPolicy
	ok     bool
}

func refreshDynamicPolicy(ctx context.Context, p *compiledPolicyProvider) <-chan dynamicPolicyResult {
	result := make(chan dynamicPolicyResult, 1)
	go func() {
		policy, ok := p.current(ctx)
		result <- dynamicPolicyResult{policy: policy, ok: ok}
	}()
	return result
}

func awaitDynamic[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(10 * time.Second):
		t.Fatal("policy lifecycle did not complete")
		var zero T
		return zero
	}
}

// Done is evaluated when entering each cancellable select: first the provider
// gate, then process-wide admission. This observes waits without timing sleeps.
type dynamicWaitContext struct {
	context.Context
	calls   atomic.Int32
	at      int32
	waiting chan struct{}
}

func (c *dynamicWaitContext) Done() <-chan struct{} {
	if c.calls.Add(1) == c.at {
		close(c.waiting)
	}
	return c.Context.Done()
}

func TestDynamicPolicyWholeReplacementAndInheritance(t *testing.T) {
	source := &dynamicPolicySource{}
	provider, _ := newDynamicTestProvider(source)
	inherited := dynamicPolicy("DYNAMIC-INHERITED")
	withExtra := dynamicPolicy("DYNAMIC-TWO")
	withExtra.CustomRules = append(withExtra.CustomRules, CustomRule{ID: "extra-token", Regex: "DYNAMIC-EXTRA"})
	probes := []string{"DYNAMIC-INHERITED", "DYNAMIC-ONE", "DYNAMIC-TWO", "DYNAMIC-EXTRA"}
	for _, step := range []struct {
		name      string
		policy    *Policy
		inherited bool
		matches   []string
	}{
		{"first inherited policy", inherited, true, []string{"DYNAMIC-INHERITED"}},
		{"tenant replaces inherited", dynamicPolicy("DYNAMIC-ONE"), false, []string{"DYNAMIC-ONE"}},
		{"same ID expression edit", dynamicPolicy("DYNAMIC-TWO"), false, []string{"DYNAMIC-TWO"}},
		{"add rule", withExtra, false, []string{"DYNAMIC-TWO", "DYNAMIC-EXTRA"}},
		{"delete rule", dynamicPolicy("DYNAMIC-TWO"), false, []string{"DYNAMIC-TWO"}},
		{"explicit empty overrides inherited", &Policy{CustomRules: []CustomRule{}}, false, nil},
		{"equivalent explicit empty", &Policy{}, false, nil},
		{"overlay removal inherits again", inherited, true, []string{"DYNAMIC-INHERITED"}},
		{"no effective override", nil, true, nil},
		{"nil and empty are equivalent", &Policy{}, false, nil},
	} {
		t.Run(step.name, func(t *testing.T) {
			source.set(step.policy, step.inherited)
			compiled, ok := provider.current(context.Background())
			require.True(t, ok)
			for _, probe := range probes {
				assert.Equal(t, slices.Contains(step.matches, probe), compiled.Detect(probe).Matched(), probe)
			}
		})
	}
}

func TestDynamicDisabledRulesReplaceAndPreserveSnapshots(t *testing.T) {
	source := &dynamicPolicySource{}
	provider, _ := newDynamicTestProvider(source)
	const value = `api_key="r9Q2m7V4x1Z8c6B3n0H5j2L9p4T7w8Y1"`
	specific := nativeCatalogFixtures(t)["github-pat"].Positive[0]
	type snapshot struct {
		policy   *CompiledPolicy
		batch    BatchDetector
		want     Verdict
		specific Verdict
	}
	var snapshots []snapshot
	for _, step := range []struct {
		name            string
		input           *Policy
		inherited       bool
		enabled         bool
		specificEnabled bool
	}{
		{"defaults", nil, true, true, true},
		{"disable", &Policy{DisabledRules: []string{"generic-api-key"}}, false, false, true},
		{"replace", &Policy{DisabledRules: []string{"github-pat"}}, false, true, false},
		{"restore", &Policy{DisabledRules: []string{}}, false, true, true},
		{"empty-equivalent", &Policy{}, false, true, true},
		{"restore-inherited", &Policy{DisabledRules: []string{"generic-api-key"}}, true, false, true},
		{"override-removes-inherited", &Policy{}, false, true, true},
	} {
		t.Run(step.name, func(t *testing.T) {
			source.set(step.input, step.inherited)
			compiled, ok := provider.current(context.Background())
			require.True(t, ok)
			want := Verdict{}
			if step.enabled {
				want.Matches = []Match{{RuleID: "generic-api-key"}}
			}
			wantSpecific := Verdict{}
			if step.specificEnabled {
				wantSpecific.Matches = []Match{{RuleID: "github-pat"}}
			}
			snapshots = append(snapshots, snapshot{policy: compiled, batch: compiled.NewBatchDetector(), want: want, specific: wantSpecific})
			// Previously handed-out policies and both cached and uncached batch
			// decisions must remain unchanged after every publication.
			for i := range snapshots {
				retained := &snapshots[i]
				require.Equal(t, retained.want, retained.policy.Detect(value))
				require.Equal(t, retained.want, retained.batch.Detect(value))
				require.Equal(t, retained.want, retained.batch.Detect(value))
				require.Equal(t, retained.specific, retained.policy.Detect(specific))
				require.Equal(t, retained.specific, retained.batch.Detect(specific))
			}
		})
	}
}

func TestDynamicInvalidDisabledRulesFailOpenAndRetainLastGood(t *testing.T) {
	const private = "tenant-private-exclusion"
	const value = `api_key="r9Q2m7V4x1Z8c6B3n0H5j2L9p4T7w8Y1"`
	for _, invalid := range []struct {
		name string
		ids  []string
	}{
		{"unknown", []string{private}},
		{"duplicate", []string{"generic-api-key", "generic-api-key"}},
	} {
		for _, stage := range []string{"initial", "replacement"} {
			t.Run(invalid.name+"-"+stage, func(t *testing.T) {
				source := &dynamicPolicySource{}
				provider, _ := newDynamicTestProvider(source)
				var logs bytes.Buffer
				provider.logger = log.NewLogfmtLogger(&logs)
				expected, err := testPolicyCompiler.CompilePolicy(Policy{})
				require.NoError(t, err)
				want := Verdict{Matches: []Match{{RuleID: "generic-api-key"}}}
				input := &Policy{DisabledRules: []string{"generic-api-key"}}
				if stage == "replacement" {
					source.set(input, false)
					var ok bool
					expected, ok = provider.current(context.Background())
					require.True(t, ok)
					want = Verdict{}
				}
				// Reuse the source's slice storage: the published snapshot and
				// accepted-input comparison must not retain mutable caller memory.
				input.DisabledRules = append(input.DisabledRules[:0], invalid.ids...)
				source.set(input, false)
				for range 3 {
					compiled, ok := provider.current(context.Background())
					require.True(t, ok)
					require.Same(t, expected, compiled)
					require.Equal(t, want, compiled.Detect(value))
					batch := compiled.NewBatchDetector()
					require.Equal(t, want, batch.Detect(value))
				}
				require.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.rejected))
				require.NotContains(t, logs.String(), private)
				require.NotContains(t, logs.String(), "generic-api-key")
				if stage == "replacement" {
					source.set(&Policy{}, false)
					want.Matches = []Match{{RuleID: "generic-api-key"}}
				} else {
					source.set(&Policy{DisabledRules: []string{"generic-api-key"}}, false)
					want = Verdict{}
				}
				recovered, ok := provider.current(context.Background())
				require.True(t, ok)
				require.Equal(t, want, recovered.Detect(value))
			})
		}
	}
}

func TestDynamicPolicyRejectsInvalidRegexRetainsLastGood(t *testing.T) {
	for _, stage := range []string{"initial", "replacement"} {
		t.Run(stage, func(t *testing.T) {
			source := &dynamicPolicySource{}
			provider, _ := newDynamicTestProvider(source)
			var expected Verdict
			if stage == "replacement" {
				source.set(dynamicPolicy("DYNAMIC-ACCEPTED"), false)
				compiled, ok := provider.current(context.Background())
				require.True(t, ok)
				expected = compiled.Detect("DYNAMIC-ACCEPTED")
				require.True(t, expected.Matched())
			}
			source.set(&Policy{CustomRules: []CustomRule{{
				ID: "dynamic-token", Regex: "(DYNAMIC-REJECTED",
			}}}, false)
			for range 2 {
				compiled, ok := provider.current(context.Background())
				require.True(t, ok)
				require.Equal(t, expected, compiled.Detect("DYNAMIC-ACCEPTED"))
				require.False(t, compiled.Detect("DYNAMIC-REJECTED").Matched(), "invalid regex must never publish")
				batch := compiled.NewBatchDetector()
				require.Equal(t, expected, batch.Detect("DYNAMIC-ACCEPTED"))
				require.False(t, batch.Detect("DYNAMIC-REJECTED").Matched())
				if stage == "initial" {
					require.True(t, hasRuleFinding(compiled.Detect("sk_test_"+"0123456789abcdefghijklmn").Matches, "stripe-access-token"), "initial rejection must retain native baseline")
				}
			}
			require.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.rejected))
			source.set(&Policy{CustomRules: []CustomRule{{
				ID: "dynamic-token", Regex: "(DYNAMIC-REJECTED)",
			}}}, false)
			recovered, ok := provider.current(context.Background())
			require.True(t, ok)
			require.Equal(t, []Match{{RuleID: "dynamic-token"}}, recovered.Detect("DYNAMIC-REJECTED").Matches)
			require.False(t, recovered.Detect("DYNAMIC-ACCEPTED").Matched(), "valid replacement must replace the last good policy")
		})
	}
}

func TestDynamicPolicyRejectsOnceRetainsSnapshotAndRecovers(t *testing.T) {
	source := &dynamicPolicySource{}
	source.set(dynamicPolicy("DYNAMIC-ONE"), false)
	provider, registry := newDynamicTestProvider(source)
	var logs bytes.Buffer
	provider.logger = log.NewLogfmtLogger(&logs)
	const unsafeError = "synthetic-credential-must-not-escape"
	compilations := 0
	provider.compile = func(policy Policy) (*CompiledPolicy, error) {
		compilations++
		if policy.CustomRules[0].Regex == "(" {
			return nil, errors.New(unsafeError)
		}
		return testPolicyCompiler.CompilePolicy(policy)
	}
	accepted, ok := provider.current(context.Background())
	require.True(t, ok)
	bad := &Policy{CustomRules: []CustomRule{{ID: "unsafe-tenant-authored-id", Regex: "("}}}
	source.set(bad, false)
	for range 3 {
		compiled, currentOK := provider.current(context.Background())
		require.True(t, currentOK)
		assert.Same(t, accepted, compiled)
		assert.True(t, compiled.Detect("DYNAMIC-ONE").Matched())
	}
	assert.Equal(t, 2, compilations)
	assert.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.rejected))
	assert.NotEmpty(t, logs.String())
	assert.NotContains(t, logs.String(), unsafeError)
	assert.NotContains(t, logs.String(), bad.CustomRules[0].ID)
	assert.NotContains(t, logs.String(), bad.CustomRules[0].Regex)

	// Reverting to the accepted input does not invalidate its immutable cache.
	source.set(dynamicPolicy("DYNAMIC-ONE"), true)
	compiled, ok := provider.current(context.Background())
	require.True(t, ok)
	assert.Same(t, accepted, compiled)
	assert.Equal(t, 2, compilations)

	source.set(dynamicPolicy("DYNAMIC-TWO"), false)
	compiled, ok = provider.current(context.Background())
	require.True(t, ok)
	assert.False(t, compiled.Detect("DYNAMIC-ONE").Matched())
	assert.True(t, compiled.Detect("DYNAMIC-TWO").Matched())
	assert.True(t, accepted.Detect("DYNAMIC-ONE").Matched())
	assert.Equal(t, 2.0, testutil.ToFloat64(provider.metrics.applied))

	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		for _, metric := range family.Metric {
			for _, label := range metric.Label {
				assert.Equal(t, "tempo_secret_detection_policy_updates_total", family.GetName())
				assert.Equal(t, "outcome", label.GetName())
				assert.Contains(t, []string{"applied", "rejected", "superseded", "canceled"}, label.GetValue())
			}
		}
	}
}

func TestDynamicPolicyInvalidFirstOverrideUsesNativeCatalog(t *testing.T) {
	source := &dynamicPolicySource{}
	source.set(dynamicPolicy("("), false)
	provider, _ := newDynamicTestProvider(source)
	compiled, ok := provider.current(context.Background())
	require.True(t, ok)
	assert.Contains(t, compiled.Detect("sk_test_"+"0123456789abcdefghijklmn").Matches, Match{RuleID: "stripe-access-token"})
	assert.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.rejected))
	source.set(nil, true)
	baseline, ok := provider.current(context.Background())
	require.True(t, ok)
	assert.Same(t, compiled, baseline)
	source.set(dynamicPolicy("DYNAMIC-RECOVERED"), false)
	recovered, ok := provider.current(context.Background())
	require.True(t, ok)
	assert.True(t, recovered.Detect("DYNAMIC-RECOVERED").Matched())
}

func TestDynamicPolicyInitialFallbackDiscardsSupersededOverride(t *testing.T) {
	source := &dynamicPolicySource{}
	source.set(dynamicPolicy("("), false)
	provider, _ := newDynamicTestProvider(source)
	started := make(chan struct{})
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	t.Cleanup(unblock)
	provider.compile = func(policy Policy) (*CompiledPolicy, error) {
		if len(policy.CustomRules) == 0 {
			close(started)
			<-release
		}
		return testPolicyCompiler.CompilePolicy(policy)
	}
	pending := refreshDynamicPolicy(context.Background(), provider)
	awaitDynamic(t, started)
	source.set(dynamicPolicy("DYNAMIC-LATEST"), false)
	unblock()
	result := awaitDynamic(t, pending)
	require.True(t, result.ok)
	assert.True(t, result.policy.Detect("DYNAMIC-LATEST").Matched())
	assert.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.superseded))
	assert.Zero(t, testutil.ToFloat64(provider.metrics.rejected))
}

func TestDynamicPolicyDiscardsStaleCompilationAndCoalescesCallers(t *testing.T) {
	source := &dynamicPolicySource{}
	source.set(dynamicPolicy("DYNAMIC-ONE"), false)
	provider, _ := newDynamicTestProvider(source)
	original, ok := provider.current(context.Background())
	require.True(t, ok)
	started := make(chan struct{})
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	t.Cleanup(unblock)
	var compiledInputs []string
	provider.compile = func(policy Policy) (*CompiledPolicy, error) {
		compiledInputs = append(compiledInputs, policy.CustomRules[0].Regex)
		if policy.CustomRules[0].Regex == "DYNAMIC-TWO" {
			close(started)
			<-release
		}
		return testPolicyCompiler.CompilePolicy(policy)
	}
	source.set(dynamicPolicy("DYNAMIC-TWO"), false)
	first := refreshDynamicPolicy(context.Background(), provider)
	awaitDynamic(t, started)
	waiting := &dynamicWaitContext{Context: context.Background(), at: 1, waiting: make(chan struct{})}
	second := refreshDynamicPolicy(waiting, provider)
	awaitDynamic(t, waiting.waiting)
	source.set(dynamicPolicy("DYNAMIC-THREE"), false)
	unblock()
	for _, result := range []dynamicPolicyResult{awaitDynamic(t, first), awaitDynamic(t, second)} {
		require.True(t, result.ok)
		assert.False(t, result.policy.Detect("DYNAMIC-TWO").Matched())
		assert.True(t, result.policy.Detect("DYNAMIC-THREE").Matched())
	}
	assert.Equal(t, []string{"DYNAMIC-TWO", "DYNAMIC-THREE"}, compiledInputs)
	assert.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.superseded))
	assert.True(t, original.Detect("DYNAMIC-ONE").Matched())
}

func TestDynamicPolicyRereadsAfterAdmissionWait(t *testing.T) {
	source := &dynamicPolicySource{}
	source.set(dynamicPolicy("DYNAMIC-ONE"), false)
	provider, _ := newDynamicTestProvider(source)
	_, ok := provider.current(context.Background())
	require.True(t, ok)
	provider.admission <- struct{}{}
	provider.admission <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	waiting := &dynamicWaitContext{Context: ctx, at: 2, waiting: make(chan struct{})}
	var compiledInputs []string
	provider.compile = func(policy Policy) (*CompiledPolicy, error) {
		compiledInputs = append(compiledInputs, policy.CustomRules[0].Regex)
		return testPolicyCompiler.CompilePolicy(policy)
	}
	source.set(dynamicPolicy("DYNAMIC-TWO"), false)
	pending := refreshDynamicPolicy(waiting, provider)
	awaitDynamic(t, waiting.waiting)
	source.set(dynamicPolicy("DYNAMIC-THREE"), false)
	<-provider.admission
	result := awaitDynamic(t, pending)
	<-provider.admission
	require.True(t, result.ok)
	assert.True(t, result.policy.Detect("DYNAMIC-THREE").Matched())
	assert.Equal(t, []string{"DYNAMIC-THREE"}, compiledInputs)
	assert.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.superseded))
}

func TestDynamicPolicyCancellationReleasesWaitersWithoutPublishing(t *testing.T) {
	for _, stage := range []string{"provider", "admission", "compiling"} {
		t.Run(stage, func(t *testing.T) {
			source := &dynamicPolicySource{}
			source.set(dynamicPolicy("DYNAMIC-ONE"), false)
			provider, _ := newDynamicTestProvider(source)
			original, ok := provider.current(context.Background())
			require.True(t, ok)
			source.set(dynamicPolicy("DYNAMIC-TWO"), false)
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			waiting := &dynamicWaitContext{Context: ctx, at: 1, waiting: make(chan struct{})}
			var unblock func()
			switch stage {
			case "provider":
				provider.updating <- struct{}{}
				unblock = sync.OnceFunc(func() { <-provider.updating })
			case "admission":
				waiting.at = 2
				provider.admission <- struct{}{}
				provider.admission <- struct{}{}
				unblock = sync.OnceFunc(func() { <-provider.admission; <-provider.admission })
			case "compiling":
				started := make(chan struct{})
				release := make(chan struct{})
				waiting.waiting = started
				waiting.at = 0
				provider.compile = func(policy Policy) (*CompiledPolicy, error) {
					close(started)
					<-release
					return testPolicyCompiler.CompilePolicy(policy)
				}
				unblock = sync.OnceFunc(func() { close(release) })
			}
			t.Cleanup(unblock)
			pending := refreshDynamicPolicy(waiting, provider)
			awaitDynamic(t, waiting.waiting)
			cancel()
			if stage == "compiling" {
				assert.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.active))
				unblock()
			}
			result := awaitDynamic(t, pending)
			require.True(t, result.ok)
			assert.Same(t, original, result.policy)
			assert.Equal(t, 1.0, testutil.ToFloat64(provider.metrics.canceled))
			assert.Equal(t, 0.0, testutil.ToFloat64(provider.metrics.active))
			unblock()
			provider.compile = testPolicyCompiler.CompilePolicy
			recovered, ok := provider.current(context.Background())
			require.True(t, ok)
			assert.True(t, recovered.Detect("DYNAMIC-TWO").Matched())
		})
	}
}

func TestDynamicPolicyCanceledFirstCallCanInitializeLater(t *testing.T) {
	source := &dynamicPolicySource{}
	source.set(dynamicPolicy("DYNAMIC-ONE"), false)
	provider, _ := newDynamicTestProvider(source)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	compiled, ok := provider.current(ctx)
	assert.False(t, ok)
	assert.Nil(t, compiled)
	compiled, ok = provider.current(context.Background())
	require.True(t, ok)
	assert.True(t, compiled.Detect("DYNAMIC-ONE").Matched())
}

func TestDynamicPolicyProcessWideCompilationBound(t *testing.T) {
	metrics := newPolicyUpdateMetrics(prometheus.NewRegistry())
	started := make(chan struct{}, 3)
	release := make(chan struct{}, 3)
	t.Cleanup(func() { close(release) })
	var results []<-chan dynamicPolicyResult
	var waiters []*dynamicWaitContext
	for range 3 {
		source := &dynamicPolicySource{}
		source.set(dynamicPolicy("DYNAMIC-ONE"), false)
		provider, _ := newDynamicTestProvider(source)
		provider.admission = policyCompilationSlots
		provider.metrics = metrics
		provider.compile = func(policy Policy) (*CompiledPolicy, error) {
			started <- struct{}{}
			<-release
			return testPolicyCompiler.CompilePolicy(policy)
		}
		waiting := &dynamicWaitContext{Context: context.Background(), at: 2, waiting: make(chan struct{})}
		waiters = append(waiters, waiting)
		results = append(results, refreshDynamicPolicy(waiting, provider))
	}
	for _, waiting := range waiters {
		awaitDynamic(t, waiting.waiting)
	}
	awaitDynamic(t, started)
	awaitDynamic(t, started)
	assert.Equal(t, 2.0, testutil.ToFloat64(metrics.active))
	select {
	case <-started:
		t.Fatal("more than two tenants compiled concurrently")
	default:
	}
	release <- struct{}{}
	awaitDynamic(t, started)
	release <- struct{}{}
	release <- struct{}{}
	for _, pending := range results {
		result := awaitDynamic(t, pending)
		require.True(t, result.ok)
		assert.True(t, result.policy.Detect("DYNAMIC-ONE").Matched())
	}
	assert.Equal(t, 0.0, testutil.ToFloat64(metrics.active))
	assert.Equal(t, 3.0, testutil.ToFloat64(metrics.applied))
}
