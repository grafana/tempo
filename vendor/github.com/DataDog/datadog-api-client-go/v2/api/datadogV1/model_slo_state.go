// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SLOState State of the SLO.
type SLOState string

// List of SLOState.
const (
	SLOSTATE_BREACHED SLOState = "breached"
	SLOSTATE_WARNING  SLOState = "warning"
	SLOSTATE_OK       SLOState = "ok"
	SLOSTATE_NO_DATA  SLOState = "no_data"
)

var allowedSLOStateEnumValues = []SLOState{
	SLOSTATE_BREACHED,
	SLOSTATE_WARNING,
	SLOSTATE_OK,
	SLOSTATE_NO_DATA,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SLOState) GetAllowedValues() []SLOState {
	return allowedSLOStateEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SLOState) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SLOState(value)
	return nil
}

// NewSLOStateFromValue returns a pointer to a valid SLOState
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSLOStateFromValue(v string) (*SLOState, error) {
	ev := SLOState(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SLOState: valid values are %v", v, allowedSLOStateEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SLOState) IsValid() bool {
	for _, existing := range allowedSLOStateEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SLOState value.
func (v SLOState) Ptr() *SLOState {
	return &v
}

// NullableSLOState handles when a null is used for SLOState.
type NullableSLOState struct {
	value *SLOState
	isSet bool
}

// Get returns the associated value.
func (v NullableSLOState) Get() *SLOState {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLOState) Set(val *SLOState) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLOState) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSLOState) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLOState initializes the struct as if Set has been called.
func NewNullableSLOState(val *SLOState) *NullableSLOState {
	return &NullableSLOState{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLOState) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLOState) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
