// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogType Type of the event.
type LogType string

// List of LogType.
const (
	LOGTYPE_LOG LogType = "log"
)

var allowedLogTypeEnumValues = []LogType{
	LOGTYPE_LOG,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogType) GetAllowedValues() []LogType {
	return allowedLogTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogType(value)
	return nil
}

// NewLogTypeFromValue returns a pointer to a valid LogType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogTypeFromValue(v string) (*LogType, error) {
	ev := LogType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogType: valid values are %v", v, allowedLogTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogType) IsValid() bool {
	for _, existing := range allowedLogTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogType value.
func (v LogType) Ptr() *LogType {
	return &v
}

// NullableLogType handles when a null is used for LogType.
type NullableLogType struct {
	value *LogType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogType) Get() *LogType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogType) Set(val *LogType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogType initializes the struct as if Set has been called.
func NewNullableLogType(val *LogType) *NullableLogType {
	return &NullableLogType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
