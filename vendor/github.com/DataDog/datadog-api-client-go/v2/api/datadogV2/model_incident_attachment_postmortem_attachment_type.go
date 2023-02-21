// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentAttachmentPostmortemAttachmentType The type of postmortem attachment attributes.
type IncidentAttachmentPostmortemAttachmentType string

// List of IncidentAttachmentPostmortemAttachmentType.
const (
	INCIDENTATTACHMENTPOSTMORTEMATTACHMENTTYPE_POSTMORTEM IncidentAttachmentPostmortemAttachmentType = "postmortem"
)

var allowedIncidentAttachmentPostmortemAttachmentTypeEnumValues = []IncidentAttachmentPostmortemAttachmentType{
	INCIDENTATTACHMENTPOSTMORTEMATTACHMENTTYPE_POSTMORTEM,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentAttachmentPostmortemAttachmentType) GetAllowedValues() []IncidentAttachmentPostmortemAttachmentType {
	return allowedIncidentAttachmentPostmortemAttachmentTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentAttachmentPostmortemAttachmentType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentAttachmentPostmortemAttachmentType(value)
	return nil
}

// NewIncidentAttachmentPostmortemAttachmentTypeFromValue returns a pointer to a valid IncidentAttachmentPostmortemAttachmentType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentAttachmentPostmortemAttachmentTypeFromValue(v string) (*IncidentAttachmentPostmortemAttachmentType, error) {
	ev := IncidentAttachmentPostmortemAttachmentType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentAttachmentPostmortemAttachmentType: valid values are %v", v, allowedIncidentAttachmentPostmortemAttachmentTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentAttachmentPostmortemAttachmentType) IsValid() bool {
	for _, existing := range allowedIncidentAttachmentPostmortemAttachmentTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentAttachmentPostmortemAttachmentType value.
func (v IncidentAttachmentPostmortemAttachmentType) Ptr() *IncidentAttachmentPostmortemAttachmentType {
	return &v
}

// NullableIncidentAttachmentPostmortemAttachmentType handles when a null is used for IncidentAttachmentPostmortemAttachmentType.
type NullableIncidentAttachmentPostmortemAttachmentType struct {
	value *IncidentAttachmentPostmortemAttachmentType
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentAttachmentPostmortemAttachmentType) Get() *IncidentAttachmentPostmortemAttachmentType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentAttachmentPostmortemAttachmentType) Set(val *IncidentAttachmentPostmortemAttachmentType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentAttachmentPostmortemAttachmentType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentAttachmentPostmortemAttachmentType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentAttachmentPostmortemAttachmentType initializes the struct as if Set has been called.
func NewNullableIncidentAttachmentPostmortemAttachmentType(val *IncidentAttachmentPostmortemAttachmentType) *NullableIncidentAttachmentPostmortemAttachmentType {
	return &NullableIncidentAttachmentPostmortemAttachmentType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentAttachmentPostmortemAttachmentType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentAttachmentPostmortemAttachmentType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
