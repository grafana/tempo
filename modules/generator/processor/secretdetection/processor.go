package secretdetection

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/viper"
	"github.com/zricethezav/gitleaks/v8/config"
	"github.com/zricethezav/gitleaks/v8/detect"

	"github.com/grafana/tempo/modules/generator/processor"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tempo_log "github.com/grafana/tempo/pkg/util/log"

	"github.com/grafana/tempo/pkg/tempopb"
)

// Ensure Processor satisfies the interface at compile time.
var _ processor.Processor = (*Processor)(nil)

const (
	// minSecretLength is the minimum string length that could plausibly match a
	// secret pattern. Shorter values are skipped to avoid unnecessary regex work.
	minSecretLength = 8
)

const (
	scopeResource = "resource"
	scopeSpan     = "span"
	scopeEvent    = "event"
	scopeLink     = "link"

	// Metric series names from span-metrics processor.
	seriesCallsTotal      = "traces_spanmetrics_calls_total"
	seriesLatency         = "traces_spanmetrics_latency"
	seriesSizeTotal       = "traces_spanmetrics_size_total"
	seriesTargetInfo      = "traces_target_info"
)

var (
	metricSecretDetectionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "secret_detections_total",
		Help:      "Total number of secrets detected in span attributes.",
	}, []string{"tenant", "attribute_scope"})

	metricSecretDetectionsInMetricsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "secret_detections_in_metrics_total",
		Help:      "Total number of secrets detected in attributes that are also span-metrics dimensions (will appear in Mimir).",
	}, []string{"tenant", "attribute_scope"})
)

// sharedDetector is a single gitleaks detector shared across all processor instances.
// Parsed once via sync.Once to avoid racing on the global viper instance that
// detect.NewDetectorDefaultConfig() uses internally.
// DetectString is safe for concurrent use: it only reads immutable config/rules
// and uses atomic operations for counters. Verify this assumption on gitleaks upgrades.
var (
	sharedDetector     *detect.Detector
	sharedDetectorOnce sync.Once
	sharedDetectorErr  error
)

func getSharedDetector() (*detect.Detector, error) {
	sharedDetectorOnce.Do(func() {
		v := viper.New()
		v.SetConfigType("toml")
		if err := v.ReadConfig(strings.NewReader(config.DefaultConfig)); err != nil {
			sharedDetectorErr = fmt.Errorf("secret-detection: failed to read default config: %w", err)
			return
		}
		var vc config.ViperConfig
		if err := v.Unmarshal(&vc); err != nil {
			sharedDetectorErr = fmt.Errorf("secret-detection: failed to unmarshal config: %w", err)
			return
		}
		cfg, err := vc.Translate()
		if err != nil {
			sharedDetectorErr = fmt.Errorf("secret-detection: failed to translate config: %w", err)
			return
		}
		sharedDetector = detect.NewDetector(cfg)
	})
	return sharedDetector, sharedDetectorErr
}

type Processor struct {
	Cfg      Config
	tenant   string
	detector *detect.Detector
	logger   *tempo_log.RateLimitedLogger

	// Pre-resolved counters to avoid WithLabelValues lookups in the hot path.
	detectionsTotal     map[string]prometheus.Counter
	detectionsInMetrics map[string]prometheus.Counter
}

func New(cfg Config, tenant string, logger log.Logger) (*Processor, error) {
	detector, err := getSharedDetector()
	if err != nil {
		return nil, err
	}

	scopes := []string{scopeResource, scopeSpan, scopeEvent, scopeLink}
	detTotal := make(map[string]prometheus.Counter, len(scopes))
	detInMetrics := make(map[string]prometheus.Counter, len(scopes))
	for _, s := range scopes {
		detTotal[s] = metricSecretDetectionsTotal.WithLabelValues(tenant, s)
		detInMetrics[s] = metricSecretDetectionsInMetricsTotal.WithLabelValues(tenant, s)
	}

	return &Processor{
		Cfg:                 cfg,
		tenant:              tenant,
		detector:            detector,
		logger:              tempo_log.NewRateLimitedLogger(1, level.Warn(logger)),
		detectionsTotal:     detTotal,
		detectionsInMetrics: detInMetrics,
	}, nil
}

func (p *Processor) Name() string {
	return processor.SecretDetectionName
}

