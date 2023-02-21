// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CIAppTestEventTypeName Type of the event.
type CIAppTestEventTypeName string

// List of CIAppTestEventTypeName.
const (
	CIAPPTESTEVENTTYPENAME_citest CIAppTestEventTypeName = "citest"
)

var allowedCIAppTestEventTypeNameEnumValues = []CIAppTestEventTypeName{
	CIAPPTESTEVENTTYPENAME_citest,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CIAppTestEventTypeName) GetAllowedValues() []CIAppTestEventTypeName {
	return allowedCIAppTestEventTypeNameEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CIAppTestEventTypeName) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CIAppTestEventTypeName(value)
	return nil
}

// NewCIAppTestEventTypeNameFromValue returns a pointer to a valid CIAppTestEventTypeName
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCIAppTestEventTypeNameFromValue(v string) (*CIAppTestEventTypeName, error) {
	ev := CIAppTestEventTypeName(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CIAppTestEventTypeName: valid values are %v", v, allowedCIAppTestEventTypeNameEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CIAppTestEventTypeName) IsValid() bool {
	for _, existing := range allowedCIAppTestEventTypeNameEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CIAppTestEventTypeName value.
func (v CIAppTestEventTypeName) Ptr() *CIAppTestEventTypeName {
	return &v
}

// NullableCIAppTestEventTypeName handles when a null is used for CIAppTestEventTypeName.
type NullableCIAppTestEventTypeName struct {
	value *CIAppTestEventTypeName
	isSet bool
}

// Get returns the associated value.
func (v NullableCIAppTestEventTypeName) Get() *CIAppTestEventTypeName {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCIAppTestEventTypeName) Set(val *CIAppTestEventTypeName) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCIAppTestEventTypeName) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCIAppTestEventTypeName) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCIAppTestEventTypeName initializes the struct as if Set has been called.
func NewNullableCIAppTestEventTypeName(val *CIAppTestEventTypeName) *NullableCIAppTestEventTypeName {
	return &NullableCIAppTestEventTypeName{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCIAppTestEventTypeName) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCIAppTestEventTypeName) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
