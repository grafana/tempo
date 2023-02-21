// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveOrderDefinitionType Type of the archive order definition.
type LogsArchiveOrderDefinitionType string

// List of LogsArchiveOrderDefinitionType.
const (
	LOGSARCHIVEORDERDEFINITIONTYPE_ARCHIVE_ORDER LogsArchiveOrderDefinitionType = "archive_order"
)

var allowedLogsArchiveOrderDefinitionTypeEnumValues = []LogsArchiveOrderDefinitionType{
	LOGSARCHIVEORDERDEFINITIONTYPE_ARCHIVE_ORDER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsArchiveOrderDefinitionType) GetAllowedValues() []LogsArchiveOrderDefinitionType {
	return allowedLogsArchiveOrderDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsArchiveOrderDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsArchiveOrderDefinitionType(value)
	return nil
}

// NewLogsArchiveOrderDefinitionTypeFromValue returns a pointer to a valid LogsArchiveOrderDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsArchiveOrderDefinitionTypeFromValue(v string) (*LogsArchiveOrderDefinitionType, error) {
	ev := LogsArchiveOrderDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsArchiveOrderDefinitionType: valid values are %v", v, allowedLogsArchiveOrderDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsArchiveOrderDefinitionType) IsValid() bool {
	for _, existing := range allowedLogsArchiveOrderDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsArchiveOrderDefinitionType value.
func (v LogsArchiveOrderDefinitionType) Ptr() *LogsArchiveOrderDefinitionType {
	return &v
}

// NullableLogsArchiveOrderDefinitionType handles when a null is used for LogsArchiveOrderDefinitionType.
type NullableLogsArchiveOrderDefinitionType struct {
	value *LogsArchiveOrderDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsArchiveOrderDefinitionType) Get() *LogsArchiveOrderDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsArchiveOrderDefinitionType) Set(val *LogsArchiveOrderDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsArchiveOrderDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsArchiveOrderDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsArchiveOrderDefinitionType initializes the struct as if Set has been called.
func NewNullableLogsArchiveOrderDefinitionType(val *LogsArchiveOrderDefinitionType) *NullableLogsArchiveOrderDefinitionType {
	return &NullableLogsArchiveOrderDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsArchiveOrderDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsArchiveOrderDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
