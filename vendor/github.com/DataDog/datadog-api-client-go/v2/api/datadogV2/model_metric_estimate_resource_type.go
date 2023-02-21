// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricEstimateResourceType The metric estimate resource type.
type MetricEstimateResourceType string

// List of MetricEstimateResourceType.
const (
	METRICESTIMATERESOURCETYPE_METRIC_CARDINALITY_ESTIMATE MetricEstimateResourceType = "metric_cardinality_estimate"
)

var allowedMetricEstimateResourceTypeEnumValues = []MetricEstimateResourceType{
	METRICESTIMATERESOURCETYPE_METRIC_CARDINALITY_ESTIMATE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricEstimateResourceType) GetAllowedValues() []MetricEstimateResourceType {
	return allowedMetricEstimateResourceTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricEstimateResourceType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricEstimateResourceType(value)
	return nil
}

// NewMetricEstimateResourceTypeFromValue returns a pointer to a valid MetricEstimateResourceType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricEstimateResourceTypeFromValue(v string) (*MetricEstimateResourceType, error) {
	ev := MetricEstimateResourceType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricEstimateResourceType: valid values are %v", v, allowedMetricEstimateResourceTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricEstimateResourceType) IsValid() bool {
	for _, existing := range allowedMetricEstimateResourceTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricEstimateResourceType value.
func (v MetricEstimateResourceType) Ptr() *MetricEstimateResourceType {
	return &v
}

// NullableMetricEstimateResourceType handles when a null is used for MetricEstimateResourceType.
type NullableMetricEstimateResourceType struct {
	value *MetricEstimateResourceType
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricEstimateResourceType) Get() *MetricEstimateResourceType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricEstimateResourceType) Set(val *MetricEstimateResourceType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricEstimateResourceType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricEstimateResourceType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricEstimateResourceType initializes the struct as if Set has been called.
func NewNullableMetricEstimateResourceType(val *MetricEstimateResourceType) *NullableMetricEstimateResourceType {
	return &NullableMetricEstimateResourceType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricEstimateResourceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricEstimateResourceType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
