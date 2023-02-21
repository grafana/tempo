// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMEventType Type of the event.
type RUMEventType string

// List of RUMEventType.
const (
	RUMEVENTTYPE_RUM RUMEventType = "rum"
)

var allowedRUMEventTypeEnumValues = []RUMEventType{
	RUMEVENTTYPE_RUM,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RUMEventType) GetAllowedValues() []RUMEventType {
	return allowedRUMEventTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RUMEventType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RUMEventType(value)
	return nil
}

// NewRUMEventTypeFromValue returns a pointer to a valid RUMEventType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRUMEventTypeFromValue(v string) (*RUMEventType, error) {
	ev := RUMEventType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RUMEventType: valid values are %v", v, allowedRUMEventTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RUMEventType) IsValid() bool {
	for _, existing := range allowedRUMEventTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RUMEventType value.
func (v RUMEventType) Ptr() *RUMEventType {
	return &v
}

// NullableRUMEventType handles when a null is used for RUMEventType.
type NullableRUMEventType struct {
	value *RUMEventType
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMEventType) Get() *RUMEventType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMEventType) Set(val *RUMEventType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMEventType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRUMEventType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMEventType initializes the struct as if Set has been called.
func NewNullableRUMEventType(val *RUMEventType) *NullableRUMEventType {
	return &NullableRUMEventType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMEventType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMEventType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
