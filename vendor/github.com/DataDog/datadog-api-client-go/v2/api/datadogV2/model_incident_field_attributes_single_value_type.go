// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentFieldAttributesSingleValueType Type of the single value field definitions.
type IncidentFieldAttributesSingleValueType string

// List of IncidentFieldAttributesSingleValueType.
const (
	INCIDENTFIELDATTRIBUTESSINGLEVALUETYPE_DROPDOWN IncidentFieldAttributesSingleValueType = "dropdown"
	INCIDENTFIELDATTRIBUTESSINGLEVALUETYPE_TEXTBOX  IncidentFieldAttributesSingleValueType = "textbox"
)

var allowedIncidentFieldAttributesSingleValueTypeEnumValues = []IncidentFieldAttributesSingleValueType{
	INCIDENTFIELDATTRIBUTESSINGLEVALUETYPE_DROPDOWN,
	INCIDENTFIELDATTRIBUTESSINGLEVALUETYPE_TEXTBOX,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentFieldAttributesSingleValueType) GetAllowedValues() []IncidentFieldAttributesSingleValueType {
	return allowedIncidentFieldAttributesSingleValueTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentFieldAttributesSingleValueType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentFieldAttributesSingleValueType(value)
	return nil
}

// NewIncidentFieldAttributesSingleValueTypeFromValue returns a pointer to a valid IncidentFieldAttributesSingleValueType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentFieldAttributesSingleValueTypeFromValue(v string) (*IncidentFieldAttributesSingleValueType, error) {
	ev := IncidentFieldAttributesSingleValueType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentFieldAttributesSingleValueType: valid values are %v", v, allowedIncidentFieldAttributesSingleValueTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentFieldAttributesSingleValueType) IsValid() bool {
	for _, existing := range allowedIncidentFieldAttributesSingleValueTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentFieldAttributesSingleValueType value.
func (v IncidentFieldAttributesSingleValueType) Ptr() *IncidentFieldAttributesSingleValueType {
	return &v
}

// NullableIncidentFieldAttributesSingleValueType handles when a null is used for IncidentFieldAttributesSingleValueType.
type NullableIncidentFieldAttributesSingleValueType struct {
	value *IncidentFieldAttributesSingleValueType
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentFieldAttributesSingleValueType) Get() *IncidentFieldAttributesSingleValueType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentFieldAttributesSingleValueType) Set(val *IncidentFieldAttributesSingleValueType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentFieldAttributesSingleValueType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentFieldAttributesSingleValueType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentFieldAttributesSingleValueType initializes the struct as if Set has been called.
func NewNullableIncidentFieldAttributesSingleValueType(val *IncidentFieldAttributesSingleValueType) *NullableIncidentFieldAttributesSingleValueType {
	return &NullableIncidentFieldAttributesSingleValueType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentFieldAttributesSingleValueType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentFieldAttributesSingleValueType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
