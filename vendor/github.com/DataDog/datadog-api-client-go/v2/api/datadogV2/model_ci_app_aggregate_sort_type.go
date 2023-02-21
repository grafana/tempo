// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CIAppAggregateSortType The type of sorting algorithm.
type CIAppAggregateSortType string

// List of CIAppAggregateSortType.
const (
	CIAPPAGGREGATESORTTYPE_ALPHABETICAL CIAppAggregateSortType = "alphabetical"
	CIAPPAGGREGATESORTTYPE_MEASURE      CIAppAggregateSortType = "measure"
)

var allowedCIAppAggregateSortTypeEnumValues = []CIAppAggregateSortType{
	CIAPPAGGREGATESORTTYPE_ALPHABETICAL,
	CIAPPAGGREGATESORTTYPE_MEASURE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CIAppAggregateSortType) GetAllowedValues() []CIAppAggregateSortType {
	return allowedCIAppAggregateSortTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CIAppAggregateSortType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CIAppAggregateSortType(value)
	return nil
}

// NewCIAppAggregateSortTypeFromValue returns a pointer to a valid CIAppAggregateSortType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCIAppAggregateSortTypeFromValue(v string) (*CIAppAggregateSortType, error) {
	ev := CIAppAggregateSortType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CIAppAggregateSortType: valid values are %v", v, allowedCIAppAggregateSortTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CIAppAggregateSortType) IsValid() bool {
	for _, existing := range allowedCIAppAggregateSortTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CIAppAggregateSortType value.
func (v CIAppAggregateSortType) Ptr() *CIAppAggregateSortType {
	return &v
}

// NullableCIAppAggregateSortType handles when a null is used for CIAppAggregateSortType.
type NullableCIAppAggregateSortType struct {
	value *CIAppAggregateSortType
	isSet bool
}

// Get returns the associated value.
func (v NullableCIAppAggregateSortType) Get() *CIAppAggregateSortType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCIAppAggregateSortType) Set(val *CIAppAggregateSortType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCIAppAggregateSortType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCIAppAggregateSortType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCIAppAggregateSortType initializes the struct as if Set has been called.
func NewNullableCIAppAggregateSortType(val *CIAppAggregateSortType) *NullableCIAppAggregateSortType {
	return &NullableCIAppAggregateSortType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCIAppAggregateSortType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCIAppAggregateSortType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
