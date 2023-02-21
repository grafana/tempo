// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SAMLAssertionAttributesType SAML assertion attributes resource type.
type SAMLAssertionAttributesType string

// List of SAMLAssertionAttributesType.
const (
	SAMLASSERTIONATTRIBUTESTYPE_SAML_ASSERTION_ATTRIBUTES SAMLAssertionAttributesType = "saml_assertion_attributes"
)

var allowedSAMLAssertionAttributesTypeEnumValues = []SAMLAssertionAttributesType{
	SAMLASSERTIONATTRIBUTESTYPE_SAML_ASSERTION_ATTRIBUTES,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SAMLAssertionAttributesType) GetAllowedValues() []SAMLAssertionAttributesType {
	return allowedSAMLAssertionAttributesTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SAMLAssertionAttributesType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SAMLAssertionAttributesType(value)
	return nil
}

// NewSAMLAssertionAttributesTypeFromValue returns a pointer to a valid SAMLAssertionAttributesType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSAMLAssertionAttributesTypeFromValue(v string) (*SAMLAssertionAttributesType, error) {
	ev := SAMLAssertionAttributesType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SAMLAssertionAttributesType: valid values are %v", v, allowedSAMLAssertionAttributesTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SAMLAssertionAttributesType) IsValid() bool {
	for _, existing := range allowedSAMLAssertionAttributesTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SAMLAssertionAttributesType value.
func (v SAMLAssertionAttributesType) Ptr() *SAMLAssertionAttributesType {
	return &v
}

// NullableSAMLAssertionAttributesType handles when a null is used for SAMLAssertionAttributesType.
type NullableSAMLAssertionAttributesType struct {
	value *SAMLAssertionAttributesType
	isSet bool
}

// Get returns the associated value.
func (v NullableSAMLAssertionAttributesType) Get() *SAMLAssertionAttributesType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSAMLAssertionAttributesType) Set(val *SAMLAssertionAttributesType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSAMLAssertionAttributesType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSAMLAssertionAttributesType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSAMLAssertionAttributesType initializes the struct as if Set has been called.
func NewNullableSAMLAssertionAttributesType(val *SAMLAssertionAttributesType) *NullableSAMLAssertionAttributesType {
	return &NullableSAMLAssertionAttributesType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSAMLAssertionAttributesType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSAMLAssertionAttributesType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
