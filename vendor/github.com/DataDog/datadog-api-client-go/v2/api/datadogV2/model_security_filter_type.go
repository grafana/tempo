// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityFilterType The type of the resource. The value should always be `security_filters`.
type SecurityFilterType string

// List of SecurityFilterType.
const (
	SECURITYFILTERTYPE_SECURITY_FILTERS SecurityFilterType = "security_filters"
)

var allowedSecurityFilterTypeEnumValues = []SecurityFilterType{
	SECURITYFILTERTYPE_SECURITY_FILTERS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityFilterType) GetAllowedValues() []SecurityFilterType {
	return allowedSecurityFilterTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityFilterType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityFilterType(value)
	return nil
}

// NewSecurityFilterTypeFromValue returns a pointer to a valid SecurityFilterType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityFilterTypeFromValue(v string) (*SecurityFilterType, error) {
	ev := SecurityFilterType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityFilterType: valid values are %v", v, allowedSecurityFilterTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityFilterType) IsValid() bool {
	for _, existing := range allowedSecurityFilterTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityFilterType value.
func (v SecurityFilterType) Ptr() *SecurityFilterType {
	return &v
}

// NullableSecurityFilterType handles when a null is used for SecurityFilterType.
type NullableSecurityFilterType struct {
	value *SecurityFilterType
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityFilterType) Get() *SecurityFilterType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityFilterType) Set(val *SecurityFilterType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityFilterType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityFilterType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityFilterType initializes the struct as if Set has been called.
func NewNullableSecurityFilterType(val *SecurityFilterType) *NullableSecurityFilterType {
	return &NullableSecurityFilterType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityFilterType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityFilterType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
