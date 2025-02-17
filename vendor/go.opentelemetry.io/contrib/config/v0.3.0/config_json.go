// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config/v0.3.0"

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// MarshalJSON implements json.Marshaler.
func (j *AttributeNameValueType) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Value)
}

var enumValuesAttributeNameValueType = []interface{}{
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

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeNameValueType) UnmarshalJSON(b []byte) error {
	var v struct {
		Value interface{}
	}
	if err := json.Unmarshal(b, &v.Value); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValuesAttributeNameValueType {
		if reflect.DeepEqual(v.Value, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValuesAttributeNameValueType, v.Value)
	}
	*j = AttributeNameValueType(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in BatchLogRecordProcessor: required")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in BatchSpanProcessor: required")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *GeneralInstrumentationPeerServiceMappingElem) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["peer"]; raw != nil && !ok {
		return errors.New("field peer in GeneralInstrumentationPeerServiceMappingElem: required")
	}
	if _, ok := raw["service"]; raw != nil && !ok {
		return errors.New("field service in GeneralInstrumentationPeerServiceMappingElem: required")
	}
	type Plain GeneralInstrumentationPeerServiceMappingElem
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = GeneralInstrumentationPeerServiceMappingElem(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *NameStringValuePair) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; !ok {
		return errors.New("json: cannot unmarshal field name in NameStringValuePair required")
	}
	if _, ok := raw["value"]; !ok {
		return errors.New("json: cannot unmarshal field value in NameStringValuePair required")
	}
	var name, value string
	var ok bool
	if name, ok = raw["name"].(string); !ok {
		return errors.New("yaml: cannot unmarshal field name in NameStringValuePair must be string")
	}
	if value, ok = raw["value"].(string); !ok {
		return errors.New("yaml: cannot unmarshal field value in NameStringValuePair must be string")
	}

	*j = NameStringValuePair{
		Name:  name,
		Value: &value,
	}
	return nil
}

var enumValuesOTLPMetricDefaultHistogramAggregation = []interface{}{
	"explicit_bucket_histogram",
	"base2_exponential_bucket_histogram",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPMetricDefaultHistogramAggregation) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValuesOTLPMetricDefaultHistogramAggregation {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValuesOTLPMetricDefaultHistogramAggregation, v)
	}
	*j = OTLPMetricDefaultHistogramAggregation(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPMetric) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return errors.New("field endpoint in OTLPMetric: required")
	}
	if _, ok := raw["protocol"]; raw != nil && !ok {
		return errors.New("field protocol in OTLPMetric: required")
	}
	type Plain OTLPMetric
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OTLPMetric(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLP) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return errors.New("field endpoint in OTLP: required")
	}
	if _, ok := raw["protocol"]; raw != nil && !ok {
		return errors.New("field protocol in OTLP: required")
	}
	type Plain OTLP
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OTLP(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OpenTelemetryConfiguration) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["file_format"]; raw != nil && !ok {
		return errors.New("field file_format in OpenTelemetryConfiguration: required")
	}
	type Plain OpenTelemetryConfiguration
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OpenTelemetryConfiguration(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in PeriodicMetricReader: required")
	}
	type Plain PeriodicMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PeriodicMetricReader(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PullMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in PullMetricReader: required")
	}
	type Plain PullMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PullMetricReader(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in SimpleLogRecordProcessor: required")
	}
	type Plain SimpleLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SimpleLogRecordProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in SimpleSpanProcessor: required")
	}
	type Plain SimpleSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SimpleSpanProcessor(plain)
	return nil
}

var enumValuesViewSelectorInstrumentType = []interface{}{
	"counter",
	"histogram",
	"observable_counter",
	"observable_gauge",
	"observable_up_down_counter",
	"up_down_counter",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ViewSelectorInstrumentType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValuesViewSelectorInstrumentType {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValuesViewSelectorInstrumentType, v)
	}
	*j = ViewSelectorInstrumentType(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Zipkin) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return errors.New("field endpoint in Zipkin: required")
	}
	type Plain Zipkin
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Zipkin(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeNameValue) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return errors.New("field name in AttributeNameValue: required")
	}
	if _, ok := raw["value"]; raw != nil && !ok {
		return errors.New("field value in AttributeNameValue: required")
	}
	type Plain AttributeNameValue
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Type != nil && plain.Type.Value == "int" {
		val, ok := plain.Value.(float64)
		if ok {
			plain.Value = int(val)
		}
	}
	if plain.Type != nil && plain.Type.Value == "int_array" {
		m, ok := plain.Value.([]interface{})
		if ok {
			var vals []interface{}
			for _, v := range m {
				val, ok := v.(float64)
				if ok {
					vals = append(vals, int(val))
				} else {
					vals = append(vals, val)
				}
			}
			plain.Value = vals
		}
	}

	*j = AttributeNameValue(plain)
	return nil
}
