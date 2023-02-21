// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsMetricType The type of the resource. The value should always be logs_metrics.
type LogsMetricType string

// List of LogsMetricType.
const (
	LOGSMETRICTYPE_LOGS_METRICS LogsMetricType = "logs_metrics"
)

var allowedLogsMetricTypeEnumValues = []LogsMetricType{
	LOGSMETRICTYPE_LOGS_METRICS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsMetricType) GetAllowedValues() []LogsMetricType {
	return allowedLogsMetricTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsMetricType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsMetricType(value)
	return nil
}

// NewLogsMetricTypeFromValue returns a pointer to a valid LogsMetricType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsMetricTypeFromValue(v string) (*LogsMetricType, error) {
	ev := LogsMetricType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsMetricType: valid values are %v", v, allowedLogsMetricTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsMetricType) IsValid() bool {
	for _, existing := range allowedLogsMetricTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsMetricType value.
func (v LogsMetricType) Ptr() *LogsMetricType {
	return &v
}

// NullableLogsMetricType handles when a null is used for LogsMetricType.
type NullableLogsMetricType struct {
	value *LogsMetricType
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsMetricType) Get() *LogsMetricType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsMetricType) Set(val *LogsMetricType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsMetricType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsMetricType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsMetricType initializes the struct as if Set has been called.
func NewNullableLogsMetricType(val *LogsMetricType) *NullableLogsMetricType {
	return &NullableLogsMetricType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsMetricType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsMetricType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
