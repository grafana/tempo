// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// MonitorOptionsNotificationPresets Toggles the display of additional content sent in the monitor notification.
type MonitorOptionsNotificationPresets string

// List of MonitorOptionsNotificationPresets.
const (
	MONITOROPTIONSNOTIFICATIONPRESETS_SHOW_ALL     MonitorOptionsNotificationPresets = "show_all"
	MONITOROPTIONSNOTIFICATIONPRESETS_HIDE_QUERY   MonitorOptionsNotificationPresets = "hide_query"
	MONITOROPTIONSNOTIFICATIONPRESETS_HIDE_HANDLES MonitorOptionsNotificationPresets = "hide_handles"
	MONITOROPTIONSNOTIFICATIONPRESETS_HIDE_ALL     MonitorOptionsNotificationPresets = "hide_all"
)

var allowedMonitorOptionsNotificationPresetsEnumValues = []MonitorOptionsNotificationPresets{
	MONITOROPTIONSNOTIFICATIONPRESETS_SHOW_ALL,
	MONITOROPTIONSNOTIFICATIONPRESETS_HIDE_QUERY,
	MONITOROPTIONSNOTIFICATIONPRESETS_HIDE_HANDLES,
	MONITOROPTIONSNOTIFICATIONPRESETS_HIDE_ALL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MonitorOptionsNotificationPresets) GetAllowedValues() []MonitorOptionsNotificationPresets {
	return allowedMonitorOptionsNotificationPresetsEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MonitorOptionsNotificationPresets) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MonitorOptionsNotificationPresets(value)
	return nil
}

// NewMonitorOptionsNotificationPresetsFromValue returns a pointer to a valid MonitorOptionsNotificationPresets
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMonitorOptionsNotificationPresetsFromValue(v string) (*MonitorOptionsNotificationPresets, error) {
	ev := MonitorOptionsNotificationPresets(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MonitorOptionsNotificationPresets: valid values are %v", v, allowedMonitorOptionsNotificationPresetsEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MonitorOptionsNotificationPresets) IsValid() bool {
	for _, existing := range allowedMonitorOptionsNotificationPresetsEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MonitorOptionsNotificationPresets value.
func (v MonitorOptionsNotificationPresets) Ptr() *MonitorOptionsNotificationPresets {
	return &v
}

// NullableMonitorOptionsNotificationPresets handles when a null is used for MonitorOptionsNotificationPresets.
type NullableMonitorOptionsNotificationPresets struct {
	value *MonitorOptionsNotificationPresets
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorOptionsNotificationPresets) Get() *MonitorOptionsNotificationPresets {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorOptionsNotificationPresets) Set(val *MonitorOptionsNotificationPresets) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorOptionsNotificationPresets) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMonitorOptionsNotificationPresets) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorOptionsNotificationPresets initializes the struct as if Set has been called.
func NewNullableMonitorOptionsNotificationPresets(val *MonitorOptionsNotificationPresets) *NullableMonitorOptionsNotificationPresets {
	return &NullableMonitorOptionsNotificationPresets{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorOptionsNotificationPresets) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorOptionsNotificationPresets) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
