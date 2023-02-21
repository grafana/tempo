// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentAttachmentRelatedObject The object related to an incident attachment.
type IncidentAttachmentRelatedObject string

// List of IncidentAttachmentRelatedObject.
const (
	INCIDENTATTACHMENTRELATEDOBJECT_USERS IncidentAttachmentRelatedObject = "users"
)

var allowedIncidentAttachmentRelatedObjectEnumValues = []IncidentAttachmentRelatedObject{
	INCIDENTATTACHMENTRELATEDOBJECT_USERS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentAttachmentRelatedObject) GetAllowedValues() []IncidentAttachmentRelatedObject {
	return allowedIncidentAttachmentRelatedObjectEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentAttachmentRelatedObject) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentAttachmentRelatedObject(value)
	return nil
}

// NewIncidentAttachmentRelatedObjectFromValue returns a pointer to a valid IncidentAttachmentRelatedObject
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentAttachmentRelatedObjectFromValue(v string) (*IncidentAttachmentRelatedObject, error) {
	ev := IncidentAttachmentRelatedObject(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentAttachmentRelatedObject: valid values are %v", v, allowedIncidentAttachmentRelatedObjectEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentAttachmentRelatedObject) IsValid() bool {
	for _, existing := range allowedIncidentAttachmentRelatedObjectEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentAttachmentRelatedObject value.
func (v IncidentAttachmentRelatedObject) Ptr() *IncidentAttachmentRelatedObject {
	return &v
}

// NullableIncidentAttachmentRelatedObject handles when a null is used for IncidentAttachmentRelatedObject.
type NullableIncidentAttachmentRelatedObject struct {
	value *IncidentAttachmentRelatedObject
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentAttachmentRelatedObject) Get() *IncidentAttachmentRelatedObject {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentAttachmentRelatedObject) Set(val *IncidentAttachmentRelatedObject) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentAttachmentRelatedObject) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentAttachmentRelatedObject) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentAttachmentRelatedObject initializes the struct as if Set has been called.
func NewNullableIncidentAttachmentRelatedObject(val *IncidentAttachmentRelatedObject) *NullableIncidentAttachmentRelatedObject {
	return &NullableIncidentAttachmentRelatedObject{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentAttachmentRelatedObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentAttachmentRelatedObject) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
