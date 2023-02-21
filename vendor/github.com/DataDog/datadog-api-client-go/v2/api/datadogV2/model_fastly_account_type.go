// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// FastlyAccountType The JSON:API type for this API. Should always be `fastly-accounts`.
type FastlyAccountType string

// List of FastlyAccountType.
const (
	FASTLYACCOUNTTYPE_FASTLY_ACCOUNTS FastlyAccountType = "fastly-accounts"
)

var allowedFastlyAccountTypeEnumValues = []FastlyAccountType{
	FASTLYACCOUNTTYPE_FASTLY_ACCOUNTS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FastlyAccountType) GetAllowedValues() []FastlyAccountType {
	return allowedFastlyAccountTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FastlyAccountType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FastlyAccountType(value)
	return nil
}

// NewFastlyAccountTypeFromValue returns a pointer to a valid FastlyAccountType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFastlyAccountTypeFromValue(v string) (*FastlyAccountType, error) {
	ev := FastlyAccountType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FastlyAccountType: valid values are %v", v, allowedFastlyAccountTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FastlyAccountType) IsValid() bool {
	for _, existing := range allowedFastlyAccountTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FastlyAccountType value.
func (v FastlyAccountType) Ptr() *FastlyAccountType {
	return &v
}

// NullableFastlyAccountType handles when a null is used for FastlyAccountType.
type NullableFastlyAccountType struct {
	value *FastlyAccountType
	isSet bool
}

// Get returns the associated value.
func (v NullableFastlyAccountType) Get() *FastlyAccountType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFastlyAccountType) Set(val *FastlyAccountType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFastlyAccountType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFastlyAccountType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFastlyAccountType initializes the struct as if Set has been called.
func NewNullableFastlyAccountType(val *FastlyAccountType) *NullableFastlyAccountType {
	return &NullableFastlyAccountType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFastlyAccountType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFastlyAccountType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
