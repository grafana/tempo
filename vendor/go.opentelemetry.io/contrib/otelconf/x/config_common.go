// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel/baggage"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	compressionGzip = "gzip"
	compressionNone = "none"
)

var enumValuesAttributeType = []any{
	nil,
	"string",
	"bool",
	"int",
	"double",
	"string_array",
	"bool_array",
	"int_array",
	"double_array",
}

var enumValuesViewSelectorInstrumentType = []any{
	"counter",
	"gauge",
	"histogram",
	"observable_counter",
	"observable_gauge",
	"observable_up_down_counter",
	"up_down_counter",
}

var enumValuesOTLPMetricDefaultHistogramAggregation = []any{
	"explicit_bucket_histogram",
	"base2_exponential_bucket_histogram",
}

type configOptions struct {
	ctx                   context.Context
	opentelemetryConfig   OpenTelemetryConfiguration
	loggerProviderOptions []sdklog.LoggerProviderOption
	meterProviderOptions  []sdkmetric.Option
	tracerProviderOptions []sdktrace.TracerProviderOption
}

type shutdownFunc func(context.Context) error

func noopShutdown(context.Context) error {
	return nil
}

type errBound struct {
	Field string
	Bound int
	Op    string
}

func (e *errBound) Error() string {
	return fmt.Sprintf("field %s: must be %s %d", e.Field, e.Op, e.Bound)
}

func (e *errBound) Is(target error) bool {
	t, ok := target.(*errBound)
	if !ok {
		return false
	}
	return e.Field == t.Field && e.Bound == t.Bound && e.Op == t.Op
}

type errRequired struct {
	Object any
	Field  string
}

func (e *errRequired) Error() string {
	return fmt.Sprintf("field %s in %s: required", e.Field, reflect.TypeOf(e.Object))
}

func (e *errRequired) Is(target error) bool {
	t, ok := target.(*errRequired)
	if !ok {
		return false
	}
	return reflect.TypeOf(e.Object) == reflect.TypeOf(t.Object) && e.Field == t.Field
}

type errUnmarshal struct {
	Object any
}

func (e *errUnmarshal) Error() string {
	return fmt.Sprintf("unmarshal error in %T", e.Object)
}

func (e *errUnmarshal) Is(target error) bool {
	t, ok := target.(*errUnmarshal)
	if !ok {
		return false
	}
	return reflect.TypeOf(e.Object) == reflect.TypeOf(t.Object)
}

// newErrGreaterOrEqualZero creates a new error indicating that the field must be greater than
// or equal to zero.
func newErrGreaterOrEqualZero(field string) error {
	return &errBound{Field: field, Bound: 0, Op: ">="}
}

// newErrGreaterThanZero creates a new error indicating that the field must be greater
// than zero.
func newErrGreaterThanZero(field string) error {
	return &errBound{Field: field, Bound: 0, Op: ">"}
}

// newErrRequired creates a new error indicating that the exporter field is required.
func newErrRequired(object any, field string) error {
	return &errRequired{Object: object, Field: field}
}

// newErrUnmarshal creates a new error indicating that an error occurred during unmarshaling.
func newErrUnmarshal(object any) error {
	return &errUnmarshal{Object: object}
}

type errInvalid struct {
	Identifier string
}

func (e *errInvalid) Error() string {
	return "invalid config: " + e.Identifier
}

func (*errInvalid) Is(target error) bool {
	_, ok := target.(*errInvalid)
	return ok
}

// newErrInvalid creates a new error indicating that an error occurred due to misconfiguration.
func newErrInvalid(id string) error {
	return &errInvalid{Identifier: id}
}

// unmarshalSamplerTypes handles always_on and always_off sampler unmarshaling.
func unmarshalSamplerTypes(raw map[string]any, plain *Sampler) {
	// always_on can be nil, must check and set here
	if _, ok := raw["always_on"]; ok {
		plain.AlwaysOn = AlwaysOnSampler{}
	}
	// always_off can be nil, must check and set here
	if _, ok := raw["always_off"]; ok {
		plain.AlwaysOff = AlwaysOffSampler{}
	}
}

// unmarshalMetricProducer handles opencensus metric producer unmarshaling.
func unmarshalMetricProducer(raw map[string]any, plain *MetricProducer) {
	// opencensus can be nil, must check and set here
	if v, ok := raw["opencensus"]; ok && v == nil {
		delete(raw, "opencensus")
		plain.Opencensus = OpenCensusMetricProducer{}
	}
	if len(raw) > 0 {
		plain.AdditionalProperties = raw
	}
}

// validatePeriodicMetricReader handles validation for PeriodicMetricReader.
func validatePeriodicMetricReader(plain *PeriodicMetricReader) error {
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	if plain.Interval != nil && 0 > *plain.Interval {
		return newErrGreaterOrEqualZero("interval")
	}
	return nil
}

// validateBatchLogRecordProcessor handles validation for BatchLogRecordProcessor.
func validateBatchLogRecordProcessor(plain *BatchLogRecordProcessor) error {
	if plain.ExportTimeout != nil && 0 > *plain.ExportTimeout {
		return newErrGreaterOrEqualZero("export_timeout")
	}
	if plain.MaxExportBatchSize != nil && 0 >= *plain.MaxExportBatchSize {
		return newErrGreaterThanZero("max_export_batch_size")
	}
	if plain.MaxQueueSize != nil && 0 >= *plain.MaxQueueSize {
		return newErrGreaterThanZero("max_queue_size")
	}
	if plain.ScheduleDelay != nil && 0 > *plain.ScheduleDelay {
		return newErrGreaterOrEqualZero("schedule_delay")
	}
	return nil
}

