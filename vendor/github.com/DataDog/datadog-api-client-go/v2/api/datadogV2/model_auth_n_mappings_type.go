// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// AuthNMappingsType AuthN Mappings resource type.
type AuthNMappingsType string

// List of AuthNMappingsType.
const (
	AUTHNMAPPINGSTYPE_AUTHN_MAPPINGS AuthNMappingsType = "authn_mappings"
)

var allowedAuthNMappingsTypeEnumValues = []AuthNMappingsType{
	AUTHNMAPPINGSTYPE_AUTHN_MAPPINGS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *AuthNMappingsType) GetAllowedValues() []AuthNMappingsType {
	return allowedAuthNMappingsTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *AuthNMappingsType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = AuthNMappingsType(value)
	return nil
}

// NewAuthNMappingsTypeFromValue returns a pointer to a valid AuthNMappingsType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewAuthNMappingsTypeFromValue(v string) (*AuthNMappingsType, error) {
	ev := AuthNMappingsType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for AuthNMappingsType: valid values are %v", v, allowedAuthNMappingsTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v AuthNMappingsType) IsValid() bool {
	for _, existing := range allowedAuthNMappingsTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to AuthNMappingsType value.
func (v AuthNMappingsType) Ptr() *AuthNMappingsType {
	return &v
}

// NullableAuthNMappingsType handles when a null is used for AuthNMappingsType.
type NullableAuthNMappingsType struct {
	value *AuthNMappingsType
	isSet bool
}

// Get returns the associated value.
func (v NullableAuthNMappingsType) Get() *AuthNMappingsType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAuthNMappingsType) Set(val *AuthNMappingsType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAuthNMappingsType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableAuthNMappingsType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAuthNMappingsType initializes the struct as if Set has been called.
func NewNullableAuthNMappingsType(val *AuthNMappingsType) *NullableAuthNMappingsType {
	return &NullableAuthNMappingsType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAuthNMappingsType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAuthNMappingsType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
