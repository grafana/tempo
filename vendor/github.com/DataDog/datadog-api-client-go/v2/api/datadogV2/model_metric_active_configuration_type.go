// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricActiveConfigurationType The metric actively queried configuration resource type.
type MetricActiveConfigurationType string

// List of MetricActiveConfigurationType.
const (
	METRICACTIVECONFIGURATIONTYPE_ACTIVELY_QUERIED_CONFIGURATIONS MetricActiveConfigurationType = "actively_queried_configurations"
)

var allowedMetricActiveConfigurationTypeEnumValues = []MetricActiveConfigurationType{
	METRICACTIVECONFIGURATIONTYPE_ACTIVELY_QUERIED_CONFIGURATIONS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricActiveConfigurationType) GetAllowedValues() []MetricActiveConfigurationType {
	return allowedMetricActiveConfigurationTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricActiveConfigurationType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricActiveConfigurationType(value)
	return nil
}

// NewMetricActiveConfigurationTypeFromValue returns a pointer to a valid MetricActiveConfigurationType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricActiveConfigurationTypeFromValue(v string) (*MetricActiveConfigurationType, error) {
	ev := MetricActiveConfigurationType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricActiveConfigurationType: valid values are %v", v, allowedMetricActiveConfigurationTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricActiveConfigurationType) IsValid() bool {
	for _, existing := range allowedMetricActiveConfigurationTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricActiveConfigurationType value.
func (v MetricActiveConfigurationType) Ptr() *MetricActiveConfigurationType {
	return &v
}

// NullableMetricActiveConfigurationType handles when a null is used for MetricActiveConfigurationType.
type NullableMetricActiveConfigurationType struct {
	value *MetricActiveConfigurationType
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricActiveConfigurationType) Get() *MetricActiveConfigurationType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricActiveConfigurationType) Set(val *MetricActiveConfigurationType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricActiveConfigurationType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricActiveConfigurationType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricActiveConfigurationType initializes the struct as if Set has been called.
func NewNullableMetricActiveConfigurationType(val *MetricActiveConfigurationType) *NullableMetricActiveConfigurationType {
	return &NullableMetricActiveConfigurationType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricActiveConfigurationType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricActiveConfigurationType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
