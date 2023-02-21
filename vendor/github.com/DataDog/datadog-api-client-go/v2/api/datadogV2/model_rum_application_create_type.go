// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMApplicationCreateType RUM application creation type.
type RUMApplicationCreateType string

// List of RUMApplicationCreateType.
const (
	RUMAPPLICATIONCREATETYPE_RUM_APPLICATION_CREATE RUMApplicationCreateType = "rum_application_create"
)

var allowedRUMApplicationCreateTypeEnumValues = []RUMApplicationCreateType{
	RUMAPPLICATIONCREATETYPE_RUM_APPLICATION_CREATE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RUMApplicationCreateType) GetAllowedValues() []RUMApplicationCreateType {
	return allowedRUMApplicationCreateTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RUMApplicationCreateType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RUMApplicationCreateType(value)
	return nil
}

// NewRUMApplicationCreateTypeFromValue returns a pointer to a valid RUMApplicationCreateType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRUMApplicationCreateTypeFromValue(v string) (*RUMApplicationCreateType, error) {
	ev := RUMApplicationCreateType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RUMApplicationCreateType: valid values are %v", v, allowedRUMApplicationCreateTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RUMApplicationCreateType) IsValid() bool {
	for _, existing := range allowedRUMApplicationCreateTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RUMApplicationCreateType value.
func (v RUMApplicationCreateType) Ptr() *RUMApplicationCreateType {
	return &v
}

// NullableRUMApplicationCreateType handles when a null is used for RUMApplicationCreateType.
type NullableRUMApplicationCreateType struct {
	value *RUMApplicationCreateType
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMApplicationCreateType) Get() *RUMApplicationCreateType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMApplicationCreateType) Set(val *RUMApplicationCreateType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMApplicationCreateType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRUMApplicationCreateType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMApplicationCreateType initializes the struct as if Set has been called.
func NewNullableRUMApplicationCreateType(val *RUMApplicationCreateType) *NullableRUMApplicationCreateType {
	return &NullableRUMApplicationCreateType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMApplicationCreateType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMApplicationCreateType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
