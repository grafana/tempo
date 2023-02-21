// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricBulkConfigureTagsType The metric bulk configure tags resource.
type MetricBulkConfigureTagsType string

// List of MetricBulkConfigureTagsType.
const (
	METRICBULKCONFIGURETAGSTYPE_BULK_MANAGE_TAGS MetricBulkConfigureTagsType = "metric_bulk_configure_tags"
)

var allowedMetricBulkConfigureTagsTypeEnumValues = []MetricBulkConfigureTagsType{
	METRICBULKCONFIGURETAGSTYPE_BULK_MANAGE_TAGS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricBulkConfigureTagsType) GetAllowedValues() []MetricBulkConfigureTagsType {
	return allowedMetricBulkConfigureTagsTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricBulkConfigureTagsType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricBulkConfigureTagsType(value)
	return nil
}

// NewMetricBulkConfigureTagsTypeFromValue returns a pointer to a valid MetricBulkConfigureTagsType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricBulkConfigureTagsTypeFromValue(v string) (*MetricBulkConfigureTagsType, error) {
	ev := MetricBulkConfigureTagsType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricBulkConfigureTagsType: valid values are %v", v, allowedMetricBulkConfigureTagsTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricBulkConfigureTagsType) IsValid() bool {
	for _, existing := range allowedMetricBulkConfigureTagsTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricBulkConfigureTagsType value.
func (v MetricBulkConfigureTagsType) Ptr() *MetricBulkConfigureTagsType {
	return &v
}

// NullableMetricBulkConfigureTagsType handles when a null is used for MetricBulkConfigureTagsType.
type NullableMetricBulkConfigureTagsType struct {
	value *MetricBulkConfigureTagsType
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricBulkConfigureTagsType) Get() *MetricBulkConfigureTagsType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricBulkConfigureTagsType) Set(val *MetricBulkConfigureTagsType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricBulkConfigureTagsType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricBulkConfigureTagsType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricBulkConfigureTagsType initializes the struct as if Set has been called.
func NewNullableMetricBulkConfigureTagsType(val *MetricBulkConfigureTagsType) *NullableMetricBulkConfigureTagsType {
	return &NullableMetricBulkConfigureTagsType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricBulkConfigureTagsType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricBulkConfigureTagsType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
