// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringRuleNewValueOptionsForgetAfter The duration in days after which a learned value is forgotten.
type SecurityMonitoringRuleNewValueOptionsForgetAfter int32

// List of SecurityMonitoringRuleNewValueOptionsForgetAfter.
const (
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_ONE_DAY     SecurityMonitoringRuleNewValueOptionsForgetAfter = 1
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_TWO_DAYS    SecurityMonitoringRuleNewValueOptionsForgetAfter = 2
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_ONE_WEEK    SecurityMonitoringRuleNewValueOptionsForgetAfter = 7
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_TWO_WEEKS   SecurityMonitoringRuleNewValueOptionsForgetAfter = 14
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_THREE_WEEKS SecurityMonitoringRuleNewValueOptionsForgetAfter = 21
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_FOUR_WEEKS  SecurityMonitoringRuleNewValueOptionsForgetAfter = 28
)

var allowedSecurityMonitoringRuleNewValueOptionsForgetAfterEnumValues = []SecurityMonitoringRuleNewValueOptionsForgetAfter{
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_ONE_DAY,
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_TWO_DAYS,
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_ONE_WEEK,
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_TWO_WEEKS,
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_THREE_WEEKS,
	SECURITYMONITORINGRULENEWVALUEOPTIONSFORGETAFTER_FOUR_WEEKS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringRuleNewValueOptionsForgetAfter) GetAllowedValues() []SecurityMonitoringRuleNewValueOptionsForgetAfter {
	return allowedSecurityMonitoringRuleNewValueOptionsForgetAfterEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringRuleNewValueOptionsForgetAfter) UnmarshalJSON(src []byte) error {
	var value int32
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringRuleNewValueOptionsForgetAfter(value)
	return nil
}

// NewSecurityMonitoringRuleNewValueOptionsForgetAfterFromValue returns a pointer to a valid SecurityMonitoringRuleNewValueOptionsForgetAfter
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringRuleNewValueOptionsForgetAfterFromValue(v int32) (*SecurityMonitoringRuleNewValueOptionsForgetAfter, error) {
	ev := SecurityMonitoringRuleNewValueOptionsForgetAfter(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringRuleNewValueOptionsForgetAfter: valid values are %v", v, allowedSecurityMonitoringRuleNewValueOptionsForgetAfterEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringRuleNewValueOptionsForgetAfter) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringRuleNewValueOptionsForgetAfterEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringRuleNewValueOptionsForgetAfter value.
func (v SecurityMonitoringRuleNewValueOptionsForgetAfter) Ptr() *SecurityMonitoringRuleNewValueOptionsForgetAfter {
	return &v
}

// NullableSecurityMonitoringRuleNewValueOptionsForgetAfter handles when a null is used for SecurityMonitoringRuleNewValueOptionsForgetAfter.
type NullableSecurityMonitoringRuleNewValueOptionsForgetAfter struct {
	value *SecurityMonitoringRuleNewValueOptionsForgetAfter
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleNewValueOptionsForgetAfter) Get() *SecurityMonitoringRuleNewValueOptionsForgetAfter {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleNewValueOptionsForgetAfter) Set(val *SecurityMonitoringRuleNewValueOptionsForgetAfter) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleNewValueOptionsForgetAfter) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringRuleNewValueOptionsForgetAfter) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleNewValueOptionsForgetAfter initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleNewValueOptionsForgetAfter(val *SecurityMonitoringRuleNewValueOptionsForgetAfter) *NullableSecurityMonitoringRuleNewValueOptionsForgetAfter {
	return &NullableSecurityMonitoringRuleNewValueOptionsForgetAfter{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleNewValueOptionsForgetAfter) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleNewValueOptionsForgetAfter) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
