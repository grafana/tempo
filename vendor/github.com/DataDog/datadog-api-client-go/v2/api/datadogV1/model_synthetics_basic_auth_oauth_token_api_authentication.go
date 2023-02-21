// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthOauthTokenApiAuthentication Type of token to use when performing the authentication.
type SyntheticsBasicAuthOauthTokenApiAuthentication string

// List of SyntheticsBasicAuthOauthTokenApiAuthentication.
const (
	SYNTHETICSBASICAUTHOAUTHTOKENAPIAUTHENTICATION_HEADER SyntheticsBasicAuthOauthTokenApiAuthentication = "header"
	SYNTHETICSBASICAUTHOAUTHTOKENAPIAUTHENTICATION_BODY   SyntheticsBasicAuthOauthTokenApiAuthentication = "body"
)

var allowedSyntheticsBasicAuthOauthTokenApiAuthenticationEnumValues = []SyntheticsBasicAuthOauthTokenApiAuthentication{
	SYNTHETICSBASICAUTHOAUTHTOKENAPIAUTHENTICATION_HEADER,
	SYNTHETICSBASICAUTHOAUTHTOKENAPIAUTHENTICATION_BODY,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsBasicAuthOauthTokenApiAuthentication) GetAllowedValues() []SyntheticsBasicAuthOauthTokenApiAuthentication {
	return allowedSyntheticsBasicAuthOauthTokenApiAuthenticationEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsBasicAuthOauthTokenApiAuthentication) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsBasicAuthOauthTokenApiAuthentication(value)
	return nil
}

// NewSyntheticsBasicAuthOauthTokenApiAuthenticationFromValue returns a pointer to a valid SyntheticsBasicAuthOauthTokenApiAuthentication
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsBasicAuthOauthTokenApiAuthenticationFromValue(v string) (*SyntheticsBasicAuthOauthTokenApiAuthentication, error) {
	ev := SyntheticsBasicAuthOauthTokenApiAuthentication(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsBasicAuthOauthTokenApiAuthentication: valid values are %v", v, allowedSyntheticsBasicAuthOauthTokenApiAuthenticationEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsBasicAuthOauthTokenApiAuthentication) IsValid() bool {
	for _, existing := range allowedSyntheticsBasicAuthOauthTokenApiAuthenticationEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsBasicAuthOauthTokenApiAuthentication value.
func (v SyntheticsBasicAuthOauthTokenApiAuthentication) Ptr() *SyntheticsBasicAuthOauthTokenApiAuthentication {
	return &v
}

// NullableSyntheticsBasicAuthOauthTokenApiAuthentication handles when a null is used for SyntheticsBasicAuthOauthTokenApiAuthentication.
type NullableSyntheticsBasicAuthOauthTokenApiAuthentication struct {
	value *SyntheticsBasicAuthOauthTokenApiAuthentication
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBasicAuthOauthTokenApiAuthentication) Get() *SyntheticsBasicAuthOauthTokenApiAuthentication {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBasicAuthOauthTokenApiAuthentication) Set(val *SyntheticsBasicAuthOauthTokenApiAuthentication) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBasicAuthOauthTokenApiAuthentication) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsBasicAuthOauthTokenApiAuthentication) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBasicAuthOauthTokenApiAuthentication initializes the struct as if Set has been called.
func NewNullableSyntheticsBasicAuthOauthTokenApiAuthentication(val *SyntheticsBasicAuthOauthTokenApiAuthentication) *NullableSyntheticsBasicAuthOauthTokenApiAuthentication {
	return &NullableSyntheticsBasicAuthOauthTokenApiAuthentication{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBasicAuthOauthTokenApiAuthentication) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBasicAuthOauthTokenApiAuthentication) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
