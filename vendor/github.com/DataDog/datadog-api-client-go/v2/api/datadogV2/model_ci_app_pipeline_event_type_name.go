// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CIAppPipelineEventTypeName Type of the event.
type CIAppPipelineEventTypeName string

// List of CIAppPipelineEventTypeName.
const (
	CIAPPPIPELINEEVENTTYPENAME_cipipeline CIAppPipelineEventTypeName = "cipipeline"
)

var allowedCIAppPipelineEventTypeNameEnumValues = []CIAppPipelineEventTypeName{
	CIAPPPIPELINEEVENTTYPENAME_cipipeline,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CIAppPipelineEventTypeName) GetAllowedValues() []CIAppPipelineEventTypeName {
	return allowedCIAppPipelineEventTypeNameEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CIAppPipelineEventTypeName) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CIAppPipelineEventTypeName(value)
	return nil
}

// NewCIAppPipelineEventTypeNameFromValue returns a pointer to a valid CIAppPipelineEventTypeName
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCIAppPipelineEventTypeNameFromValue(v string) (*CIAppPipelineEventTypeName, error) {
	ev := CIAppPipelineEventTypeName(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CIAppPipelineEventTypeName: valid values are %v", v, allowedCIAppPipelineEventTypeNameEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CIAppPipelineEventTypeName) IsValid() bool {
	for _, existing := range allowedCIAppPipelineEventTypeNameEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CIAppPipelineEventTypeName value.
func (v CIAppPipelineEventTypeName) Ptr() *CIAppPipelineEventTypeName {
	return &v
}

// NullableCIAppPipelineEventTypeName handles when a null is used for CIAppPipelineEventTypeName.
type NullableCIAppPipelineEventTypeName struct {
	value *CIAppPipelineEventTypeName
	isSet bool
}

// Get returns the associated value.
func (v NullableCIAppPipelineEventTypeName) Get() *CIAppPipelineEventTypeName {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCIAppPipelineEventTypeName) Set(val *CIAppPipelineEventTypeName) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCIAppPipelineEventTypeName) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCIAppPipelineEventTypeName) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCIAppPipelineEventTypeName initializes the struct as if Set has been called.
func NewNullableCIAppPipelineEventTypeName(val *CIAppPipelineEventTypeName) *NullableCIAppPipelineEventTypeName {
	return &NullableCIAppPipelineEventTypeName{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCIAppPipelineEventTypeName) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCIAppPipelineEventTypeName) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
