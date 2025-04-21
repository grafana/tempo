// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterconfig // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"

import (
	"errors"
	"fmt"
	"sort"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset/regexp"
)

// MatchConfig has two optional MatchProperties one to define what is processed
// by the processor, captured under the 'include' and the second, exclude, to
// define what is excluded from the processor.
type MatchConfig struct {
	// Include specifies the set of input data properties that must be present in order
	// for this processor to apply to it.
	// Note: If `exclude` is specified, the input data is compared against those
	// properties after the `include` properties.
	// This is an optional field. If neither `include` and `exclude` are set, all input data
	// are processed. If `include` is set and `exclude` isn't set, then all
	// input data matching the properties in this structure are processed.
	Include *MatchProperties `mapstructure:"include"`

	// Exclude specifies when this processor will not be applied to the input data
	// which match the specified properties.
	// Note: The `exclude` properties are checked after the `include` properties,
	// if they exist, are checked.
	// If `include` isn't specified, the `exclude` properties are checked against
	// all input data.
	// This is an optional field. If neither `include` and `exclude` are set, all input data
	// is processed. If `exclude` is set and `include` isn't set, then all the
	// input data that does not match the properties in this structure are processed.
	Exclude *MatchProperties `mapstructure:"exclude"`
}

// MatchProperties specifies the set of properties in a spans/log/metric to match
// against and if the input data should be included or excluded from the
// processor. At least one of services (spans only), names or
// attributes must be specified. It is supported to have all specified, but
// this requires all the properties to match for the inclusion/exclusion to
// occur.
// The following are examples of invalid configurations:
//
//	attributes/bad1:
//	  # This is invalid because include is specified with neither services or
//	  # attributes.
//	  include:
//	  actions: ...
//
//	span/bad2:
//	  exclude:
//	  	# This is invalid because services, span_names and attributes have empty values.
//	    services:
//	    span_names:
//	    attributes:
//	  actions: ...
//
// Please refer to processor/attributesprocessor/testdata/config.yaml and
// processor/spanprocessor/testdata/config.yaml for valid configurations.
type MatchProperties struct {
	// Config configures the matching patterns used when matching span properties.
	filterset.Config `mapstructure:",squash"`

	// Note: For spans, one of Services, SpanNames, Attributes, Resources or Libraries must be specified with a
	// non-empty value for a valid configuration.

	// For logs, one of LogNames, Attributes, Resources or Libraries must be specified with a
	// non-empty value for a valid configuration.

	// For metrics, one of MetricNames, Expressions, or ResourceAttributes must be specified with a
	// non-empty value for a valid configuration.

	// Services specify the list of items to match service name against.
	// A match occurs if the span's service name matches at least one item in this list.
	// This is an optional field.
	Services []string `mapstructure:"services"`

	// SpanNames specify the list of items to match span name against.
	// A match occurs if the span name matches at least one item in this list.
	// This is an optional field.
	SpanNames []string `mapstructure:"span_names"`

	// LogBodies is a list of strings that the LogRecord's body field must match
	// against.
	LogBodies []string `mapstructure:"log_bodies"`

	// LogSeverityTexts is a list of strings that the LogRecord's severity text field must match
	// against.
	LogSeverityTexts []string `mapstructure:"log_severity_texts"`

	// LogSeverityNumber defines how to match against a log record's SeverityNumber, if defined.
	LogSeverityNumber *LogSeverityNumberMatchProperties `mapstructure:"log_severity_number"`

	// MetricNames is a list of strings to match metric name against.
	// A match occurs if metric name matches at least one item in the list.
	// This field is optional.
	MetricNames []string `mapstructure:"metric_names"`

	// Attributes specifies the list of attributes to match against.
	// All of these attributes must match for a match to occur.
	// This is an optional field.
	Attributes []Attribute `mapstructure:"attributes"`

	// Resources specify the list of items to match the resources against.
	// A match occurs if the data's resources match at least one item in this list.
	// This is an optional field.
	Resources []Attribute `mapstructure:"resources"`

	// Libraries specify the list of items to match the implementation library against.
	// A match occurs if the span's implementation library matches at least one item in this list.
	// This is an optional field.
	Libraries []InstrumentationLibrary `mapstructure:"libraries"`

	// SpanKinds specify the list of items to match the span kind against.
	// A match occurs if the span's span kind matches at least one item in this list.
	// This is an optional field
	SpanKinds []string `mapstructure:"span_kinds"`
}

