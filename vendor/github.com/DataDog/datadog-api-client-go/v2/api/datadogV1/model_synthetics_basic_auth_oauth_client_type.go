// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthOauthClientType The type of basic authentication to use when performing the test.
type SyntheticsBasicAuthOauthClientType string

// List of SyntheticsBasicAuthOauthClientType.
const (
	SYNTHETICSBASICAUTHOAUTHCLIENTTYPE_OAUTH_CLIENT SyntheticsBasicAuthOauthClientType = "oauth-client"
)

var allowedSyntheticsBasicAuthOauthClientTypeEnumValues = []SyntheticsBasicAuthOauthClientType{
	SYNTHETICSBASICAUTHOAUTHCLIENTTYPE_OAUTH_CLIENT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsBasicAuthOauthClientType) GetAllowedValues() []SyntheticsBasicAuthOauthClientType {
	return allowedSyntheticsBasicAuthOauthClientTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsBasicAuthOauthClientType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsBasicAuthOauthClientType(value)
	return nil
}

// NewSyntheticsBasicAuthOauthClientTypeFromValue returns a pointer to a valid SyntheticsBasicAuthOauthClientType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsBasicAuthOauthClientTypeFromValue(v string) (*SyntheticsBasicAuthOauthClientType, error) {
	ev := SyntheticsBasicAuthOauthClientType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsBasicAuthOauthClientType: valid values are %v", v, allowedSyntheticsBasicAuthOauthClientTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsBasicAuthOauthClientType) IsValid() bool {
	for _, existing := range allowedSyntheticsBasicAuthOauthClientTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsBasicAuthOauthClientType value.
func (v SyntheticsBasicAuthOauthClientType) Ptr() *SyntheticsBasicAuthOauthClientType {
	return &v
}

// NullableSyntheticsBasicAuthOauthClientType handles when a null is used for SyntheticsBasicAuthOauthClientType.
type NullableSyntheticsBasicAuthOauthClientType struct {
	value *SyntheticsBasicAuthOauthClientType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBasicAuthOauthClientType) Get() *SyntheticsBasicAuthOauthClientType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBasicAuthOauthClientType) Set(val *SyntheticsBasicAuthOauthClientType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBasicAuthOauthClientType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsBasicAuthOauthClientType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBasicAuthOauthClientType initializes the struct as if Set has been called.
func NewNullableSyntheticsBasicAuthOauthClientType(val *SyntheticsBasicAuthOauthClientType) *NullableSyntheticsBasicAuthOauthClientType {
	return &NullableSyntheticsBasicAuthOauthClientType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBasicAuthOauthClientType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBasicAuthOauthClientType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
