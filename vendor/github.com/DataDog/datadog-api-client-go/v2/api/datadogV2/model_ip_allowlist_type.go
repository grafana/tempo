// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IPAllowlistType IP allowlist type.
type IPAllowlistType string

// List of IPAllowlistType.
const (
	IPALLOWLISTTYPE_IP_ALLOWLIST IPAllowlistType = "ip_allowlist"
)

var allowedIPAllowlistTypeEnumValues = []IPAllowlistType{
	IPALLOWLISTTYPE_IP_ALLOWLIST,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IPAllowlistType) GetAllowedValues() []IPAllowlistType {
	return allowedIPAllowlistTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IPAllowlistType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IPAllowlistType(value)
	return nil
}

// NewIPAllowlistTypeFromValue returns a pointer to a valid IPAllowlistType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIPAllowlistTypeFromValue(v string) (*IPAllowlistType, error) {
	ev := IPAllowlistType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IPAllowlistType: valid values are %v", v, allowedIPAllowlistTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IPAllowlistType) IsValid() bool {
	for _, existing := range allowedIPAllowlistTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IPAllowlistType value.
func (v IPAllowlistType) Ptr() *IPAllowlistType {
	return &v
}

// NullableIPAllowlistType handles when a null is used for IPAllowlistType.
type NullableIPAllowlistType struct {
	value *IPAllowlistType
	isSet bool
}

// Get returns the associated value.
func (v NullableIPAllowlistType) Get() *IPAllowlistType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIPAllowlistType) Set(val *IPAllowlistType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIPAllowlistType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIPAllowlistType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIPAllowlistType initializes the struct as if Set has been called.
func NewNullableIPAllowlistType(val *IPAllowlistType) *NullableIPAllowlistType {
	return &NullableIPAllowlistType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIPAllowlistType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIPAllowlistType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
