// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentIntegrationMetadataType Integration metadata resource type.
type IncidentIntegrationMetadataType string

// List of IncidentIntegrationMetadataType.
const (
	INCIDENTINTEGRATIONMETADATATYPE_INCIDENT_INTEGRATIONS IncidentIntegrationMetadataType = "incident_integrations"
)

var allowedIncidentIntegrationMetadataTypeEnumValues = []IncidentIntegrationMetadataType{
	INCIDENTINTEGRATIONMETADATATYPE_INCIDENT_INTEGRATIONS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentIntegrationMetadataType) GetAllowedValues() []IncidentIntegrationMetadataType {
	return allowedIncidentIntegrationMetadataTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentIntegrationMetadataType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentIntegrationMetadataType(value)
	return nil
}

// NewIncidentIntegrationMetadataTypeFromValue returns a pointer to a valid IncidentIntegrationMetadataType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentIntegrationMetadataTypeFromValue(v string) (*IncidentIntegrationMetadataType, error) {
	ev := IncidentIntegrationMetadataType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentIntegrationMetadataType: valid values are %v", v, allowedIncidentIntegrationMetadataTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentIntegrationMetadataType) IsValid() bool {
	for _, existing := range allowedIncidentIntegrationMetadataTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentIntegrationMetadataType value.
func (v IncidentIntegrationMetadataType) Ptr() *IncidentIntegrationMetadataType {
	return &v
}

// NullableIncidentIntegrationMetadataType handles when a null is used for IncidentIntegrationMetadataType.
type NullableIncidentIntegrationMetadataType struct {
	value *IncidentIntegrationMetadataType
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentIntegrationMetadataType) Get() *IncidentIntegrationMetadataType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentIntegrationMetadataType) Set(val *IncidentIntegrationMetadataType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentIntegrationMetadataType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentIntegrationMetadataType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentIntegrationMetadataType initializes the struct as if Set has been called.
func NewNullableIncidentIntegrationMetadataType(val *IncidentIntegrationMetadataType) *NullableIncidentIntegrationMetadataType {
	return &NullableIncidentIntegrationMetadataType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentIntegrationMetadataType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentIntegrationMetadataType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
