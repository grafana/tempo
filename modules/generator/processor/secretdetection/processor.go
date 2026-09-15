package secretdetection

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/tempopb"
)

var _ processor.Processor = (*Processor)(nil)

const (
	scopeResource = "resource"
	scopeScope    = "scope"
	scopeSpan     = "span"
	scopeEvent    = "event"
	scopeLink     = "link"
	scopeTrace    = "trace"

	maxFindingLogsPerTrace      = 1000
	findingLogsPerSecond        = 100
	findingLogBurst             = 1000
	processFindingLogsPerSecond = 1000
	processFindingLogBurst      = 2000
	coverageLogsPerSecond       = 1
	coverageLogBurst            = 10
	metricNamespace             = "tempo"
)

var allScopes = []string{scopeResource, scopeScope, scopeSpan, scopeEvent, scopeLink, scopeTrace}

var (
	sharedFindingLogLimiter  = rate.NewLimiter(processFindingLogsPerSecond, processFindingLogBurst)
	sharedCoverageLogLimiter = rate.NewLimiter(coverageLogsPerSecond, coverageLogBurst)
)

var (
	metricSecretDetectionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Name:      "secret_detections_total",
		Help:      "Total number of secret rule matches in trace fields.",
	}, []string{"attribute_scope"})

	metricSecretDetectionPushesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Name:      "secret_detection_pushes_total",
		Help:      "Total number of trace batches presented for secret detection.",
	}, []string{"source_stream"})

	metricSecretDetectionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metricNamespace,
		Name:      "secret_detection_duration_seconds",
		Help:      "Time spent scanning trace batches for secrets.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"source_stream"})
)

type Processor struct {
	Cfg                       Config
	tenant                    string
	policy                    *secrets.CompiledPolicy
	logFinding                func(...interface{}) error
	findingLogLimiter         *rate.Limiter
	processFindingLogLimiter  *rate.Limiter
	coverageLogLimiter        *rate.Limiter
	processCoverageLogLimiter *rate.Limiter
	tenantMetrics             *TenantMetrics
	now                       func() time.Time

	detectionsTotal map[string]prometheus.Counter
	pushesTotal     prometheus.Counter
	duration        prometheus.Observer
}

func New(cfg Config, tenant string, logger log.Logger, tenantMetrics *TenantMetrics) (*Processor, error) {
	compiled := cfg.CompiledPolicy
	if compiled == nil {
		var err error
		if cfg.PolicyCompiler == nil {
			cfg.PolicyCompiler, err = secrets.NewPolicyCompiler(nil)
			if err != nil {
				return nil, err
			}
		}
		compiled, err = cfg.PolicyCompiler.CompilePolicy(secrets.Policy{})
		if err != nil {
			return nil, err
		}
	}

	detectionsTotal := make(map[string]prometheus.Counter, len(allScopes))
	for _, scope := range allScopes {
		detectionsTotal[scope] = metricSecretDetectionsTotal.WithLabelValues(scope)
	}

	sourceStream := cfg.SourceStream
	if sourceStream == "" {
		sourceStream = "tempo-ingest"
	}
	return &Processor{
		Cfg:                       cfg,
		tenant:                    tenant,
		policy:                    compiled,
		tenantMetrics:             tenantMetrics,
		logFinding:                level.Warn(logger).Log,
		findingLogLimiter:         rate.NewLimiter(findingLogsPerSecond, findingLogBurst),
		coverageLogLimiter:        rate.NewLimiter(coverageLogsPerSecond, coverageLogBurst),
		processCoverageLogLimiter: sharedCoverageLogLimiter,
		detectionsTotal:           detectionsTotal,
		now:                       time.Now,
		processFindingLogLimiter:  sharedFindingLogLimiter,
		pushesTotal:               metricSecretDetectionPushesTotal.WithLabelValues(sourceStream),
		duration:                  metricSecretDetectionDuration.WithLabelValues(sourceStream),
	}, nil
}

func (p *Processor) Name() string {
	return processor.SecretDetectionName
}

