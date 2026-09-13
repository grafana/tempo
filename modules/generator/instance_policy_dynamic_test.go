package generator

import (
	"context"
	"flag"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/secretdetection"
	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dynamicInstanceOverrides struct {
	mockOverrides
	policy           atomic.Pointer[secrets.Policy]
	detectionEnabled atomic.Bool
}

func (o *dynamicInstanceOverrides) SecretsPolicy(string) (*secrets.Policy, bool) {
	return o.policy.Load(), false
}

func (o *dynamicInstanceOverrides) MetricsGeneratorProcessors(tenant string) map[string]struct{} {
	if o.detectionEnabled.Load() {
		return map[string]struct{}{processor.SecretDetectionName: {}}
	}
	return o.mockOverrides.MetricsGeneratorProcessors(tenant)
}

type dynamicInstanceFinding struct {
	traceID string
	rule    string
}

type dynamicInstanceLogger struct {
	mu           sync.Mutex
	findings     []dynamicInstanceFinding
	blockTraceID string
	blocked      chan struct{}
	release      chan struct{}
	blockOnce    sync.Once
}

func (l *dynamicInstanceLogger) Log(keyvals ...any) error {
	var message string
	var finding dynamicInstanceFinding
	for i := 0; i+1 < len(keyvals); i += 2 {
		switch keyvals[i] {
		case "msg":
			message, _ = keyvals[i+1].(string)
		case "traceID":
			finding.traceID, _ = keyvals[i+1].(string)
		case "rule":
			finding.rule, _ = keyvals[i+1].(string)
		}
	}
	if message != "secret detected in trace field" {
		return nil
	}
	l.mu.Lock()
	l.findings = append(l.findings, finding)
	l.mu.Unlock()
	if finding.traceID == l.blockTraceID {
		l.blockOnce.Do(func() {
			close(l.blocked)
			<-l.release
		})
	}
	return nil
}

func (l *dynamicInstanceLogger) rules(traceID string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var rules []string
	for _, finding := range l.findings {
		if finding.traceID == traceID {
			rules = append(rules, finding.rule)
		}
	}
	return rules
}

func dynamicInstancePolicy(id, regex string) *secrets.Policy {
	return &secrets.Policy{CustomRules: []secrets.CustomRule{{ID: id, Regex: regex}}}
}

func dynamicInstanceRequest(traceID byte, values ...string) *tempopb.PushSpansRequest {
	attributes := make([]*commonv1.KeyValue, 0, len(values))
	for _, value := range values {
		attributes = append(attributes, test.MakeAttribute("payload", value))
	}
	return &tempopb.PushSpansRequest{
		SkipMetricsGeneration: true,
		Batches: []*v1.ResourceSpans{{ScopeSpans: []*v1.ScopeSpans{{Spans: []*v1.Span{{
			TraceId:         []byte{traceID},
			SpanId:          []byte{1},
			EndTimeUnixNano: 1, // Expired spans still require detection before filtering.
			Attributes:      attributes,
		}}}}}},
	}
}

func newDynamicPolicyInstance(t *testing.T, overrides *dynamicInstanceOverrides, logger *dynamicInstanceLogger) *instance {
	t.Helper()
	overrides.detectionEnabled.Store(true)
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.Processor.SecretDetection.Enabled = true
	compiler, err := secrets.NewPolicyCompiler(nil)
	require.NoError(t, err)
	cfg.Processor.SecretDetection.PolicyCompiler = compiler
	i, err := newInstance(cfg, t.Name(), overrides, &noopStorage{}, logger)
	require.NoError(t, err)
	return i
}

func awaitInstancePolicy[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(10 * time.Second):
		t.Fatal("instance policy lifecycle did not complete")
		var zero T
		return zero
	}
}

func refreshInstancePolicy(i *instance) <-chan error {
	result := make(chan error, 1)
	go func() { result <- i.updateProcessors() }()
	return result
}

func pushInstancePolicy(i *instance, request *tempopb.PushSpansRequest) <-chan struct{} {
	result := make(chan struct{})
	go func() {
		i.pushSpans(context.Background(), request)
		close(result)
	}()
	return result
}

