// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricEstimateType Estimate type based on the queried configuration. By default, `count_or_gauge` is returned. `distribution` is returned for distribution metrics without percentiles enabled. Lastly, `percentile` is returned if `filter[pct]=true` is queried with a distribution metric.
type MetricEstimateType string

// List of MetricEstimateType.
const (
	METRICESTIMATETYPE_COUNT_OR_GAUGE MetricEstimateType = "count_or_gauge"
	METRICESTIMATETYPE_DISTRIBUTION   MetricEstimateType = "distribution"
	METRICESTIMATETYPE_PERCENTILE     MetricEstimateType = "percentile"
)

var allowedMetricEstimateTypeEnumValues = []MetricEstimateType{
	METRICESTIMATETYPE_COUNT_OR_GAUGE,
	METRICESTIMATETYPE_DISTRIBUTION,
	METRICESTIMATETYPE_PERCENTILE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricEstimateType) GetAllowedValues() []MetricEstimateType {
	return allowedMetricEstimateTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricEstimateType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricEstimateType(value)
	return nil
}

// NewMetricEstimateTypeFromValue returns a pointer to a valid MetricEstimateType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricEstimateTypeFromValue(v string) (*MetricEstimateType, error) {
	ev := MetricEstimateType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricEstimateType: valid values are %v", v, allowedMetricEstimateTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricEstimateType) IsValid() bool {
	for _, existing := range allowedMetricEstimateTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricEstimateType value.
func (v MetricEstimateType) Ptr() *MetricEstimateType {
	return &v
}

// NullableMetricEstimateType handles when a null is used for MetricEstimateType.
type NullableMetricEstimateType struct {
	value *MetricEstimateType
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricEstimateType) Get() *MetricEstimateType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricEstimateType) Set(val *MetricEstimateType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricEstimateType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricEstimateType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricEstimateType initializes the struct as if Set has been called.
func NewNullableMetricEstimateType(val *MetricEstimateType) *NullableMetricEstimateType {
	return &NullableMetricEstimateType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricEstimateType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricEstimateType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
