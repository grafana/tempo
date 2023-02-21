// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricDistinctVolumeType The metric distinct volume type.
type MetricDistinctVolumeType string

// List of MetricDistinctVolumeType.
const (
	METRICDISTINCTVOLUMETYPE_DISTINCT_METRIC_VOLUMES MetricDistinctVolumeType = "distinct_metric_volumes"
)

var allowedMetricDistinctVolumeTypeEnumValues = []MetricDistinctVolumeType{
	METRICDISTINCTVOLUMETYPE_DISTINCT_METRIC_VOLUMES,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricDistinctVolumeType) GetAllowedValues() []MetricDistinctVolumeType {
	return allowedMetricDistinctVolumeTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricDistinctVolumeType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricDistinctVolumeType(value)
	return nil
}

// NewMetricDistinctVolumeTypeFromValue returns a pointer to a valid MetricDistinctVolumeType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricDistinctVolumeTypeFromValue(v string) (*MetricDistinctVolumeType, error) {
	ev := MetricDistinctVolumeType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricDistinctVolumeType: valid values are %v", v, allowedMetricDistinctVolumeTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricDistinctVolumeType) IsValid() bool {
	for _, existing := range allowedMetricDistinctVolumeTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricDistinctVolumeType value.
func (v MetricDistinctVolumeType) Ptr() *MetricDistinctVolumeType {
	return &v
}

// NullableMetricDistinctVolumeType handles when a null is used for MetricDistinctVolumeType.
type NullableMetricDistinctVolumeType struct {
	value *MetricDistinctVolumeType
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricDistinctVolumeType) Get() *MetricDistinctVolumeType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricDistinctVolumeType) Set(val *MetricDistinctVolumeType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricDistinctVolumeType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricDistinctVolumeType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricDistinctVolumeType initializes the struct as if Set has been called.
func NewNullableMetricDistinctVolumeType(val *MetricDistinctVolumeType) *NullableMetricDistinctVolumeType {
	return &NullableMetricDistinctVolumeType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricDistinctVolumeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricDistinctVolumeType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
