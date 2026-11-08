// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// UnmarshalJSON implements json.Unmarshaler.
func (j *ConsoleExporter) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = ConsoleExporter{}
	} else {
		*j = ConsoleExporter(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *B3Propagator) UnmarshalJSON(b []byte) error {
	type plain B3Propagator
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = B3Propagator{}
	} else {
		*j = B3Propagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *B3MultiPropagator) UnmarshalJSON(b []byte) error {
	type plain B3MultiPropagator
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = B3MultiPropagator{}
	} else {
		*j = B3MultiPropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BaggagePropagator) UnmarshalJSON(b []byte) error {
	type plain BaggagePropagator
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = BaggagePropagator{}
	} else {
		*j = BaggagePropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JaegerPropagator) UnmarshalJSON(b []byte) error {
	type plain JaegerPropagator
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = JaegerPropagator{}
	} else {
		*j = JaegerPropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OpenTracingPropagator) UnmarshalJSON(b []byte) error {
	type plain OpenTracingPropagator
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = OpenTracingPropagator{}
	} else {
		*j = OpenTracingPropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *TraceContextPropagator) UnmarshalJSON(b []byte) error {
	type plain TraceContextPropagator
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = TraceContextPropagator{}
	} else {
		*j = TraceContextPropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalGCPResourceDetector) UnmarshalJSON(b []byte) error {
	type plain ExperimentalGCPResourceDetector
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if p == nil {
		*j = ExperimentalGCPResourceDetector{}
	} else {
		*j = ExperimentalGCPResourceDetector(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalAWSECSResourceDetector) UnmarshalJSON(b []byte) error {
	type plain ExperimentalAWSECSResourceDetector
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if p == nil {
		*j = ExperimentalAWSECSResourceDetector{}
	} else {
		*j = ExperimentalAWSECSResourceDetector(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalAWSEKSResourceDetector) UnmarshalJSON(b []byte) error {
	type plain ExperimentalAWSEKSResourceDetector
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if p == nil {
		*j = ExperimentalAWSEKSResourceDetector{}
	} else {
		*j = ExperimentalAWSEKSResourceDetector(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalAzureVMResourceDetector) UnmarshalJSON(b []byte) error {
	type plain ExperimentalAzureVMResourceDetector
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if p == nil {
		*j = ExperimentalAzureVMResourceDetector{}
	} else {
		*j = ExperimentalAzureVMResourceDetector(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalContainerResourceDetector) UnmarshalJSON(b []byte) error {
	type plain ExperimentalContainerResourceDetector
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = ExperimentalContainerResourceDetector{}
	} else {
		*j = ExperimentalContainerResourceDetector(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalHostResourceDetector) UnmarshalJSON(b []byte) error {
	type plain ExperimentalHostResourceDetector
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = ExperimentalHostResourceDetector{}
	} else {
		*j = ExperimentalHostResourceDetector(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalProcessResourceDetector) UnmarshalJSON(b []byte) error {
	type plain ExperimentalProcessResourceDetector
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = ExperimentalProcessResourceDetector{}
	} else {
		*j = ExperimentalProcessResourceDetector(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalServiceResourceDetector) UnmarshalJSON(b []byte) error {
	type plain ExperimentalServiceResourceDetector
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = ExperimentalServiceResourceDetector{}
	} else {
		*j = ExperimentalServiceResourceDetector(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalResourceDetector) UnmarshalJSON(b []byte) error {
	// Use a shadow struct with a RawMessage field to detect key presence.
	type Plain ExperimentalResourceDetector
	type shadow struct {
		Plain
		GCP       json.RawMessage `json:"gcp"`
		AWSECS    json.RawMessage `json:"aws.ecs"`
		AWSEKS    json.RawMessage `json:"aws.eks"`
		AzureVM   json.RawMessage `json:"azure.vm"`
		Container json.RawMessage `json:"container"`
		Host      json.RawMessage `json:"host"`
		Process   json.RawMessage `json:"process"`
		Service   json.RawMessage `json:"service"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.GCP != nil {
		var c ExperimentalGCPResourceDetector
		if err := json.Unmarshal(sh.GCP, &c); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		if c == nil {
			c = ExperimentalGCPResourceDetector{}
		}
		sh.Plain.GCP = c
	}

	if sh.AWSECS != nil {
		var c ExperimentalAWSECSResourceDetector
		if err := json.Unmarshal(sh.AWSECS, &c); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		if c == nil {
			c = ExperimentalAWSECSResourceDetector{}
		}
		sh.Plain.AWSECS = c
	}

	if sh.AWSEKS != nil {
		var c ExperimentalAWSEKSResourceDetector
		if err := json.Unmarshal(sh.AWSEKS, &c); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		if c == nil {
			c = ExperimentalAWSEKSResourceDetector{}
		}
		sh.Plain.AWSEKS = c
	}

	if sh.AzureVM != nil {
		var c ExperimentalAzureVMResourceDetector
		if err := json.Unmarshal(sh.AzureVM, &c); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		if c == nil {
			c = ExperimentalAzureVMResourceDetector{}
		}
		sh.Plain.AzureVM = c
	}

	if sh.Container != nil {
		var c ExperimentalContainerResourceDetector
		if err := json.Unmarshal(sh.Container, &c); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Container = c
	}

	if sh.Host != nil {
		var c ExperimentalHostResourceDetector
		if err := json.Unmarshal(sh.Host, &c); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Host = c
	}

	if sh.Process != nil {
		var c ExperimentalProcessResourceDetector
		if err := json.Unmarshal(sh.Process, &c); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Process = c
	}

	if sh.Service != nil {
		var c ExperimentalServiceResourceDetector
		if err := json.Unmarshal(sh.Service, &c); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Service = c
	}
	*j = ExperimentalResourceDetector(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PushMetricExporter) UnmarshalJSON(b []byte) error {
	// Use a shadow struct with a RawMessage field to detect key presence.
	type Plain PushMetricExporter
	type shadow struct {
		Plain
		Console json.RawMessage `json:"console"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.Console != nil {
		var c ConsoleMetricExporter
		if err := json.Unmarshal(sh.Console, &c); err != nil {
			return err
		}
		sh.Plain.Console = &c
	}
	*j = PushMetricExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SpanExporter) UnmarshalJSON(b []byte) error {
	// Use a shadow struct with a RawMessage field to detect key presence.
	type Plain SpanExporter
	type shadow struct {
		Plain
		Console json.RawMessage `json:"console"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.Console != nil {
		var c ConsoleExporter
		if err := json.Unmarshal(sh.Console, &c); err != nil {
			return err
		}
		sh.Plain.Console = c
	}
	*j = SpanExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *LogRecordExporter) UnmarshalJSON(b []byte) error {
	// Use a shadow struct with a RawMessage field to detect key presence.
	type Plain LogRecordExporter
	type shadow struct {
		Plain
		Console json.RawMessage `json:"console"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.Console != nil {
		var c ConsoleExporter
		if err := json.Unmarshal(sh.Console, &c); err != nil {
			return err
		}
		sh.Plain.Console = c
	}
	*j = LogRecordExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *TextMapPropagator) UnmarshalJSON(b []byte) error {
	type Plain TextMapPropagator
	type shadow struct {
		Plain
		B3           json.RawMessage `json:"b3"`
		B3Multi      json.RawMessage `json:"b3multi"`
		Baggage      json.RawMessage `json:"baggage"`
		Jaeger       json.RawMessage `json:"jaeger"`
		Ottrace      json.RawMessage `json:"ottrace"`
		Tracecontext json.RawMessage `json:"tracecontext"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.B3 != nil {
		var p B3Propagator
		if err := json.Unmarshal(sh.B3, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.B3 = p
	}

	if sh.B3Multi != nil {
		var p B3MultiPropagator
		if err := json.Unmarshal(sh.B3Multi, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.B3Multi = p
	}

	if sh.Baggage != nil {
		var p BaggagePropagator
		if err := json.Unmarshal(sh.Baggage, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Baggage = p
	}

	if sh.Jaeger != nil {
		var p JaegerPropagator
		if err := json.Unmarshal(sh.Jaeger, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Jaeger = p
	}

	if sh.Ottrace != nil {
		var p OpenTracingPropagator
		if err := json.Unmarshal(sh.Ottrace, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Ottrace = p
	}

	if sh.Tracecontext != nil {
		var p TraceContextPropagator
		if err := json.Unmarshal(sh.Tracecontext, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Tracecontext = p
	}

	*j = TextMapPropagator(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalJSON(b []byte) error {
	type Plain BatchLogRecordProcessor
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	if err := validateBatchLogRecordProcessor((*BatchLogRecordProcessor)(&sh.Plain)); err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalJSON(b []byte) error {
	type Plain BatchSpanProcessor
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	if err := validateBatchSpanProcessor((*BatchSpanProcessor)(&sh.Plain)); err != nil {
		return err
	}
	*j = BatchSpanProcessor(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OpenTelemetryConfiguration) UnmarshalJSON(b []byte) error {
	type Plain OpenTelemetryConfiguration
	type shadow struct {
		Plain
		FileFormat                 json.RawMessage `json:"file_format"`
		LoggerProvider             json.RawMessage `json:"logger_provider"`
		MeterProvider              json.RawMessage `json:"meter_provider"`
		TracerProvider             json.RawMessage `json:"tracer_provider"`
		Propagator                 json.RawMessage `json:"propagator"`
		Resource                   json.RawMessage `json:"resource"`
		InstrumentationDevelopment json.RawMessage `json:"instrumentation/development"`
		AttributeLimits            json.RawMessage `json:"attribute_limits"`
		Disabled                   json.RawMessage `json:"disabled"`
		LogLevel                   json.RawMessage `json:"log_level"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if len(sh.FileFormat) == 0 {
		return newErrRequired(j, "file_format")
	}

	if err := json.Unmarshal(sh.FileFormat, &sh.Plain.FileFormat); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.LoggerProvider != nil {
		var l LoggerProvider
		if err := json.Unmarshal(sh.LoggerProvider, &l); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.LoggerProvider = &l
	}

	if sh.MeterProvider != nil {
		var m MeterProvider
		if err := json.Unmarshal(sh.MeterProvider, &m); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.MeterProvider = &m
	}

	if sh.TracerProvider != nil {
		var t TracerProvider
		if err := json.Unmarshal(sh.TracerProvider, &t); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.TracerProvider = &t
	}

	if sh.Propagator != nil {
		var p Propagator
		if err := json.Unmarshal(sh.Propagator, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Propagator = &p
	}

	if sh.Resource != nil {
		var r Resource
		if err := json.Unmarshal(sh.Resource, &r); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Resource = &r
	}

	if sh.InstrumentationDevelopment != nil {
		var r ExperimentalInstrumentation
		if err := json.Unmarshal(sh.InstrumentationDevelopment, &r); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.InstrumentationDevelopment = &r
	}

	if sh.AttributeLimits != nil {
		var r AttributeLimits
		if err := json.Unmarshal(sh.AttributeLimits, &r); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.AttributeLimits = &r
	}

	if sh.Disabled != nil {
		if err := json.Unmarshal(sh.Disabled, &sh.Plain.Disabled); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
	} else {
		// Configure if the SDK is disabled or not.
		// If omitted or null, false is used.
		sh.Plain.Disabled = ptr(false)
	}

	if sh.LogLevel != nil {
		if err := json.Unmarshal(sh.LogLevel, &sh.Plain.LogLevel); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
	} else {
		// Configure the log level of the internal logger used by the SDK.
		// If omitted, info is used.
		sh.Plain.LogLevel = ptr(SeverityNumberInfo)
	}

	*j = OpenTelemetryConfiguration(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalJSON(b []byte) error {
	type Plain PeriodicMetricReader
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	err := validatePeriodicMetricReader((*PeriodicMetricReader)(&sh.Plain))
	if err != nil {
		return err
	}
	*j = PeriodicMetricReader(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CardinalityLimits) UnmarshalJSON(value []byte) error {
	type Plain CardinalityLimits
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateCardinalityLimits((*CardinalityLimits)(&plain)); err != nil {
		return err
	}
	*j = CardinalityLimits(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SpanLimits) UnmarshalJSON(value []byte) error {
	type Plain SpanLimits
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateSpanLimits((*SpanLimits)(&plain)); err != nil {
		return err
	}
	*j = SpanLimits(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPHttpMetricExporter) UnmarshalJSON(b []byte) error {
	type Plain OTLPHttpMetricExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}
	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPHttpMetricExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPGrpcMetricExporter) UnmarshalJSON(b []byte) error {
	type Plain OTLPGrpcMetricExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}
	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPGrpcMetricExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPHttpExporter) UnmarshalJSON(b []byte) error {
	type Plain OTLPHttpExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}
	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPHttpExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPGrpcExporter) UnmarshalJSON(b []byte) error {
	type Plain OTLPGrpcExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}
	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPGrpcExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
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

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeNameValue) UnmarshalJSON(b []byte) error {
	type Plain AttributeNameValue
	type shadow struct {
		Plain
		Name  json.RawMessage `json:"name"`
		Value json.RawMessage `json:"value"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Name == nil {
		return newErrRequired(j, "name")
	}
	if err := json.Unmarshal(sh.Name, &sh.Plain.Name); err != nil {
		return err
	}
	if sh.Value == nil {
		return newErrRequired(j, "value")
	}
	if err := json.Unmarshal(sh.Value, &sh.Plain.Value); err != nil {
		return err
	}

	// json unmarshaller defaults to unmarshalling to float for int values
	if sh.Type != nil && *sh.Type == AttributeTypeInt {
		val, ok := sh.Plain.Value.(float64)
		if ok {
			sh.Plain.Value = int(val)
		}
	}

	if sh.Type != nil && *sh.Type == AttributeTypeIntArray {
		m, ok := sh.Plain.Value.([]any)
		if ok {
			var vals []any
			for _, v := range m {
				val, ok := v.(float64)
				if ok {
					vals = append(vals, int(val))
				} else {
					vals = append(vals, v)
				}
			}
			sh.Plain.Value = vals
		}
	}

	*j = AttributeNameValue(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleLogRecordProcessor) UnmarshalJSON(b []byte) error {
	type Plain SimpleLogRecordProcessor
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	*j = SimpleLogRecordProcessor(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleSpanProcessor) UnmarshalJSON(b []byte) error {
	type Plain SimpleSpanProcessor
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	*j = SimpleSpanProcessor(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *NameStringValuePair) UnmarshalJSON(b []byte) error {
	type Plain NameStringValuePair
	type shadow struct {
		Plain
		Name  json.RawMessage `json:"name"`
		Value json.RawMessage `json:"value"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Name == nil {
		return newErrRequired(j, "name")
	}
	if err := json.Unmarshal(sh.Name, &sh.Plain.Name); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Value == nil {
		return newErrRequired(j, "value")
	}
	if err := json.Unmarshal(sh.Value, &sh.Plain.Value); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	*j = NameStringValuePair(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InstrumentType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := supportedInstrumentType(InstrumentType(v)); err != nil {
		return err
	}
	*j = InstrumentType(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalPeerServiceMapping) UnmarshalJSON(b []byte) error {
	type Plain ExperimentalPeerServiceMapping
	type shadow struct {
		Plain
		Peer    json.RawMessage `json:"peer"`
		Service json.RawMessage `json:"service"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Peer == nil {
		return newErrRequired(j, "peer")
	}
	if err := json.Unmarshal(sh.Peer, &sh.Plain.Peer); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Service == nil {
		return newErrRequired(j, "service")
	}
	if err := json.Unmarshal(sh.Service, &sh.Plain.Service); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	*j = ExperimentalPeerServiceMapping(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExporterDefaultHistogramAggregation) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := supportedHistogramAggregation(ExporterDefaultHistogramAggregation(v)); err != nil {
		return err
	}
	*j = ExporterDefaultHistogramAggregation(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PullMetricReader) UnmarshalJSON(b []byte) error {
	type Plain PullMetricReader
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	*j = PullMetricReader(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Sampler) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain Sampler
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	unmarshalSamplerTypes(raw, (*Sampler)(&plain))
	*j = Sampler(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *MetricProducer) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain MetricProducer
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	unmarshalMetricProducer(raw, (*MetricProducer)(&plain))
	*j = MetricProducer(plain)
	return nil
}
