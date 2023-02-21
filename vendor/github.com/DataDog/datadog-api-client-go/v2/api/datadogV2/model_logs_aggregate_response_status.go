// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsAggregateResponseStatus The status of the response
type LogsAggregateResponseStatus string

// List of LogsAggregateResponseStatus.
const (
	LOGSAGGREGATERESPONSESTATUS_DONE    LogsAggregateResponseStatus = "done"
	LOGSAGGREGATERESPONSESTATUS_TIMEOUT LogsAggregateResponseStatus = "timeout"
)

var allowedLogsAggregateResponseStatusEnumValues = []LogsAggregateResponseStatus{
	LOGSAGGREGATERESPONSESTATUS_DONE,
	LOGSAGGREGATERESPONSESTATUS_TIMEOUT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsAggregateResponseStatus) GetAllowedValues() []LogsAggregateResponseStatus {
	return allowedLogsAggregateResponseStatusEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsAggregateResponseStatus) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsAggregateResponseStatus(value)
	return nil
}

// NewLogsAggregateResponseStatusFromValue returns a pointer to a valid LogsAggregateResponseStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsAggregateResponseStatusFromValue(v string) (*LogsAggregateResponseStatus, error) {
	ev := LogsAggregateResponseStatus(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsAggregateResponseStatus: valid values are %v", v, allowedLogsAggregateResponseStatusEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsAggregateResponseStatus) IsValid() bool {
	for _, existing := range allowedLogsAggregateResponseStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsAggregateResponseStatus value.
func (v LogsAggregateResponseStatus) Ptr() *LogsAggregateResponseStatus {
	return &v
}

// NullableLogsAggregateResponseStatus handles when a null is used for LogsAggregateResponseStatus.
type NullableLogsAggregateResponseStatus struct {
	value *LogsAggregateResponseStatus
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsAggregateResponseStatus) Get() *LogsAggregateResponseStatus {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsAggregateResponseStatus) Set(val *LogsAggregateResponseStatus) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsAggregateResponseStatus) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsAggregateResponseStatus) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsAggregateResponseStatus initializes the struct as if Set has been called.
func NewNullableLogsAggregateResponseStatus(val *LogsAggregateResponseStatus) *NullableLogsAggregateResponseStatus {
	return &NullableLogsAggregateResponseStatus{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsAggregateResponseStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsAggregateResponseStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
