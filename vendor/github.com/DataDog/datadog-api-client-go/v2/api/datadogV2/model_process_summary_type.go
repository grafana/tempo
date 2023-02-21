// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ProcessSummaryType Type of process summary.
type ProcessSummaryType string

// List of ProcessSummaryType.
const (
	PROCESSSUMMARYTYPE_PROCESS ProcessSummaryType = "process"
)

var allowedProcessSummaryTypeEnumValues = []ProcessSummaryType{
	PROCESSSUMMARYTYPE_PROCESS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ProcessSummaryType) GetAllowedValues() []ProcessSummaryType {
	return allowedProcessSummaryTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ProcessSummaryType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ProcessSummaryType(value)
	return nil
}

// NewProcessSummaryTypeFromValue returns a pointer to a valid ProcessSummaryType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewProcessSummaryTypeFromValue(v string) (*ProcessSummaryType, error) {
	ev := ProcessSummaryType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ProcessSummaryType: valid values are %v", v, allowedProcessSummaryTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ProcessSummaryType) IsValid() bool {
	for _, existing := range allowedProcessSummaryTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ProcessSummaryType value.
func (v ProcessSummaryType) Ptr() *ProcessSummaryType {
	return &v
}

// NullableProcessSummaryType handles when a null is used for ProcessSummaryType.
type NullableProcessSummaryType struct {
	value *ProcessSummaryType
	isSet bool
}

// Get returns the associated value.
func (v NullableProcessSummaryType) Get() *ProcessSummaryType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableProcessSummaryType) Set(val *ProcessSummaryType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableProcessSummaryType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableProcessSummaryType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableProcessSummaryType initializes the struct as if Set has been called.
func NewNullableProcessSummaryType(val *ProcessSummaryType) *NullableProcessSummaryType {
	return &NullableProcessSummaryType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableProcessSummaryType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableProcessSummaryType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
