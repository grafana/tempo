// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMAggregateSortType The type of sorting algorithm.
type RUMAggregateSortType string

// List of RUMAggregateSortType.
const (
	RUMAGGREGATESORTTYPE_ALPHABETICAL RUMAggregateSortType = "alphabetical"
	RUMAGGREGATESORTTYPE_MEASURE      RUMAggregateSortType = "measure"
)

var allowedRUMAggregateSortTypeEnumValues = []RUMAggregateSortType{
	RUMAGGREGATESORTTYPE_ALPHABETICAL,
	RUMAGGREGATESORTTYPE_MEASURE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RUMAggregateSortType) GetAllowedValues() []RUMAggregateSortType {
	return allowedRUMAggregateSortTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RUMAggregateSortType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RUMAggregateSortType(value)
	return nil
}

// NewRUMAggregateSortTypeFromValue returns a pointer to a valid RUMAggregateSortType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRUMAggregateSortTypeFromValue(v string) (*RUMAggregateSortType, error) {
	ev := RUMAggregateSortType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RUMAggregateSortType: valid values are %v", v, allowedRUMAggregateSortTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RUMAggregateSortType) IsValid() bool {
	for _, existing := range allowedRUMAggregateSortTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RUMAggregateSortType value.
func (v RUMAggregateSortType) Ptr() *RUMAggregateSortType {
	return &v
}

// NullableRUMAggregateSortType handles when a null is used for RUMAggregateSortType.
type NullableRUMAggregateSortType struct {
	value *RUMAggregateSortType
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMAggregateSortType) Get() *RUMAggregateSortType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMAggregateSortType) Set(val *RUMAggregateSortType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMAggregateSortType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRUMAggregateSortType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMAggregateSortType initializes the struct as if Set has been called.
func NewNullableRUMAggregateSortType(val *RUMAggregateSortType) *NullableRUMAggregateSortType {
	return &NullableRUMAggregateSortType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMAggregateSortType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMAggregateSortType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
