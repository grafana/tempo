// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentAttachmentType The incident attachment resource type.
type IncidentAttachmentType string

// List of IncidentAttachmentType.
const (
	INCIDENTATTACHMENTTYPE_INCIDENT_ATTACHMENTS IncidentAttachmentType = "incident_attachments"
)

var allowedIncidentAttachmentTypeEnumValues = []IncidentAttachmentType{
	INCIDENTATTACHMENTTYPE_INCIDENT_ATTACHMENTS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentAttachmentType) GetAllowedValues() []IncidentAttachmentType {
	return allowedIncidentAttachmentTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentAttachmentType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentAttachmentType(value)
	return nil
}

// NewIncidentAttachmentTypeFromValue returns a pointer to a valid IncidentAttachmentType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentAttachmentTypeFromValue(v string) (*IncidentAttachmentType, error) {
	ev := IncidentAttachmentType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentAttachmentType: valid values are %v", v, allowedIncidentAttachmentTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentAttachmentType) IsValid() bool {
	for _, existing := range allowedIncidentAttachmentTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentAttachmentType value.
func (v IncidentAttachmentType) Ptr() *IncidentAttachmentType {
	return &v
}

// NullableIncidentAttachmentType handles when a null is used for IncidentAttachmentType.
type NullableIncidentAttachmentType struct {
	value *IncidentAttachmentType
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentAttachmentType) Get() *IncidentAttachmentType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentAttachmentType) Set(val *IncidentAttachmentType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentAttachmentType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentAttachmentType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentAttachmentType initializes the struct as if Set has been called.
func NewNullableIncidentAttachmentType(val *IncidentAttachmentType) *NullableIncidentAttachmentType {
	return &NullableIncidentAttachmentType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentAttachmentType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentAttachmentType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
