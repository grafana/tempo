// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthOauthROPType The type of basic authentication to use when performing the test.
type SyntheticsBasicAuthOauthROPType string

// List of SyntheticsBasicAuthOauthROPType.
const (
	SYNTHETICSBASICAUTHOAUTHROPTYPE_OAUTH_ROP SyntheticsBasicAuthOauthROPType = "oauth-rop"
)

var allowedSyntheticsBasicAuthOauthROPTypeEnumValues = []SyntheticsBasicAuthOauthROPType{
	SYNTHETICSBASICAUTHOAUTHROPTYPE_OAUTH_ROP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsBasicAuthOauthROPType) GetAllowedValues() []SyntheticsBasicAuthOauthROPType {
	return allowedSyntheticsBasicAuthOauthROPTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsBasicAuthOauthROPType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsBasicAuthOauthROPType(value)
	return nil
}

// NewSyntheticsBasicAuthOauthROPTypeFromValue returns a pointer to a valid SyntheticsBasicAuthOauthROPType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsBasicAuthOauthROPTypeFromValue(v string) (*SyntheticsBasicAuthOauthROPType, error) {
	ev := SyntheticsBasicAuthOauthROPType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsBasicAuthOauthROPType: valid values are %v", v, allowedSyntheticsBasicAuthOauthROPTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsBasicAuthOauthROPType) IsValid() bool {
	for _, existing := range allowedSyntheticsBasicAuthOauthROPTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsBasicAuthOauthROPType value.
func (v SyntheticsBasicAuthOauthROPType) Ptr() *SyntheticsBasicAuthOauthROPType {
	return &v
}

// NullableSyntheticsBasicAuthOauthROPType handles when a null is used for SyntheticsBasicAuthOauthROPType.
type NullableSyntheticsBasicAuthOauthROPType struct {
	value *SyntheticsBasicAuthOauthROPType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBasicAuthOauthROPType) Get() *SyntheticsBasicAuthOauthROPType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBasicAuthOauthROPType) Set(val *SyntheticsBasicAuthOauthROPType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBasicAuthOauthROPType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsBasicAuthOauthROPType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBasicAuthOauthROPType initializes the struct as if Set has been called.
func NewNullableSyntheticsBasicAuthOauthROPType(val *SyntheticsBasicAuthOauthROPType) *NullableSyntheticsBasicAuthOauthROPType {
	return &NullableSyntheticsBasicAuthOauthROPType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBasicAuthOauthROPType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBasicAuthOauthROPType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
