// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMComputeType The type of compute.
type RUMComputeType string

// List of RUMComputeType.
const (
	RUMCOMPUTETYPE_TIMESERIES RUMComputeType = "timeseries"
	RUMCOMPUTETYPE_TOTAL      RUMComputeType = "total"
)

var allowedRUMComputeTypeEnumValues = []RUMComputeType{
	RUMCOMPUTETYPE_TIMESERIES,
	RUMCOMPUTETYPE_TOTAL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RUMComputeType) GetAllowedValues() []RUMComputeType {
	return allowedRUMComputeTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RUMComputeType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RUMComputeType(value)
	return nil
}

// NewRUMComputeTypeFromValue returns a pointer to a valid RUMComputeType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRUMComputeTypeFromValue(v string) (*RUMComputeType, error) {
	ev := RUMComputeType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RUMComputeType: valid values are %v", v, allowedRUMComputeTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RUMComputeType) IsValid() bool {
	for _, existing := range allowedRUMComputeTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RUMComputeType value.
func (v RUMComputeType) Ptr() *RUMComputeType {
	return &v
}

// NullableRUMComputeType handles when a null is used for RUMComputeType.
type NullableRUMComputeType struct {
	value *RUMComputeType
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMComputeType) Get() *RUMComputeType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMComputeType) Set(val *RUMComputeType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMComputeType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRUMComputeType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMComputeType initializes the struct as if Set has been called.
func NewNullableRUMComputeType(val *RUMComputeType) *NullableRUMComputeType {
	return &NullableRUMComputeType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMComputeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMComputeType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
