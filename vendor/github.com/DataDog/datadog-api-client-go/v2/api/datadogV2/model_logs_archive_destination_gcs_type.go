// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveDestinationGCSType Type of the GCS archive destination.
type LogsArchiveDestinationGCSType string

// List of LogsArchiveDestinationGCSType.
const (
	LOGSARCHIVEDESTINATIONGCSTYPE_GCS LogsArchiveDestinationGCSType = "gcs"
)

var allowedLogsArchiveDestinationGCSTypeEnumValues = []LogsArchiveDestinationGCSType{
	LOGSARCHIVEDESTINATIONGCSTYPE_GCS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsArchiveDestinationGCSType) GetAllowedValues() []LogsArchiveDestinationGCSType {
	return allowedLogsArchiveDestinationGCSTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsArchiveDestinationGCSType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsArchiveDestinationGCSType(value)
	return nil
}

// NewLogsArchiveDestinationGCSTypeFromValue returns a pointer to a valid LogsArchiveDestinationGCSType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsArchiveDestinationGCSTypeFromValue(v string) (*LogsArchiveDestinationGCSType, error) {
	ev := LogsArchiveDestinationGCSType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsArchiveDestinationGCSType: valid values are %v", v, allowedLogsArchiveDestinationGCSTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsArchiveDestinationGCSType) IsValid() bool {
	for _, existing := range allowedLogsArchiveDestinationGCSTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsArchiveDestinationGCSType value.
func (v LogsArchiveDestinationGCSType) Ptr() *LogsArchiveDestinationGCSType {
	return &v
}

// NullableLogsArchiveDestinationGCSType handles when a null is used for LogsArchiveDestinationGCSType.
type NullableLogsArchiveDestinationGCSType struct {
	value *LogsArchiveDestinationGCSType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsArchiveDestinationGCSType) Get() *LogsArchiveDestinationGCSType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsArchiveDestinationGCSType) Set(val *LogsArchiveDestinationGCSType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsArchiveDestinationGCSType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsArchiveDestinationGCSType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsArchiveDestinationGCSType initializes the struct as if Set has been called.
func NewNullableLogsArchiveDestinationGCSType(val *LogsArchiveDestinationGCSType) *NullableLogsArchiveDestinationGCSType {
	return &NullableLogsArchiveDestinationGCSType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsArchiveDestinationGCSType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsArchiveDestinationGCSType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
