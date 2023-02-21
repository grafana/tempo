// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RolesType Roles type.
type RolesType string

// List of RolesType.
const (
	ROLESTYPE_ROLES RolesType = "roles"
)

var allowedRolesTypeEnumValues = []RolesType{
	ROLESTYPE_ROLES,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RolesType) GetAllowedValues() []RolesType {
	return allowedRolesTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RolesType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RolesType(value)
	return nil
}

// NewRolesTypeFromValue returns a pointer to a valid RolesType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRolesTypeFromValue(v string) (*RolesType, error) {
	ev := RolesType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RolesType: valid values are %v", v, allowedRolesTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RolesType) IsValid() bool {
	for _, existing := range allowedRolesTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RolesType value.
func (v RolesType) Ptr() *RolesType {
	return &v
}

// NullableRolesType handles when a null is used for RolesType.
type NullableRolesType struct {
	value *RolesType
	isSet bool
}

// Get returns the associated value.
func (v NullableRolesType) Get() *RolesType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRolesType) Set(val *RolesType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRolesType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRolesType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRolesType initializes the struct as if Set has been called.
func NewNullableRolesType(val *RolesType) *NullableRolesType {
	return &NullableRolesType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRolesType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRolesType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
