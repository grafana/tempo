// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// UsageTimeSeriesType Type of usage data.
type UsageTimeSeriesType string

// List of UsageTimeSeriesType.
const (
	USAGETIMESERIESTYPE_USAGE_TIMESERIES UsageTimeSeriesType = "usage_timeseries"
)

var allowedUsageTimeSeriesTypeEnumValues = []UsageTimeSeriesType{
	USAGETIMESERIESTYPE_USAGE_TIMESERIES,
}

// GetAllowedValues reeturns the list of possible values.
func (v *UsageTimeSeriesType) GetAllowedValues() []UsageTimeSeriesType {
	return allowedUsageTimeSeriesTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *UsageTimeSeriesType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = UsageTimeSeriesType(value)
	return nil
}

// NewUsageTimeSeriesTypeFromValue returns a pointer to a valid UsageTimeSeriesType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewUsageTimeSeriesTypeFromValue(v string) (*UsageTimeSeriesType, error) {
	ev := UsageTimeSeriesType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for UsageTimeSeriesType: valid values are %v", v, allowedUsageTimeSeriesTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v UsageTimeSeriesType) IsValid() bool {
	for _, existing := range allowedUsageTimeSeriesTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to UsageTimeSeriesType value.
func (v UsageTimeSeriesType) Ptr() *UsageTimeSeriesType {
	return &v
}

// NullableUsageTimeSeriesType handles when a null is used for UsageTimeSeriesType.
type NullableUsageTimeSeriesType struct {
	value *UsageTimeSeriesType
	isSet bool
}

// Get returns the associated value.
func (v NullableUsageTimeSeriesType) Get() *UsageTimeSeriesType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableUsageTimeSeriesType) Set(val *UsageTimeSeriesType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableUsageTimeSeriesType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableUsageTimeSeriesType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableUsageTimeSeriesType initializes the struct as if Set has been called.
func NewNullableUsageTimeSeriesType(val *UsageTimeSeriesType) *NullableUsageTimeSeriesType {
	return &NullableUsageTimeSeriesType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableUsageTimeSeriesType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableUsageTimeSeriesType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
