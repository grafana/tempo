// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsTestCallType The type of gRPC call to perform.
type SyntheticsTestCallType string

// List of SyntheticsTestCallType.
const (
	SYNTHETICSTESTCALLTYPE_HEALTHCHECK SyntheticsTestCallType = "healthcheck"
	SYNTHETICSTESTCALLTYPE_UNARY       SyntheticsTestCallType = "unary"
)

var allowedSyntheticsTestCallTypeEnumValues = []SyntheticsTestCallType{
	SYNTHETICSTESTCALLTYPE_HEALTHCHECK,
	SYNTHETICSTESTCALLTYPE_UNARY,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsTestCallType) GetAllowedValues() []SyntheticsTestCallType {
	return allowedSyntheticsTestCallTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsTestCallType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsTestCallType(value)
	return nil
}

// NewSyntheticsTestCallTypeFromValue returns a pointer to a valid SyntheticsTestCallType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsTestCallTypeFromValue(v string) (*SyntheticsTestCallType, error) {
	ev := SyntheticsTestCallType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsTestCallType: valid values are %v", v, allowedSyntheticsTestCallTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsTestCallType) IsValid() bool {
	for _, existing := range allowedSyntheticsTestCallTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsTestCallType value.
func (v SyntheticsTestCallType) Ptr() *SyntheticsTestCallType {
	return &v
}

// NullableSyntheticsTestCallType handles when a null is used for SyntheticsTestCallType.
type NullableSyntheticsTestCallType struct {
	value *SyntheticsTestCallType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsTestCallType) Get() *SyntheticsTestCallType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsTestCallType) Set(val *SyntheticsTestCallType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsTestCallType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsTestCallType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsTestCallType initializes the struct as if Set has been called.
func NewNullableSyntheticsTestCallType(val *SyntheticsTestCallType) *NullableSyntheticsTestCallType {
	return &NullableSyntheticsTestCallType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsTestCallType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsTestCallType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
