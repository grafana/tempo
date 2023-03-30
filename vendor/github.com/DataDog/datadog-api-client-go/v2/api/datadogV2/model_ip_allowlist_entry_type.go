// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IPAllowlistEntryType IP allowlist Entry type.
type IPAllowlistEntryType string

// List of IPAllowlistEntryType.
const (
	IPALLOWLISTENTRYTYPE_IP_ALLOWLIST_ENTRY IPAllowlistEntryType = "ip_allowlist_entry"
)

var allowedIPAllowlistEntryTypeEnumValues = []IPAllowlistEntryType{
	IPALLOWLISTENTRYTYPE_IP_ALLOWLIST_ENTRY,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IPAllowlistEntryType) GetAllowedValues() []IPAllowlistEntryType {
	return allowedIPAllowlistEntryTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IPAllowlistEntryType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IPAllowlistEntryType(value)
	return nil
}

// NewIPAllowlistEntryTypeFromValue returns a pointer to a valid IPAllowlistEntryType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIPAllowlistEntryTypeFromValue(v string) (*IPAllowlistEntryType, error) {
	ev := IPAllowlistEntryType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IPAllowlistEntryType: valid values are %v", v, allowedIPAllowlistEntryTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IPAllowlistEntryType) IsValid() bool {
	for _, existing := range allowedIPAllowlistEntryTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IPAllowlistEntryType value.
func (v IPAllowlistEntryType) Ptr() *IPAllowlistEntryType {
	return &v
}

// NullableIPAllowlistEntryType handles when a null is used for IPAllowlistEntryType.
type NullableIPAllowlistEntryType struct {
	value *IPAllowlistEntryType
	isSet bool
}

// Get returns the associated value.
func (v NullableIPAllowlistEntryType) Get() *IPAllowlistEntryType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIPAllowlistEntryType) Set(val *IPAllowlistEntryType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIPAllowlistEntryType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIPAllowlistEntryType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIPAllowlistEntryType initializes the struct as if Set has been called.
func NewNullableIPAllowlistEntryType(val *IPAllowlistEntryType) *NullableIPAllowlistEntryType {
	return &NullableIPAllowlistEntryType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIPAllowlistEntryType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIPAllowlistEntryType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