func TestInstanceSecretDetectionTenantOptInLifecycle(t *testing.T) {
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.Processor.SecretDetection.Enabled = true
	compiler, err := secrets.NewPolicyCompiler(&[]string{})
	require.NoError(t, err)
	cfg.Processor.SecretDetection.PolicyCompiler = compiler
	overrides := &dynamicInstanceOverrides{
		mockOverrides: mockOverrides{processors: map[string]struct{}{processor.SpanMetricsName: {}}},
	}
	overrides.policy.Store(dynamicInstancePolicy("tenant-old", "TENANT-OLD"))
	logger := &dynamicInstanceLogger{}
	i, err := newInstance(cfg, t.Name(), overrides, &noopStorage{}, logger)
	require.NoError(t, err)
	t.Cleanup(i.shutdown)

	// A configured policy and another active processor do not opt this tenant in.
	i.pushSpans(context.Background(), dynamicInstanceRequest(1, "TENANT-OLD"))
	require.Empty(t, logger.rules("01"))

	overrides.detectionEnabled.Store(true)
	require.NoError(t, i.updateProcessors())
	i.pushSpans(context.Background(), dynamicInstanceRequest(2, "TENANT-OLD"))
	require.Equal(t, []string{"tenant-old"}, logger.rules("02"))

	overrides.detectionEnabled.Store(false)
	require.NoError(t, i.updateProcessors())
	overrides.policy.Store(dynamicInstancePolicy("tenant-new", "TENANT-NEW"))
	i.pushSpansFromQueue(context.Background(), time.Now(), dynamicInstanceRequest(3, "TENANT-OLD", "TENANT-NEW"))
	require.Empty(t, logger.rules("03"))

	// Re-enabling uses the current policy, not the policy from before opt-out.
	overrides.detectionEnabled.Store(true)
	require.NoError(t, i.updateProcessors())
	i.pushSpansFromQueue(context.Background(), time.Now(), dynamicInstanceRequest(4, "TENANT-OLD", "TENANT-NEW"))
	require.Equal(t, []string{"tenant-new"}, logger.rules("04"))
}

func TestInstanceSecretDetectionRequiresProcessFeature(t *testing.T) {
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	overrides := &dynamicInstanceOverrides{}
	overrides.detectionEnabled.Store(true)
	overrides.policy.Store(dynamicInstancePolicy("tenant-rule", "TENANT-SECRET"))
	logger := &dynamicInstanceLogger{}
	i, err := newInstance(cfg, t.Name(), overrides, &noopStorage{}, logger)
	require.NoError(t, err)
	t.Cleanup(i.shutdown)

	i.pushSpans(context.Background(), dynamicInstanceRequest(1, "TENANT-SECRET"))
	require.Empty(t, logger.rules("01"))
}

func TestInstanceFirstBatchUsesEffectivePolicyOrSelectedFallback(t *testing.T) {
	for _, selection := range []struct {
		name  string
		ids   *[]string
		rules []string
	}{
		{"default", nil, []string{"slack-bot-token", "stripe-access-token"}},
		{"selected", &[]string{"stripe-access-token"}, []string{"stripe-access-token"}},
		{"empty", &[]string{}, nil},
	} {
		for _, codec := range []string{codecPushBytes, codecOTLP} {
			for _, valid := range []bool{false, true} {
				name := selection.name + "/" + codec + "/invalid"
				if valid {
					name = selection.name + "/" + codec + "/valid"
				}
				t.Run(name, func(t *testing.T) {
					cfg := &Config{}
					cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
					cfg.Codec = codec
					cfg.Processor.SecretDetection.Enabled = true
					compiler, err := secrets.NewPolicyCompiler(selection.ids)
					require.NoError(t, err)
					cfg.Processor.SecretDetection.PolicyCompiler = compiler
					overrides := &dynamicInstanceOverrides{}
					overrides.detectionEnabled.Store(true)
					expected := append([]string(nil), selection.rules...)
					if valid {
						overrides.policy.Store(dynamicInstancePolicy("first-rule", "DYNAMIC-FIRST"))
						expected = append(expected, "first-rule")
					} else {
						overrides.policy.Store(dynamicInstancePolicy("broken-rule", "("))
					}
					logger := &dynamicInstanceLogger{}
					i, err := newInstance(cfg, t.Name(), overrides, &noopStorage{}, logger)
					require.NoError(t, err)
					t.Cleanup(i.shutdown)
					request := dynamicInstanceRequest(1, "DYNAMIC-FIRST", "sk_test_"+"0123456789abcdefghijklmn", "xoxb-"+"1234567890-1234567890123-abcdefghijklmnopqrstuvwx")
					i.pushSpansFromQueue(context.Background(), time.Now(), request)
					assert.ElementsMatch(t, expected, logger.rules("01"))
					assert.Empty(t, request.Batches[0].ScopeSpans[0].Spans)
				})
			}
		}
	}
}

