// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityFilterFilteredDataType The filtered data type.
type SecurityFilterFilteredDataType string

// List of SecurityFilterFilteredDataType.
const (
	SECURITYFILTERFILTEREDDATATYPE_LOGS SecurityFilterFilteredDataType = "logs"
)

var allowedSecurityFilterFilteredDataTypeEnumValues = []SecurityFilterFilteredDataType{
	SECURITYFILTERFILTEREDDATATYPE_LOGS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityFilterFilteredDataType) GetAllowedValues() []SecurityFilterFilteredDataType {
	return allowedSecurityFilterFilteredDataTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityFilterFilteredDataType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityFilterFilteredDataType(value)
	return nil
}

// NewSecurityFilterFilteredDataTypeFromValue returns a pointer to a valid SecurityFilterFilteredDataType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityFilterFilteredDataTypeFromValue(v string) (*SecurityFilterFilteredDataType, error) {
	ev := SecurityFilterFilteredDataType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityFilterFilteredDataType: valid values are %v", v, allowedSecurityFilterFilteredDataTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityFilterFilteredDataType) IsValid() bool {
	for _, existing := range allowedSecurityFilterFilteredDataTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityFilterFilteredDataType value.
func (v SecurityFilterFilteredDataType) Ptr() *SecurityFilterFilteredDataType {
	return &v
}

// NullableSecurityFilterFilteredDataType handles when a null is used for SecurityFilterFilteredDataType.
type NullableSecurityFilterFilteredDataType struct {
	value *SecurityFilterFilteredDataType
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityFilterFilteredDataType) Get() *SecurityFilterFilteredDataType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityFilterFilteredDataType) Set(val *SecurityFilterFilteredDataType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityFilterFilteredDataType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityFilterFilteredDataType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityFilterFilteredDataType initializes the struct as if Set has been called.
func NewNullableSecurityFilterFilteredDataType(val *SecurityFilterFilteredDataType) *NullableSecurityFilterFilteredDataType {
	return &NullableSecurityFilterFilteredDataType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityFilterFilteredDataType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityFilterFilteredDataType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
