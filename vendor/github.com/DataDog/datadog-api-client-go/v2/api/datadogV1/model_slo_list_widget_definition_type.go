// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SLOListWidgetDefinitionType Type of the SLO List widget.
type SLOListWidgetDefinitionType string

// List of SLOListWidgetDefinitionType.
const (
	SLOLISTWIDGETDEFINITIONTYPE_SLO_LIST SLOListWidgetDefinitionType = "slo_list"
)

var allowedSLOListWidgetDefinitionTypeEnumValues = []SLOListWidgetDefinitionType{
	SLOLISTWIDGETDEFINITIONTYPE_SLO_LIST,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SLOListWidgetDefinitionType) GetAllowedValues() []SLOListWidgetDefinitionType {
	return allowedSLOListWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SLOListWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SLOListWidgetDefinitionType(value)
	return nil
}

// NewSLOListWidgetDefinitionTypeFromValue returns a pointer to a valid SLOListWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSLOListWidgetDefinitionTypeFromValue(v string) (*SLOListWidgetDefinitionType, error) {
	ev := SLOListWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SLOListWidgetDefinitionType: valid values are %v", v, allowedSLOListWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SLOListWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedSLOListWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SLOListWidgetDefinitionType value.
func (v SLOListWidgetDefinitionType) Ptr() *SLOListWidgetDefinitionType {
	return &v
}

// NullableSLOListWidgetDefinitionType handles when a null is used for SLOListWidgetDefinitionType.
type NullableSLOListWidgetDefinitionType struct {
	value *SLOListWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableSLOListWidgetDefinitionType) Get() *SLOListWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLOListWidgetDefinitionType) Set(val *SLOListWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLOListWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSLOListWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLOListWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableSLOListWidgetDefinitionType(val *SLOListWidgetDefinitionType) *NullableSLOListWidgetDefinitionType {
	return &NullableSLOListWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLOListWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLOListWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
