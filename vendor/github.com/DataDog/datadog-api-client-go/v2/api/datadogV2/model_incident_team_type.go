// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentTeamType Incident Team resource type.
type IncidentTeamType string

// List of IncidentTeamType.
const (
	INCIDENTTEAMTYPE_TEAMS IncidentTeamType = "teams"
)

var allowedIncidentTeamTypeEnumValues = []IncidentTeamType{
	INCIDENTTEAMTYPE_TEAMS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentTeamType) GetAllowedValues() []IncidentTeamType {
	return allowedIncidentTeamTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentTeamType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentTeamType(value)
	return nil
}

// NewIncidentTeamTypeFromValue returns a pointer to a valid IncidentTeamType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentTeamTypeFromValue(v string) (*IncidentTeamType, error) {
	ev := IncidentTeamType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentTeamType: valid values are %v", v, allowedIncidentTeamTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentTeamType) IsValid() bool {
	for _, existing := range allowedIncidentTeamTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentTeamType value.
func (v IncidentTeamType) Ptr() *IncidentTeamType {
	return &v
}

// NullableIncidentTeamType handles when a null is used for IncidentTeamType.
type NullableIncidentTeamType struct {
	value *IncidentTeamType
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentTeamType) Get() *IncidentTeamType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentTeamType) Set(val *IncidentTeamType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentTeamType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentTeamType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentTeamType initializes the struct as if Set has been called.
func NewNullableIncidentTeamType(val *IncidentTeamType) *NullableIncidentTeamType {
	return &NullableIncidentTeamType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentTeamType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentTeamType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