func TestInstancePolicyRefreshKeepsBatchesOnCompleteSnapshots(t *testing.T) {
	overrides := &dynamicInstanceOverrides{}
	overrides.policy.Store(dynamicInstancePolicy("old-rule", "DYNAMIC-(SHARED|OLD)"))
	logger := &dynamicInstanceLogger{
		blockTraceID: "02",
		blocked:      make(chan struct{}),
		release:      make(chan struct{}),
	}
	i := newDynamicPolicyInstance(t, overrides, logger)
	t.Cleanup(i.shutdown)
	releaseScan := sync.OnceFunc(func() { close(logger.release) })
	t.Cleanup(releaseScan)
	oldDetector := i.processors[processor.SecretDetectionName].(*secretdetection.Processor)

	buildStarted := make(chan struct{})
	buildReady := make(chan struct{})
	releaseBuild := make(chan struct{})
	unblockBuild := sync.OnceFunc(func() { close(releaseBuild) })
	t.Cleanup(unblockBuild)
	i.processorUpdatesMtx.Lock()
	actualProvider := i.secretsPolicyProvider
	var startOnce, readyOnce sync.Once
	// Hold the provider's synchronous build phase at a channel-controlled
	// boundary, then execute the real compiler/provider before publication.
	i.secretsPolicyProvider = func(ctx context.Context) (*secrets.CompiledPolicy, bool) {
		startOnce.Do(func() {
			close(buildStarted)
			select {
			case <-releaseBuild:
			case <-ctx.Done():
			}
		})
		compiled, ok := actualProvider(ctx)
		readyOnce.Do(func() { close(buildReady) })
		return compiled, ok
	}
	i.processorUpdatesMtx.Unlock()
	overrides.policy.Store(dynamicInstancePolicy("new-rule", "DYNAMIC-(SHARED|NEW)"))
	refresh := refreshInstancePolicy(i)
	awaitInstancePolicy(t, buildStarted)
	values := []string{"DYNAMIC-SHARED", "DYNAMIC-OLD", "DYNAMIC-NEW", "DYNAMIC-SHARED"}

	// A full batch completes while the refresh build is blocked. A compile
	// under processorsMtx would deadlock here instead of serving the old policy.
	awaitInstancePolicy(t, pushInstancePolicy(i, dynamicInstanceRequest(1, values...)))
	assert.Equal(t, []string{"old-rule", "old-rule", "old-rule"}, logger.rules("01"))

	// Keep another old batch inside its scanner while the new build finishes.
	// Replacement must wait at the existing processor publication boundary.
	oldBatch := pushInstancePolicy(i, dynamicInstanceRequest(2, values...))
	awaitInstancePolicy(t, logger.blocked)
	unblockBuild()
	awaitInstancePolicy(t, buildReady)
	releaseScan()
	awaitInstancePolicy(t, oldBatch)
	require.NoError(t, awaitInstancePolicy(t, refresh))
	i.processorUpdatesMtx.Lock()
	i.secretsPolicyProvider = actualProvider
	i.processorUpdatesMtx.Unlock()
	newDetector := i.processors[processor.SecretDetectionName].(*secretdetection.Processor)
	require.NotSame(t, oldDetector, newDetector)
	assert.Equal(t, []string{"old-rule", "old-rule", "old-rule"}, logger.rules("02"))

	// Repeated values must be re-evaluated under the new batch policy, not a
	// value cache populated by either old batch.
	i.pushSpans(context.Background(), dynamicInstanceRequest(3, values...))
	assert.Equal(t, []string{"new-rule", "new-rule", "new-rule"}, logger.rules("03"))
	require.NoError(t, i.updateProcessors())
	assert.Same(t, newDetector, i.processors[processor.SecretDetectionName])
}

func TestInstanceShutdownCancelsPolicyWaitAndPreventsReplacement(t *testing.T) {
	overrides := &dynamicInstanceOverrides{}
	overrides.policy.Store(dynamicInstancePolicy("old-rule", "DYNAMIC-OLD"))
	i := newDynamicPolicyInstance(t, overrides, &dynamicInstanceLogger{})
	shutdown := sync.OnceFunc(i.shutdown)
	t.Cleanup(shutdown)
	i.processorUpdatesMtx.Lock()
	actualProvider := i.secretsPolicyProvider
	waiting := make(chan struct{})
	var waitOnce sync.Once
	i.secretsPolicyProvider = func(ctx context.Context) (*secrets.CompiledPolicy, bool) {
		waitOnce.Do(func() { close(waiting) })
		<-ctx.Done()
		return actualProvider(ctx)
	}
	i.processorUpdatesMtx.Unlock()
	overrides.policy.Store(dynamicInstancePolicy("new-rule", "DYNAMIC-NEW"))
	refresh := refreshInstancePolicy(i)
	awaitInstancePolicy(t, waiting)
	stopped := make(chan struct{})
	go func() {
		shutdown()
		close(stopped)
	}()
	require.NoError(t, awaitInstancePolicy(t, refresh))
	awaitInstancePolicy(t, stopped)
	assert.ErrorIs(t, i.lifecycleCtx.Err(), context.Canceled)
	require.NoError(t, i.updateProcessors())
	assert.Empty(t, i.processors)
}
