// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveState The state of the archive.
type LogsArchiveState string

// List of LogsArchiveState.
const (
	LOGSARCHIVESTATE_UNKNOWN             LogsArchiveState = "UNKNOWN"
	LOGSARCHIVESTATE_WORKING             LogsArchiveState = "WORKING"
	LOGSARCHIVESTATE_FAILING             LogsArchiveState = "FAILING"
	LOGSARCHIVESTATE_WORKING_AUTH_LEGACY LogsArchiveState = "WORKING_AUTH_LEGACY"
)

var allowedLogsArchiveStateEnumValues = []LogsArchiveState{
	LOGSARCHIVESTATE_UNKNOWN,
	LOGSARCHIVESTATE_WORKING,
	LOGSARCHIVESTATE_FAILING,
	LOGSARCHIVESTATE_WORKING_AUTH_LEGACY,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsArchiveState) GetAllowedValues() []LogsArchiveState {
	return allowedLogsArchiveStateEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsArchiveState) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsArchiveState(value)
	return nil
}

// NewLogsArchiveStateFromValue returns a pointer to a valid LogsArchiveState
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsArchiveStateFromValue(v string) (*LogsArchiveState, error) {
	ev := LogsArchiveState(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsArchiveState: valid values are %v", v, allowedLogsArchiveStateEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsArchiveState) IsValid() bool {
	for _, existing := range allowedLogsArchiveStateEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsArchiveState value.
func (v LogsArchiveState) Ptr() *LogsArchiveState {
	return &v
}

// NullableLogsArchiveState handles when a null is used for LogsArchiveState.
type NullableLogsArchiveState struct {
	value *LogsArchiveState
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsArchiveState) Get() *LogsArchiveState {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsArchiveState) Set(val *LogsArchiveState) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsArchiveState) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsArchiveState) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsArchiveState initializes the struct as if Set has been called.
func NewNullableLogsArchiveState(val *LogsArchiveState) *NullableLogsArchiveState {
	return &NullableLogsArchiveState{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsArchiveState) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsArchiveState) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
