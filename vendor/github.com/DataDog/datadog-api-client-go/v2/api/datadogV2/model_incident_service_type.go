// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentServiceType Incident service resource type.
type IncidentServiceType string

// List of IncidentServiceType.
const (
	INCIDENTSERVICETYPE_SERVICES IncidentServiceType = "services"
)

var allowedIncidentServiceTypeEnumValues = []IncidentServiceType{
	INCIDENTSERVICETYPE_SERVICES,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentServiceType) GetAllowedValues() []IncidentServiceType {
	return allowedIncidentServiceTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentServiceType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentServiceType(value)
	return nil
}

// NewIncidentServiceTypeFromValue returns a pointer to a valid IncidentServiceType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentServiceTypeFromValue(v string) (*IncidentServiceType, error) {
	ev := IncidentServiceType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentServiceType: valid values are %v", v, allowedIncidentServiceTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentServiceType) IsValid() bool {
	for _, existing := range allowedIncidentServiceTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentServiceType value.
func (v IncidentServiceType) Ptr() *IncidentServiceType {
	return &v
}

// NullableIncidentServiceType handles when a null is used for IncidentServiceType.
type NullableIncidentServiceType struct {
	value *IncidentServiceType
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentServiceType) Get() *IncidentServiceType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentServiceType) Set(val *IncidentServiceType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentServiceType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentServiceType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentServiceType initializes the struct as if Set has been called.
func NewNullableIncidentServiceType(val *IncidentServiceType) *NullableIncidentServiceType {
	return &NullableIncidentServiceType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentServiceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentServiceType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
