// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringSignalType The type of event.
type SecurityMonitoringSignalType string

// List of SecurityMonitoringSignalType.
const (
	SECURITYMONITORINGSIGNALTYPE_SIGNAL SecurityMonitoringSignalType = "signal"
)

var allowedSecurityMonitoringSignalTypeEnumValues = []SecurityMonitoringSignalType{
	SECURITYMONITORINGSIGNALTYPE_SIGNAL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringSignalType) GetAllowedValues() []SecurityMonitoringSignalType {
	return allowedSecurityMonitoringSignalTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringSignalType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringSignalType(value)
	return nil
}

// NewSecurityMonitoringSignalTypeFromValue returns a pointer to a valid SecurityMonitoringSignalType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringSignalTypeFromValue(v string) (*SecurityMonitoringSignalType, error) {
	ev := SecurityMonitoringSignalType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringSignalType: valid values are %v", v, allowedSecurityMonitoringSignalTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringSignalType) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringSignalTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringSignalType value.
func (v SecurityMonitoringSignalType) Ptr() *SecurityMonitoringSignalType {
	return &v
}

// NullableSecurityMonitoringSignalType handles when a null is used for SecurityMonitoringSignalType.
type NullableSecurityMonitoringSignalType struct {
	value *SecurityMonitoringSignalType
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringSignalType) Get() *SecurityMonitoringSignalType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringSignalType) Set(val *SecurityMonitoringSignalType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringSignalType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringSignalType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringSignalType initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringSignalType(val *SecurityMonitoringSignalType) *NullableSecurityMonitoringSignalType {
	return &NullableSecurityMonitoringSignalType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringSignalType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringSignalType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