func (p *Processor) PushSpans(_ context.Context, req *tempopb.PushSpansRequest) {
	for _, rs := range req.Batches {
		p.scanAttrs(rs.GetResource().GetAttributes(), scopeResource, nil, nil)

		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				p.scanAttrs(span.GetAttributes(), scopeSpan, span.TraceId, span.SpanId)

				for _, event := range span.GetEvents() {
					p.scanAttrs(event.GetAttributes(), scopeEvent, span.TraceId, span.SpanId)
				}
				for _, link := range span.GetLinks() {
					p.scanAttrs(link.GetAttributes(), scopeLink, span.TraceId, span.SpanId)
				}
			}
		}
	}
}

func (p *Processor) scanAttrs(attrs []*v1_common.KeyValue, scope string, traceID, spanID []byte) {
	for _, attr := range attrs {
		av := attr.GetValue()
		if av == nil {
			continue
		}

		// Fast-path: non-string scalar types produce short, structured values
		// that cannot match any secret pattern. Skip without calling DetectString.
		switch av.Value.(type) {
		case *v1_common.AnyValue_BoolValue,
			*v1_common.AnyValue_IntValue,
			*v1_common.AnyValue_DoubleValue:
			continue
		}

		value := av.GetStringValue()
		if len(value) < minSecretLength {
			continue
		}

		findings := p.detector.DetectString(value)
		if len(findings) == 0 {
			continue
		}

		// Compute metrics exposure once per attribute, not per finding.
		inMetrics, metricSeries := p.checkMetricsExposure(attr.GetKey(), scope)

		for _, f := range findings {
			p.detectionsTotal[scope].Inc()

			if inMetrics {
				p.detectionsInMetrics[scope].Inc()
			}

			p.logger.Log(
				"msg", "secret detected in span attribute",
				"tenant", p.tenant,
				"traceID", hex.EncodeToString(traceID),
				"spanID", hex.EncodeToString(spanID),
				"attr_key", attr.GetKey(),
				"rule", f.RuleID,
				"scope", scope,
				"in_metrics", inMetrics,
				"metric_series", metricSeries,
				"ts", time.Now().UTC().Format(time.RFC3339),
			)
		}
	}
}

// checkMetricsExposure determines if the secret-bearing attribute will appear as a
// label value in generated metrics (Mimir). Returns whether it's exposed and a
// comma-separated list of affected metric series names.
//
// Limitations:
// - May over-report when SkipMetricsGeneration is set on the request (the secret
//   won't actually land in metrics, but this processor doesn't see that flag).
// - Only checks user-configured dimensions and dimension mappings, not intrinsic
//   dimensions (service, span_name, etc.) which are unlikely to contain secrets.
// - Reports all span-metrics series regardless of which subprocessors are enabled.
func (p *Processor) checkMetricsExposure(attrKey, scope string) (bool, string) {
	info := p.Cfg.SpanMetricsInfo
	var series []string

	switch scope {
	case scopeResource:
		// target_info dumps all resource attributes as label values unless excluded.
		if info.EnableTargetInfo {
			if _, excluded := info.TargetInfoExcludedDimensions[attrKey]; !excluded {
				series = append(series, seriesTargetInfo)
			}
		}

	case scopeSpan:
		// Check if attribute key is a configured span-metrics dimension.
		if _, ok := info.Dimensions[attrKey]; ok {
			series = append(series, seriesCallsTotal, seriesLatency, seriesSizeTotal)
		}
		// Check if attribute key is a source label in a dimension mapping.
		if _, ok := info.DimensionMappingSourceLabels[attrKey]; ok {
			series = append(series, seriesCallsTotal, seriesLatency, seriesSizeTotal)
		}

	case scopeEvent, scopeLink:
		// Event and link attributes are never used by span-metrics.
	}

	if len(series) == 0 {
		return false, ""
	}

	// Deduplicate in case both dimension and mapping matched.
	seen := make(map[string]struct{}, len(series))
	var unique []string
	for _, s := range series {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			unique = append(unique, s)
		}
	}

	return true, strings.Join(unique, ",")
}

func (p *Processor) Shutdown(_ context.Context) {
	// No-op. The detector is a shared singleton and not owned by this instance.
}
