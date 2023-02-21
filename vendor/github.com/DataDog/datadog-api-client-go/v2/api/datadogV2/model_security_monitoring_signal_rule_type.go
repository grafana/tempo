// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringSignalRuleType The rule type.
type SecurityMonitoringSignalRuleType string

// List of SecurityMonitoringSignalRuleType.
const (
	SECURITYMONITORINGSIGNALRULETYPE_SIGNAL_CORRELATION SecurityMonitoringSignalRuleType = "signal_correlation"
)

var allowedSecurityMonitoringSignalRuleTypeEnumValues = []SecurityMonitoringSignalRuleType{
	SECURITYMONITORINGSIGNALRULETYPE_SIGNAL_CORRELATION,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringSignalRuleType) GetAllowedValues() []SecurityMonitoringSignalRuleType {
	return allowedSecurityMonitoringSignalRuleTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringSignalRuleType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringSignalRuleType(value)
	return nil
}

// NewSecurityMonitoringSignalRuleTypeFromValue returns a pointer to a valid SecurityMonitoringSignalRuleType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringSignalRuleTypeFromValue(v string) (*SecurityMonitoringSignalRuleType, error) {
	ev := SecurityMonitoringSignalRuleType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringSignalRuleType: valid values are %v", v, allowedSecurityMonitoringSignalRuleTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringSignalRuleType) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringSignalRuleTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringSignalRuleType value.
func (v SecurityMonitoringSignalRuleType) Ptr() *SecurityMonitoringSignalRuleType {
	return &v
}

// NullableSecurityMonitoringSignalRuleType handles when a null is used for SecurityMonitoringSignalRuleType.
type NullableSecurityMonitoringSignalRuleType struct {
	value *SecurityMonitoringSignalRuleType
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringSignalRuleType) Get() *SecurityMonitoringSignalRuleType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringSignalRuleType) Set(val *SecurityMonitoringSignalRuleType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringSignalRuleType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringSignalRuleType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringSignalRuleType initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringSignalRuleType(val *SecurityMonitoringSignalRuleType) *NullableSecurityMonitoringSignalRuleType {
	return &NullableSecurityMonitoringSignalRuleType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringSignalRuleType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringSignalRuleType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