// validateBatchSpanProcessor handles validation for BatchSpanProcessor.
func validateBatchSpanProcessor(plain *BatchSpanProcessor) error {
	if plain.ExportTimeout != nil && 0 > *plain.ExportTimeout {
		return newErrGreaterOrEqualZero("export_timeout")
	}
	if plain.MaxExportBatchSize != nil && 0 >= *plain.MaxExportBatchSize {
		return newErrGreaterThanZero("max_export_batch_size")
	}
	if plain.MaxQueueSize != nil && 0 >= *plain.MaxQueueSize {
		return newErrGreaterThanZero("max_queue_size")
	}
	if plain.ScheduleDelay != nil && 0 > *plain.ScheduleDelay {
		return newErrGreaterOrEqualZero("schedule_delay")
	}
	return nil
}

// validateCardinalityLimits handles validation for CardinalityLimits.
func validateCardinalityLimits(plain *CardinalityLimits) error {
	if plain.Counter != nil && 0 >= *plain.Counter {
		return newErrGreaterThanZero("counter")
	}
	if plain.Default != nil && 0 >= *plain.Default {
		return newErrGreaterThanZero("default")
	}
	if plain.Gauge != nil && 0 >= *plain.Gauge {
		return newErrGreaterThanZero("gauge")
	}
	if plain.Histogram != nil && 0 >= *plain.Histogram {
		return newErrGreaterThanZero("histogram")
	}
	if plain.ObservableCounter != nil && 0 >= *plain.ObservableCounter {
		return newErrGreaterThanZero("observable_counter")
	}
	if plain.ObservableGauge != nil && 0 >= *plain.ObservableGauge {
		return newErrGreaterThanZero("observable_gauge")
	}
	if plain.ObservableUpDownCounter != nil && 0 >= *plain.ObservableUpDownCounter {
		return newErrGreaterThanZero("observable_up_down_counter")
	}
	if plain.UpDownCounter != nil && 0 >= *plain.UpDownCounter {
		return newErrGreaterThanZero("up_down_counter")
	}
	return nil
}

// validateSpanLimits handles validation for SpanLimits.
func validateSpanLimits(plain *SpanLimits) error {
	if plain.AttributeCountLimit != nil && 0 > *plain.AttributeCountLimit {
		return newErrGreaterOrEqualZero("attribute_count_limit")
	}
	if plain.AttributeValueLengthLimit != nil && 0 > *plain.AttributeValueLengthLimit {
		return newErrGreaterOrEqualZero("attribute_value_length_limit")
	}
	if plain.EventAttributeCountLimit != nil && 0 > *plain.EventAttributeCountLimit {
		return newErrGreaterOrEqualZero("event_attribute_count_limit")
	}
	if plain.EventCountLimit != nil && 0 > *plain.EventCountLimit {
		return newErrGreaterOrEqualZero("event_count_limit")
	}
	if plain.LinkAttributeCountLimit != nil && 0 > *plain.LinkAttributeCountLimit {
		return newErrGreaterOrEqualZero("link_attribute_count_limit")
	}
	if plain.LinkCountLimit != nil && 0 > *plain.LinkCountLimit {
		return newErrGreaterOrEqualZero("link_count_limit")
	}
	return nil
}

func ptr[T any](v T) *T {
	return &v
}

// validateOTLPHTTPEncoding validates the encoding configuration.
// The Go SDK only supports protobuf encoding for OTLP HTTP exporters.
func validateOTLPHTTPEncoding(encoding *OTLPHttpEncoding) error {
	if encoding != nil && *encoding != OTLPHttpEncodingProtobuf {
		return newErrInvalid(fmt.Sprintf("unsupported encoding %q", *encoding))
	}
	return nil
}

// createHeadersConfig combines the two header config fields. Headers take precedence over headersList.
func createHeadersConfig(headers []NameStringValuePair, headersList *string) (map[string]string, error) {
	result := make(map[string]string)
	if headersList != nil {
		// Parsing follows https://github.com/open-telemetry/opentelemetry-configuration/blob/568e5080816d40d75792eb754fc96bde09654159/schema/type_descriptions.yaml#L584.
		headerslist, err := baggage.Parse(*headersList)
		if err != nil {
			return nil, errors.Join(newErrInvalid("invalid headers_list"), err)
		}
		for _, kv := range headerslist.Members() {
			result[kv.Key()] = kv.Value()
		}
	}
	// Headers take precedence over HeadersList, so this has to be after HeadersList is processed.
	for _, kv := range headers {
		if kv.Value != nil {
			result[kv.Name] = *kv.Value
		}
	}
	return result, nil
}

// supportedInstrumentType return an error if the instrument type is not supported.
func supportedInstrumentType(in InstrumentType) error {
	for _, expected := range enumValuesViewSelectorInstrumentType {
		if string(in) == fmt.Sprintf("%s", expected) {
			return nil
		}
	}
	return newErrInvalid(fmt.Sprintf("invalid selector (expected one of %#v): %#v", enumValuesViewSelectorInstrumentType, in))
}

// supportedHistogramAggregation return an error if the histogram aggregation is not supported.
func supportedHistogramAggregation(in ExporterDefaultHistogramAggregation) error {
	for _, expected := range enumValuesOTLPMetricDefaultHistogramAggregation {
		if string(in) == fmt.Sprintf("%s", expected) {
			return nil
		}
	}
	return newErrInvalid(fmt.Sprintf("invalid histogram aggregation (expected one of %#v): %#v", enumValuesOTLPMetricDefaultHistogramAggregation, in))
}
