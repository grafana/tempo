// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveDestinationAzureType Type of the Azure archive destination.
type LogsArchiveDestinationAzureType string

// List of LogsArchiveDestinationAzureType.
const (
	LOGSARCHIVEDESTINATIONAZURETYPE_AZURE LogsArchiveDestinationAzureType = "azure"
)

var allowedLogsArchiveDestinationAzureTypeEnumValues = []LogsArchiveDestinationAzureType{
	LOGSARCHIVEDESTINATIONAZURETYPE_AZURE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsArchiveDestinationAzureType) GetAllowedValues() []LogsArchiveDestinationAzureType {
	return allowedLogsArchiveDestinationAzureTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsArchiveDestinationAzureType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsArchiveDestinationAzureType(value)
	return nil
}

// NewLogsArchiveDestinationAzureTypeFromValue returns a pointer to a valid LogsArchiveDestinationAzureType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsArchiveDestinationAzureTypeFromValue(v string) (*LogsArchiveDestinationAzureType, error) {
	ev := LogsArchiveDestinationAzureType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsArchiveDestinationAzureType: valid values are %v", v, allowedLogsArchiveDestinationAzureTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsArchiveDestinationAzureType) IsValid() bool {
	for _, existing := range allowedLogsArchiveDestinationAzureTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsArchiveDestinationAzureType value.
func (v LogsArchiveDestinationAzureType) Ptr() *LogsArchiveDestinationAzureType {
	return &v
}

// NullableLogsArchiveDestinationAzureType handles when a null is used for LogsArchiveDestinationAzureType.
type NullableLogsArchiveDestinationAzureType struct {
	value *LogsArchiveDestinationAzureType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsArchiveDestinationAzureType) Get() *LogsArchiveDestinationAzureType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsArchiveDestinationAzureType) Set(val *LogsArchiveDestinationAzureType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsArchiveDestinationAzureType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsArchiveDestinationAzureType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsArchiveDestinationAzureType initializes the struct as if Set has been called.
func NewNullableLogsArchiveDestinationAzureType(val *LogsArchiveDestinationAzureType) *NullableLogsArchiveDestinationAzureType {
	return &NullableLogsArchiveDestinationAzureType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsArchiveDestinationAzureType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsArchiveDestinationAzureType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
