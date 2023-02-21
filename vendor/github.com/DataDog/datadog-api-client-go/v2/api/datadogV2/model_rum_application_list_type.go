// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMApplicationListType RUM application list type.
type RUMApplicationListType string

// List of RUMApplicationListType.
const (
	RUMAPPLICATIONLISTTYPE_RUM_APPLICATION RUMApplicationListType = "rum_application"
)

var allowedRUMApplicationListTypeEnumValues = []RUMApplicationListType{
	RUMAPPLICATIONLISTTYPE_RUM_APPLICATION,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RUMApplicationListType) GetAllowedValues() []RUMApplicationListType {
	return allowedRUMApplicationListTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RUMApplicationListType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RUMApplicationListType(value)
	return nil
}

// NewRUMApplicationListTypeFromValue returns a pointer to a valid RUMApplicationListType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRUMApplicationListTypeFromValue(v string) (*RUMApplicationListType, error) {
	ev := RUMApplicationListType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RUMApplicationListType: valid values are %v", v, allowedRUMApplicationListTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RUMApplicationListType) IsValid() bool {
	for _, existing := range allowedRUMApplicationListTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RUMApplicationListType value.
func (v RUMApplicationListType) Ptr() *RUMApplicationListType {
	return &v
}

// NullableRUMApplicationListType handles when a null is used for RUMApplicationListType.
type NullableRUMApplicationListType struct {
	value *RUMApplicationListType
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMApplicationListType) Get() *RUMApplicationListType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMApplicationListType) Set(val *RUMApplicationListType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMApplicationListType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRUMApplicationListType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMApplicationListType initializes the struct as if Set has been called.
func NewNullableRUMApplicationListType(val *RUMApplicationListType) *NullableRUMApplicationListType {
	return &NullableRUMApplicationListType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMApplicationListType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMApplicationListType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
