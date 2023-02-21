// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MonitorConfigPolicyType The monitor configuration policy type.
type MonitorConfigPolicyType string

// List of MonitorConfigPolicyType.
const (
	MONITORCONFIGPOLICYTYPE_TAG MonitorConfigPolicyType = "tag"
)

var allowedMonitorConfigPolicyTypeEnumValues = []MonitorConfigPolicyType{
	MONITORCONFIGPOLICYTYPE_TAG,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MonitorConfigPolicyType) GetAllowedValues() []MonitorConfigPolicyType {
	return allowedMonitorConfigPolicyTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MonitorConfigPolicyType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MonitorConfigPolicyType(value)
	return nil
}

// NewMonitorConfigPolicyTypeFromValue returns a pointer to a valid MonitorConfigPolicyType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMonitorConfigPolicyTypeFromValue(v string) (*MonitorConfigPolicyType, error) {
	ev := MonitorConfigPolicyType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MonitorConfigPolicyType: valid values are %v", v, allowedMonitorConfigPolicyTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MonitorConfigPolicyType) IsValid() bool {
	for _, existing := range allowedMonitorConfigPolicyTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MonitorConfigPolicyType value.
func (v MonitorConfigPolicyType) Ptr() *MonitorConfigPolicyType {
	return &v
}

// NullableMonitorConfigPolicyType handles when a null is used for MonitorConfigPolicyType.
type NullableMonitorConfigPolicyType struct {
	value *MonitorConfigPolicyType
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorConfigPolicyType) Get() *MonitorConfigPolicyType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorConfigPolicyType) Set(val *MonitorConfigPolicyType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorConfigPolicyType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMonitorConfigPolicyType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorConfigPolicyType initializes the struct as if Set has been called.
func NewNullableMonitorConfigPolicyType(val *MonitorConfigPolicyType) *NullableMonitorConfigPolicyType {
	return &NullableMonitorConfigPolicyType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorConfigPolicyType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorConfigPolicyType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
