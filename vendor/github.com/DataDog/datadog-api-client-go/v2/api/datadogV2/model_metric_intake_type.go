// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricIntakeType The type of metric. The available types are `0` (unspecified), `1` (count), `2` (rate), and `3` (gauge).
type MetricIntakeType int32

// List of MetricIntakeType.
const (
	METRICINTAKETYPE_UNSPECIFIED MetricIntakeType = 0
	METRICINTAKETYPE_COUNT       MetricIntakeType = 1
	METRICINTAKETYPE_RATE        MetricIntakeType = 2
	METRICINTAKETYPE_GAUGE       MetricIntakeType = 3
)

var allowedMetricIntakeTypeEnumValues = []MetricIntakeType{
	METRICINTAKETYPE_UNSPECIFIED,
	METRICINTAKETYPE_COUNT,
	METRICINTAKETYPE_RATE,
	METRICINTAKETYPE_GAUGE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricIntakeType) GetAllowedValues() []MetricIntakeType {
	return allowedMetricIntakeTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricIntakeType) UnmarshalJSON(src []byte) error {
	var value int32
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricIntakeType(value)
	return nil
}

// NewMetricIntakeTypeFromValue returns a pointer to a valid MetricIntakeType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricIntakeTypeFromValue(v int32) (*MetricIntakeType, error) {
	ev := MetricIntakeType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricIntakeType: valid values are %v", v, allowedMetricIntakeTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricIntakeType) IsValid() bool {
	for _, existing := range allowedMetricIntakeTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricIntakeType value.
func (v MetricIntakeType) Ptr() *MetricIntakeType {
	return &v
}

// NullableMetricIntakeType handles when a null is used for MetricIntakeType.
type NullableMetricIntakeType struct {
	value *MetricIntakeType
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricIntakeType) Get() *MetricIntakeType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricIntakeType) Set(val *MetricIntakeType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricIntakeType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricIntakeType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricIntakeType initializes the struct as if Set has been called.
func NewNullableMetricIntakeType(val *MetricIntakeType) *NullableMetricIntakeType {
	return &NullableMetricIntakeType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricIntakeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricIntakeType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
