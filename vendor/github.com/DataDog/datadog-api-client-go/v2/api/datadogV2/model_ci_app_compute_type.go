// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CIAppComputeType The type of compute.
type CIAppComputeType string

// List of CIAppComputeType.
const (
	CIAPPCOMPUTETYPE_TIMESERIES CIAppComputeType = "timeseries"
	CIAPPCOMPUTETYPE_TOTAL      CIAppComputeType = "total"
)

var allowedCIAppComputeTypeEnumValues = []CIAppComputeType{
	CIAPPCOMPUTETYPE_TIMESERIES,
	CIAPPCOMPUTETYPE_TOTAL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CIAppComputeType) GetAllowedValues() []CIAppComputeType {
	return allowedCIAppComputeTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CIAppComputeType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CIAppComputeType(value)
	return nil
}

// NewCIAppComputeTypeFromValue returns a pointer to a valid CIAppComputeType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCIAppComputeTypeFromValue(v string) (*CIAppComputeType, error) {
	ev := CIAppComputeType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CIAppComputeType: valid values are %v", v, allowedCIAppComputeTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CIAppComputeType) IsValid() bool {
	for _, existing := range allowedCIAppComputeTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CIAppComputeType value.
func (v CIAppComputeType) Ptr() *CIAppComputeType {
	return &v
}

// NullableCIAppComputeType handles when a null is used for CIAppComputeType.
type NullableCIAppComputeType struct {
	value *CIAppComputeType
	isSet bool
}

// Get returns the associated value.
func (v NullableCIAppComputeType) Get() *CIAppComputeType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCIAppComputeType) Set(val *CIAppComputeType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCIAppComputeType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCIAppComputeType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCIAppComputeType initializes the struct as if Set has been called.
func NewNullableCIAppComputeType(val *CIAppComputeType) *NullableCIAppComputeType {
	return &NullableCIAppComputeType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCIAppComputeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCIAppComputeType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
