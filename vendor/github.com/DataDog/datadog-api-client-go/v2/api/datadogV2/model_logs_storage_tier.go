// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsStorageTier Specifies storage type as indexes or online-archives
type LogsStorageTier string

// List of LogsStorageTier.
const (
	LOGSSTORAGETIER_INDEXES         LogsStorageTier = "indexes"
	LOGSSTORAGETIER_ONLINE_ARCHIVES LogsStorageTier = "online-archives"
)

var allowedLogsStorageTierEnumValues = []LogsStorageTier{
	LOGSSTORAGETIER_INDEXES,
	LOGSSTORAGETIER_ONLINE_ARCHIVES,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsStorageTier) GetAllowedValues() []LogsStorageTier {
	return allowedLogsStorageTierEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsStorageTier) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsStorageTier(value)
	return nil
}

// NewLogsStorageTierFromValue returns a pointer to a valid LogsStorageTier
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsStorageTierFromValue(v string) (*LogsStorageTier, error) {
	ev := LogsStorageTier(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsStorageTier: valid values are %v", v, allowedLogsStorageTierEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsStorageTier) IsValid() bool {
	for _, existing := range allowedLogsStorageTierEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsStorageTier value.
func (v LogsStorageTier) Ptr() *LogsStorageTier {
	return &v
}

// NullableLogsStorageTier handles when a null is used for LogsStorageTier.
type NullableLogsStorageTier struct {
	value *LogsStorageTier
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsStorageTier) Get() *LogsStorageTier {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsStorageTier) Set(val *LogsStorageTier) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsStorageTier) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsStorageTier) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsStorageTier initializes the struct as if Set has been called.
func NewNullableLogsStorageTier(val *LogsStorageTier) *NullableLogsStorageTier {
	return &NullableLogsStorageTier{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsStorageTier) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsStorageTier) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
