// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveDestinationS3Type Type of the S3 archive destination.
type LogsArchiveDestinationS3Type string

// List of LogsArchiveDestinationS3Type.
const (
	LOGSARCHIVEDESTINATIONS3TYPE_S3 LogsArchiveDestinationS3Type = "s3"
)

var allowedLogsArchiveDestinationS3TypeEnumValues = []LogsArchiveDestinationS3Type{
	LOGSARCHIVEDESTINATIONS3TYPE_S3,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsArchiveDestinationS3Type) GetAllowedValues() []LogsArchiveDestinationS3Type {
	return allowedLogsArchiveDestinationS3TypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsArchiveDestinationS3Type) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsArchiveDestinationS3Type(value)
	return nil
}

// NewLogsArchiveDestinationS3TypeFromValue returns a pointer to a valid LogsArchiveDestinationS3Type
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsArchiveDestinationS3TypeFromValue(v string) (*LogsArchiveDestinationS3Type, error) {
	ev := LogsArchiveDestinationS3Type(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsArchiveDestinationS3Type: valid values are %v", v, allowedLogsArchiveDestinationS3TypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsArchiveDestinationS3Type) IsValid() bool {
	for _, existing := range allowedLogsArchiveDestinationS3TypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsArchiveDestinationS3Type value.
func (v LogsArchiveDestinationS3Type) Ptr() *LogsArchiveDestinationS3Type {
	return &v
}

// NullableLogsArchiveDestinationS3Type handles when a null is used for LogsArchiveDestinationS3Type.
type NullableLogsArchiveDestinationS3Type struct {
	value *LogsArchiveDestinationS3Type
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsArchiveDestinationS3Type) Get() *LogsArchiveDestinationS3Type {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsArchiveDestinationS3Type) Set(val *LogsArchiveDestinationS3Type) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsArchiveDestinationS3Type) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsArchiveDestinationS3Type) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsArchiveDestinationS3Type initializes the struct as if Set has been called.
func NewNullableLogsArchiveDestinationS3Type(val *LogsArchiveDestinationS3Type) *NullableLogsArchiveDestinationS3Type {
	return &NullableLogsArchiveDestinationS3Type{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsArchiveDestinationS3Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsArchiveDestinationS3Type) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
