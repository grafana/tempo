// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MonitorConfigPolicyResourceType Monitor configuration policy resource type.
type MonitorConfigPolicyResourceType string

// List of MonitorConfigPolicyResourceType.
const (
	MONITORCONFIGPOLICYRESOURCETYPE_MONITOR_CONFIG_POLICY MonitorConfigPolicyResourceType = "monitor-config-policy"
)

var allowedMonitorConfigPolicyResourceTypeEnumValues = []MonitorConfigPolicyResourceType{
	MONITORCONFIGPOLICYRESOURCETYPE_MONITOR_CONFIG_POLICY,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MonitorConfigPolicyResourceType) GetAllowedValues() []MonitorConfigPolicyResourceType {
	return allowedMonitorConfigPolicyResourceTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MonitorConfigPolicyResourceType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MonitorConfigPolicyResourceType(value)
	return nil
}

// NewMonitorConfigPolicyResourceTypeFromValue returns a pointer to a valid MonitorConfigPolicyResourceType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMonitorConfigPolicyResourceTypeFromValue(v string) (*MonitorConfigPolicyResourceType, error) {
	ev := MonitorConfigPolicyResourceType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MonitorConfigPolicyResourceType: valid values are %v", v, allowedMonitorConfigPolicyResourceTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MonitorConfigPolicyResourceType) IsValid() bool {
	for _, existing := range allowedMonitorConfigPolicyResourceTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MonitorConfigPolicyResourceType value.
func (v MonitorConfigPolicyResourceType) Ptr() *MonitorConfigPolicyResourceType {
	return &v
}

// NullableMonitorConfigPolicyResourceType handles when a null is used for MonitorConfigPolicyResourceType.
type NullableMonitorConfigPolicyResourceType struct {
	value *MonitorConfigPolicyResourceType
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorConfigPolicyResourceType) Get() *MonitorConfigPolicyResourceType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorConfigPolicyResourceType) Set(val *MonitorConfigPolicyResourceType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorConfigPolicyResourceType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMonitorConfigPolicyResourceType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorConfigPolicyResourceType initializes the struct as if Set has been called.
func NewNullableMonitorConfigPolicyResourceType(val *MonitorConfigPolicyResourceType) *NullableMonitorConfigPolicyResourceType {
	return &NullableMonitorConfigPolicyResourceType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorConfigPolicyResourceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorConfigPolicyResourceType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