var (
	ErrMissingRequiredSpanField   = errors.New(`at least one of "attributes", "libraries",  or "resources" field must be specified`)
	ErrInvalidLogField            = errors.New("services, span_names, span_kinds and metric_names are not valid for log records")
	ErrMissingRequiredLogField    = errors.New(`at least one of "attributes", "libraries", "span_kinds", "resources", "log_bodies", "log_severity_texts" or "log_severity_number" field must be specified`)
	ErrMissingRequiredMetricField = errors.New(`at least one of "metric_names" or "resources" field must be specified`)
	ErrInvalidMetricField         = errors.New(`"span_names", "span_kinds", "log_bodies", "log_severity_texts", "log_severity_number", "services", "attributes" and "libraries" are not valid for metrics`)

	spanKinds = map[string]bool{
		traceutil.SpanKindStr(ptrace.SpanKindInternal): true,
		traceutil.SpanKindStr(ptrace.SpanKindClient):   true,
		traceutil.SpanKindStr(ptrace.SpanKindServer):   true,
		traceutil.SpanKindStr(ptrace.SpanKindConsumer): true,
		traceutil.SpanKindStr(ptrace.SpanKindProducer): true,
	}
)

// ValidateForSpans validates properties for spans.
func (mp *MatchProperties) ValidateForSpans() error {
	if len(mp.LogBodies) > 0 {
		return errors.New("log_bodies should not be specified for trace spans")
	}

	if len(mp.LogSeverityTexts) > 0 {
		return errors.New("log_severity_texts should not be specified for trace spans")
	}

	if mp.LogSeverityNumber != nil {
		return errors.New("log_severity_number should not be specified for trace spans")
	}

	if len(mp.MetricNames) > 0 {
		return errors.New("metric_names should not be specified for trace spans")
	}

	if len(mp.Services) == 0 && len(mp.SpanNames) == 0 && len(mp.Attributes) == 0 &&
		len(mp.Libraries) == 0 && len(mp.Resources) == 0 && len(mp.SpanKinds) == 0 {
		return ErrMissingRequiredSpanField
	}

	if len(mp.SpanKinds) > 0 && mp.MatchType == "strict" {
		for _, kind := range mp.SpanKinds {
			if !spanKinds[kind] {
				validSpanKinds := make([]string, len(spanKinds))
				for k := range spanKinds {
					validSpanKinds = append(validSpanKinds, k)
				}
				sort.Strings(validSpanKinds)
				return fmt.Errorf("span_kinds string must match one of the standard span kinds when match_type=strict: %v", validSpanKinds)
			}
		}
	}

	return nil
}

// ValidateForLogs validates properties for logs.
func (mp *MatchProperties) ValidateForLogs() error {
	if len(mp.SpanNames) > 0 || len(mp.Services) > 0 || len(mp.SpanKinds) > 0 || len(mp.MetricNames) > 0 {
		return ErrInvalidLogField
	}

	if len(mp.Attributes) == 0 && len(mp.Libraries) == 0 &&
		len(mp.Resources) == 0 && len(mp.LogBodies) == 0 &&
		len(mp.LogSeverityTexts) == 0 && mp.LogSeverityNumber == nil &&
		len(mp.SpanKinds) == 0 {
		return ErrMissingRequiredLogField
	}

	return nil
}

