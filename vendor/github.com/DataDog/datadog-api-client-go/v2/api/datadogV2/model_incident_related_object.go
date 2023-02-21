// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentRelatedObject Object related to an incident.
type IncidentRelatedObject string

// List of IncidentRelatedObject.
const (
	INCIDENTRELATEDOBJECT_USERS       IncidentRelatedObject = "users"
	INCIDENTRELATEDOBJECT_ATTACHMENTS IncidentRelatedObject = "attachments"
)

var allowedIncidentRelatedObjectEnumValues = []IncidentRelatedObject{
	INCIDENTRELATEDOBJECT_USERS,
	INCIDENTRELATEDOBJECT_ATTACHMENTS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentRelatedObject) GetAllowedValues() []IncidentRelatedObject {
	return allowedIncidentRelatedObjectEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentRelatedObject) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentRelatedObject(value)
	return nil
}

// NewIncidentRelatedObjectFromValue returns a pointer to a valid IncidentRelatedObject
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentRelatedObjectFromValue(v string) (*IncidentRelatedObject, error) {
	ev := IncidentRelatedObject(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentRelatedObject: valid values are %v", v, allowedIncidentRelatedObjectEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentRelatedObject) IsValid() bool {
	for _, existing := range allowedIncidentRelatedObjectEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentRelatedObject value.
func (v IncidentRelatedObject) Ptr() *IncidentRelatedObject {
	return &v
}

// NullableIncidentRelatedObject handles when a null is used for IncidentRelatedObject.
type NullableIncidentRelatedObject struct {
	value *IncidentRelatedObject
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentRelatedObject) Get() *IncidentRelatedObject {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentRelatedObject) Set(val *IncidentRelatedObject) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentRelatedObject) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentRelatedObject) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentRelatedObject initializes the struct as if Set has been called.
func NewNullableIncidentRelatedObject(val *IncidentRelatedObject) *NullableIncidentRelatedObject {
	return &NullableIncidentRelatedObject{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentRelatedObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentRelatedObject) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