func (p *Processor) PushSpans(_ context.Context, request *tempopb.PushSpansRequest) {
	started := p.now()
	p.pushesTotal.Inc()
	if p.tenantMetrics != nil {
		p.tenantMetrics.recordPush()
	}
	defer func() {
		p.duration.Observe(p.now().Sub(started).Seconds())
	}()
	var findingStates traceFindingStates
	directTraceIDs := [1][]byte{}
	sharedTraceIDs := map[traceSetKey][][]byte{}
	detector := p.policy.NewBatchDetector()

	secrets.WalkPushSpansRequest(request, func(field secrets.TraceField) bool {
		traceIDs := traceIDsForField(request, field, directTraceIDs[:], sharedTraceIDs)
		verdict := detector.Detect(field.Value)
		p.recordVerdict(field, verdict, traceIDs, &findingStates, started)
		return true
	})
}

func (p *Processor) recordVerdict(
	field secrets.TraceField,
	verdict secrets.Verdict,
	traceIDs [][]byte,
	states *traceFindingStates,
	started time.Time,
) {
	if !verdict.Matched() {
		return
	}
	scope := fieldScope(field.Kind)
	for _, match := range verdict.Matches {
		p.detectionsTotal[scope].Inc()
		if p.tenantMetrics != nil {
			p.tenantMetrics.recordDetection(scope)
		}
		for _, traceID := range traceIDs {
			state := states.stateFor(traceID, field)
			if state.findingLogs >= maxFindingLogsPerTrace {
				if !state.findingLimitReported {
					state.findingLimitReported = true
					p.logCoverageGap("finding_log_limit_exceeded", traceID, field, started)
				}
				continue
			}
			if !p.findingLogLimiter.Allow() || !p.processFindingLogLimiter.Allow() {
				if !state.findingRateLimitReported {
					state.findingRateLimitReported = true
					p.logCoverageGap("finding_log_rate_limit_exceeded", traceID, field, started)
				}
				continue
			}
			state.findingLogs++
			if err := p.logFinding(
				"msg", "secret detected in trace field",
				"tenant", p.tenant,
				"traceID", hex.EncodeToString(traceID),
				"spanID", hex.EncodeToString(field.SpanID),
				"field_kind", string(field.Kind),
				"rule", match.RuleID,
				"ts", started.UTC().Format(time.RFC3339),
			); err != nil {
				p.logCoverageGap("finding_log_error", traceID, field, started)
			}
		}
	}
}

type traceFindingKey struct {
	traceID        [16]byte
	invalidTraceID string
	resource       int
	scope          int
}

func traceFindingKeysEqual(left, right traceFindingKey) bool {
	return left.traceID == right.traceID &&
		left.resource == right.resource &&
		left.scope == right.scope &&
		left.invalidTraceID == right.invalidTraceID
}

type traceFindingState struct {
	findingLogs              int
	findingLimitReported     bool
	findingRateLimitReported bool
}

type traceFindingStates struct {
	inlineKeys   [8]traceFindingKey
	inlineStates [8]traceFindingState
	inlineCount  int
	index        map[traceFindingKey]int
	states       []traceFindingState
}

func (s *traceFindingStates) stateFor(traceID []byte, field secrets.TraceField) *traceFindingState {
	key := newTraceFindingKey(traceID, field)
	if s.index != nil {
		if index, ok := s.index[key]; ok {
			return &s.states[index]
		}
		index := len(s.states)
		s.index[key] = index
		s.states = append(s.states, traceFindingState{})
		return &s.states[index]
	}
	if s.inlineCount > 0 {
		last := s.inlineCount - 1
		if traceFindingKeysEqual(s.inlineKeys[last], key) {
			return &s.inlineStates[last]
		}
		for i := 0; i < last; i++ {
			if traceFindingKeysEqual(s.inlineKeys[i], key) {
				return &s.inlineStates[i]
			}
		}
	}
	if s.inlineCount < len(s.inlineKeys) {
		index := s.inlineCount
		s.inlineKeys[index] = key
		s.inlineCount++
		return &s.inlineStates[index]
	}

	s.index = make(map[traceFindingKey]int, len(s.inlineKeys)*2)
	s.states = make([]traceFindingState, len(s.inlineStates), len(s.inlineStates)*2)
	copy(s.states, s.inlineStates[:])
	for i, inlineKey := range s.inlineKeys {
		s.index[inlineKey] = i
	}
	index := len(s.states)
	s.index[key] = index
	s.states = append(s.states, traceFindingState{})
	return &s.states[index]
}