// ValidateForMetrics validates properties for metrics.
func (mp *MatchProperties) ValidateForMetrics() error {
	if len(mp.LogBodies) > 0 ||
		len(mp.LogSeverityTexts) > 0 ||
		len(mp.SpanNames) > 0 ||
		len(mp.Services) > 0 ||
		len(mp.SpanKinds) > 0 ||
		len(mp.Attributes) > 0 ||
		len(mp.Libraries) > 0 ||
		mp.LogSeverityNumber != nil {
		return ErrInvalidMetricField
	}

	if len(mp.MetricNames) == 0 && len(mp.Resources) == 0 {
		return ErrMissingRequiredMetricField
	}

	return nil
}

// Attribute specifies the attribute key and optional value to match against.
type Attribute struct {
	// Key specifies the attribute key.
	Key string `mapstructure:"key"`

	// Values specifies the value to match against.
	// If it is not set, any value will match.
	Value any `mapstructure:"value"`
}

// InstrumentationLibrary specifies the instrumentation library and optional version to match against.
type InstrumentationLibrary struct {
	Name string `mapstructure:"name"`
	// version match
	//  expected actual  match
	//  nil      <blank> yes
	//  nil      1       yes
	//  <blank>  <blank> yes
	//  <blank>  1       no
	//  1        <blank> no
	//  1        1       yes
	Version *string `mapstructure:"version"`
}

// LogSeverityNumberMatchProperties defines how to match based on a log record's SeverityNumber field.
type LogSeverityNumberMatchProperties struct {
	// Min is the lowest severity that may be matched.
	// e.g. if this is plog.SeverityNumberInfo, INFO, WARN, ERROR, and FATAL logs will match.
	Min plog.SeverityNumber `mapstructure:"min"`

	// MatchUndefined controls whether logs with "undefined" severity matches.
	// If this is true, entries with undefined severity will match.
	MatchUndefined bool `mapstructure:"match_undefined"`
}

// MetricMatchType specifies the strategy for matching against `pmetric.Metric`s. This
// is distinct from filterset.MatchType which matches against metric (and
// tracing) names only. To support matching against metric names and
// `pmetric.Metric`s, filtermetric.MatchType is effectively a superset of
// filterset.MatchType.
type MetricMatchType string

// These are the MetricMatchType that users can specify for filtering
// `pmetric.Metric`s.
const (
	MetricRegexp = MetricMatchType(filterset.Regexp)
	MetricStrict = MetricMatchType(filterset.Strict)
	MetricExpr   = "expr"
)

// MetricMatchProperties specifies the set of properties in a metric to match against and the
// type of string pattern matching to use.
type MetricMatchProperties struct {
	// MatchType specifies the type of matching desired
	MatchType MetricMatchType `mapstructure:"match_type"`
	// RegexpConfig specifies options for the MetricRegexp match type
	RegexpConfig *regexp.Config `mapstructure:"regexp"`

	// MetricNames specifies the list of string patterns to match metric names against.
	// A match occurs if the metric name matches at least one string pattern in this list.
	MetricNames []string `mapstructure:"metric_names"`

	// Expressions specifies the list of expr expressions to match metrics against.
	// A match occurs if any datapoint in a metric matches at least one expression in this list.
	Expressions []string `mapstructure:"expressions"`

	// ResourceAttributes defines a list of possible resource attributes to match metrics against.
	// A match occurs if any resource attribute matches all expressions in this given list.
	ResourceAttributes []Attribute `mapstructure:"resource_attributes"`
}

func CreateMetricMatchPropertiesFromDefault(properties *MatchProperties) (*MetricMatchProperties, error) {
	if properties == nil {
		return nil, nil
	}

	if err := properties.ValidateForMetrics(); err != nil {
		return nil, err
	}

	return &MetricMatchProperties{
		MatchType:          MetricMatchType(properties.MatchType),
		RegexpConfig:       properties.RegexpConfig,
		MetricNames:        properties.MetricNames,
		ResourceAttributes: properties.Resources,
	}, nil
}
