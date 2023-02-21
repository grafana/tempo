// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthDigestType The type of basic authentication to use when performing the test.
type SyntheticsBasicAuthDigestType string

// List of SyntheticsBasicAuthDigestType.
const (
	SYNTHETICSBASICAUTHDIGESTTYPE_DIGEST SyntheticsBasicAuthDigestType = "digest"
)

var allowedSyntheticsBasicAuthDigestTypeEnumValues = []SyntheticsBasicAuthDigestType{
	SYNTHETICSBASICAUTHDIGESTTYPE_DIGEST,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsBasicAuthDigestType) GetAllowedValues() []SyntheticsBasicAuthDigestType {
	return allowedSyntheticsBasicAuthDigestTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsBasicAuthDigestType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsBasicAuthDigestType(value)
	return nil
}

// NewSyntheticsBasicAuthDigestTypeFromValue returns a pointer to a valid SyntheticsBasicAuthDigestType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsBasicAuthDigestTypeFromValue(v string) (*SyntheticsBasicAuthDigestType, error) {
	ev := SyntheticsBasicAuthDigestType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsBasicAuthDigestType: valid values are %v", v, allowedSyntheticsBasicAuthDigestTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsBasicAuthDigestType) IsValid() bool {
	for _, existing := range allowedSyntheticsBasicAuthDigestTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsBasicAuthDigestType value.
func (v SyntheticsBasicAuthDigestType) Ptr() *SyntheticsBasicAuthDigestType {
	return &v
}

// NullableSyntheticsBasicAuthDigestType handles when a null is used for SyntheticsBasicAuthDigestType.
type NullableSyntheticsBasicAuthDigestType struct {
	value *SyntheticsBasicAuthDigestType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBasicAuthDigestType) Get() *SyntheticsBasicAuthDigestType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBasicAuthDigestType) Set(val *SyntheticsBasicAuthDigestType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBasicAuthDigestType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsBasicAuthDigestType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBasicAuthDigestType initializes the struct as if Set has been called.
func NewNullableSyntheticsBasicAuthDigestType(val *SyntheticsBasicAuthDigestType) *NullableSyntheticsBasicAuthDigestType {
	return &NullableSyntheticsBasicAuthDigestType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBasicAuthDigestType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBasicAuthDigestType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