func newTraceFindingKey(traceID []byte, field secrets.TraceField) traceFindingKey {
	key := traceFindingKey{resource: -1, scope: -1}
	switch len(traceID) {
	case 0:
		key.resource = field.Location.Resource
		key.scope = field.Location.Scope
	case len(key.traceID):
		copy(key.traceID[:], traceID)
	default:
		key.invalidTraceID = string(traceID)
	}
	return key
}

func (p *Processor) logCoverageGap(reason string, traceID []byte, field secrets.TraceField, timestamp time.Time) {
	if !p.coverageLogLimiter.Allow() || !p.processCoverageLogLimiter.Allow() {
		return
	}
	// A failed log sink may also reject this diagnostic. Do not retry without
	// bounds or include logger errors, which could contain the rejected input.
	_ = p.logFinding(
		"msg", "secret detection coverage gap",
		"tenant", p.tenant,
		"traceID", hex.EncodeToString(traceID),
		"field_kind", string(field.Kind),
		"reason", reason,
		"ts", timestamp.UTC().Format(time.RFC3339),
	)
}

func fieldScope(kind secrets.FieldKind) string {
	switch kind {
	case secrets.FieldKindResourceAttribute:
		return scopeResource
	case secrets.FieldKindScopeAttribute:
		return scopeScope
	case secrets.FieldKindSpanAttribute:
		return scopeSpan
	case secrets.FieldKindEventAttribute:
		return scopeEvent
	case secrets.FieldKindLinkAttribute:
		return scopeLink
	default:
		return scopeTrace
	}
}

type traceSetKey struct {
	resource int
	scope    int
}

func traceIDsForField(request *tempopb.PushSpansRequest, field secrets.TraceField, direct [][]byte, shared map[traceSetKey][][]byte) [][]byte {
	if len(field.TraceID) > 0 {
		direct[0] = field.TraceID
		return direct
	}
	key := traceSetKey{resource: field.Location.Resource, scope: field.Location.Scope}
	if traceIDs, ok := shared[key]; ok {
		return traceIDs
	}
	if request == nil || key.resource < 0 || key.resource >= len(request.Batches) {
		shared[key] = [][]byte{nil}
		return shared[key]
	}

	resource := request.Batches[key.resource]
	if resource == nil {
		shared[key] = [][]byte{nil}
		return shared[key]
	}

	scopes := resource.ScopeSpans
	if key.scope >= 0 {
		if key.scope >= len(scopes) {
			shared[key] = [][]byte{nil}
			return shared[key]
		}
		scopes = scopes[key.scope : key.scope+1]
	}

	seen := map[[16]byte]struct{}{}
	var seenInvalid map[string]struct{}
	var traceIDs [][]byte
	for _, scope := range scopes {
		if scope == nil {
			continue
		}
		for _, span := range scope.Spans {
			if span == nil || len(span.TraceId) == 0 {
				continue
			}
			if len(span.TraceId) == 16 {
				var traceKey [16]byte
				copy(traceKey[:], span.TraceId)
				if _, ok := seen[traceKey]; ok {
					continue
				}
				seen[traceKey] = struct{}{}
			} else {
				if seenInvalid == nil {
					seenInvalid = map[string]struct{}{}
				}
				traceKey := string(span.TraceId)
				if _, ok := seenInvalid[traceKey]; ok {
					continue
				}
				seenInvalid[traceKey] = struct{}{}
			}
			traceIDs = append(traceIDs, span.TraceId)
		}
	}
	if len(traceIDs) == 0 {
		traceIDs = append(traceIDs, nil)
	}
	shared[key] = traceIDs
	return traceIDs
}

func (p *Processor) Shutdown(context.Context) {}
