// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"errors"
	"fmt"
	"reflect"

	"go.yaml.in/yaml/v3"
)

// hasYAMLMapKey reports whether the provided mapping node contains the given
// key. It assumes the node is a mapping node and performs a linear scan of its
// key nodes.
func hasYAMLMapKey(node *yaml.Node, key string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return true
		}
	}
	return false
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *ExperimentalResourceDetector) UnmarshalYAML(node *yaml.Node) error {
	type Plain ExperimentalResourceDetector
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if hasYAMLMapKey(node, "gcp") && plain.GCP == nil {
		plain.GCP = ExperimentalGCPResourceDetector{}
	}
	if hasYAMLMapKey(node, "aws.ecs") && plain.AWSECS == nil {
		plain.AWSECS = ExperimentalAWSECSResourceDetector{}
	}
	if hasYAMLMapKey(node, "aws.eks") && plain.AWSEKS == nil {
		plain.AWSEKS = ExperimentalAWSEKSResourceDetector{}
	}
	if hasYAMLMapKey(node, "azure.vm") && plain.AzureVM == nil {
		plain.AzureVM = ExperimentalAzureVMResourceDetector{}
	}
	// container can be nil, must check and set here
	if hasYAMLMapKey(node, "container") && plain.Container == nil {
		plain.Container = ExperimentalContainerResourceDetector{}
	}
	// host can be nil, must check and set here
	if hasYAMLMapKey(node, "host") && plain.Host == nil {
		plain.Host = ExperimentalHostResourceDetector{}
	}
	// process can be nil, must check and set here
	if hasYAMLMapKey(node, "process") && plain.Process == nil {
		plain.Process = ExperimentalProcessResourceDetector{}
	}
	// service can be nil, must check and set here
	if hasYAMLMapKey(node, "service") && plain.Service == nil {
		plain.Service = ExperimentalServiceResourceDetector{}
	}
	*j = ExperimentalResourceDetector(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *PushMetricExporter) UnmarshalYAML(node *yaml.Node) error {
	type Plain PushMetricExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// console can be nil, must check and set here
	if hasYAMLMapKey(node, "console") && plain.Console == nil {
		plain.Console = &ConsoleMetricExporter{}
	}
	*j = PushMetricExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OpenTelemetryConfiguration) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "file_format") {
		return newErrRequired(j, "file_format")
	}
	type Plain OpenTelemetryConfiguration
	type shadow struct {
		Plain
		LogLevel                   *SeverityNumber              `yaml:"log_level,omitempty"`
		AttributeLimits            *AttributeLimits             `yaml:"attribute_limits,omitempty"`
		Disabled                   *bool                        `yaml:"disabled,omitempty"`
		FileFormat                 string                       `yaml:"file_format"`
		LoggerProvider             *LoggerProvider              `yaml:"logger_provider,omitempty"`
		MeterProvider              *MeterProvider               `yaml:"meter_provider,omitempty"`
		TracerProvider             *TracerProvider              `yaml:"tracer_provider,omitempty"`
		Propagator                 *Propagator                  `yaml:"propagator,omitempty"`
		Resource                   *Resource                    `yaml:"resource,omitempty"`
		InstrumentationDevelopment *ExperimentalInstrumentation `yaml:"instrumentation/development"`
	}
	var sh shadow

	if err := node.Decode(&sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.AttributeLimits != nil {
		sh.Plain.AttributeLimits = sh.AttributeLimits
	}

	sh.Plain.FileFormat = sh.FileFormat
	if sh.Disabled != nil {
		sh.Plain.Disabled = sh.Disabled
	} else {
		// Configure the log level of the internal logger used by the SDK.
		// If omitted, info is used.
		sh.Plain.Disabled = ptr(false)
	}
	if sh.LoggerProvider != nil {
		sh.Plain.LoggerProvider = sh.LoggerProvider
	}
	if sh.MeterProvider != nil {
		sh.Plain.MeterProvider = sh.MeterProvider
	}
	if sh.TracerProvider != nil {
		sh.Plain.TracerProvider = sh.TracerProvider
	}
	if sh.Propagator != nil {
		sh.Plain.Propagator = sh.Propagator
	}
	if sh.Resource != nil {
		sh.Plain.Resource = sh.Resource
	}
	if sh.InstrumentationDevelopment != nil {
		sh.Plain.InstrumentationDevelopment = sh.InstrumentationDevelopment
	}

	if sh.LogLevel != nil {
		sh.Plain.LogLevel = sh.LogLevel
	} else {
		// Configure the log level of the internal logger used by the SDK.
		// If omitted, info is used.
		sh.Plain.LogLevel = ptr(SeverityNumberInfo)
	}

	*j = OpenTelemetryConfiguration(sh.Plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *SpanExporter) UnmarshalYAML(node *yaml.Node) error {
	type Plain SpanExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// console can be nil, must check and set here
	if hasYAMLMapKey(node, "console") && plain.Console == nil {
		plain.Console = ConsoleExporter{}
	}
	*j = SpanExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *LogRecordExporter) UnmarshalYAML(node *yaml.Node) error {
	type Plain LogRecordExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// console can be nil, must check and set here
	if hasYAMLMapKey(node, "console") && plain.Console == nil {
		plain.Console = ConsoleExporter{}
	}
	*j = LogRecordExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *TextMapPropagator) UnmarshalYAML(node *yaml.Node) error {
	type Plain TextMapPropagator
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// b3 can be nil, must check and set here
	if hasYAMLMapKey(node, "b3") && plain.B3 == nil {
		plain.B3 = B3Propagator{}
	}
	// b3multi can be nil, must check and set here
	if hasYAMLMapKey(node, "b3multi") && plain.B3Multi == nil {
		plain.B3Multi = B3MultiPropagator{}
	}
	// baggage can be nil, must check and set here
	if hasYAMLMapKey(node, "baggage") && plain.Baggage == nil {
		plain.Baggage = BaggagePropagator{}
	}
	// jaeger can be nil, must check and set here
	if hasYAMLMapKey(node, "jaeger") && plain.Jaeger == nil {
		plain.Jaeger = JaegerPropagator{}
	}
	// ottrace can be nil, must check and set here
	if hasYAMLMapKey(node, "ottrace") && plain.Ottrace == nil {
		plain.Ottrace = OpenTracingPropagator{}
	}
	// tracecontext can be nil, must check and set here
	if hasYAMLMapKey(node, "tracecontext") && plain.Tracecontext == nil {
		plain.Tracecontext = TraceContextPropagator{}
	}
	*j = TextMapPropagator(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateBatchLogRecordProcessor((*BatchLogRecordProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *Sampler) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	type Plain Sampler
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return err
	}
	unmarshalSamplerTypes(raw, (*Sampler)(&plain))
	*j = Sampler(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *MetricProducer) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	type Plain MetricProducer
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return err
	}
	unmarshalMetricProducer(raw, (*MetricProducer)(&plain))
	*j = MetricProducer(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateBatchSpanProcessor((*BatchSpanProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain PeriodicMetricReader
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validatePeriodicMetricReader((*PeriodicMetricReader)(&plain)); err != nil {
		return err
	}
	*j = PeriodicMetricReader(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *CardinalityLimits) UnmarshalYAML(node *yaml.Node) error {
	type Plain CardinalityLimits
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateCardinalityLimits((*CardinalityLimits)(&plain)); err != nil {
		return err
	}
	*j = CardinalityLimits(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *SpanLimits) UnmarshalYAML(node *yaml.Node) error {
	type Plain SpanLimits
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateSpanLimits((*SpanLimits)(&plain)); err != nil {
		return err
	}
	*j = SpanLimits(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OTLPHttpMetricExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain OTLPHttpMetricExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPHttpMetricExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OTLPGrpcMetricExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain OTLPGrpcMetricExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPGrpcMetricExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OTLPHttpExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain OTLPHttpExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPHttpExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OTLPGrpcExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain OTLPGrpcExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPGrpcExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *AttributeType) UnmarshalYAML(node *yaml.Node) error {
	var v string
	if err := node.Decode(&v); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	var ok bool
	for _, expected := range enumValuesAttributeType {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return newErrInvalid(fmt.Sprintf("unexpected value type %#v, expected one of %#v)", v, enumValuesAttributeType))
	}
	*j = AttributeType(v)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *AttributeNameValue) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "name") {
		return newErrRequired(j, "name")
	}
	if !hasYAMLMapKey(node, "value") {
		return newErrRequired(j, "value")
	}
	type Plain AttributeNameValue
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	// yaml unmarshaller defaults to unmarshalling to int
	if plain.Type != nil && *plain.Type == AttributeTypeDouble {
		val, ok := plain.Value.(int)
		if ok {
			plain.Value = float64(val)
		}
	}

	if plain.Type != nil && *plain.Type == AttributeTypeDoubleArray {
		m, ok := plain.Value.([]any)
		if ok {
			var vals []any
			for _, v := range m {
				val, ok := v.(int)
				if ok {
					vals = append(vals, float64(val))
				} else {
					vals = append(vals, v)
				}
			}
			plain.Value = vals
		}
	}

	*j = AttributeNameValue(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *SimpleLogRecordProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain SimpleLogRecordProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	*j = SimpleLogRecordProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *SimpleSpanProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain SimpleSpanProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	*j = SimpleSpanProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *NameStringValuePair) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "name") {
		return newErrRequired(j, "name")
	}
	if !hasYAMLMapKey(node, "value") {
		return newErrRequired(j, "value")
	}

	type Plain NameStringValuePair
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	*j = NameStringValuePair(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *InstrumentType) UnmarshalYAML(node *yaml.Node) error {
	type Plain InstrumentType
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := supportedInstrumentType(InstrumentType(plain)); err != nil {
		return err
	}

	*j = InstrumentType(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *ExperimentalPeerServiceMapping) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "peer") {
		return newErrRequired(j, "peer")
	}
	if !hasYAMLMapKey(node, "service") {
		return newErrRequired(j, "service")
	}

	type Plain ExperimentalPeerServiceMapping
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	*j = ExperimentalPeerServiceMapping(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *ExporterDefaultHistogramAggregation) UnmarshalYAML(node *yaml.Node) error {
	type Plain ExporterDefaultHistogramAggregation
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := supportedHistogramAggregation(ExporterDefaultHistogramAggregation(plain)); err != nil {
		return err
	}

	*j = ExporterDefaultHistogramAggregation(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *PullMetricReader) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain PullMetricReader
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	*j = PullMetricReader(plain)
	return nil
}
