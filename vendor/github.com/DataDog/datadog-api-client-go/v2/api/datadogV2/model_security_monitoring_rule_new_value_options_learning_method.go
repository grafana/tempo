// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringRuleNewValueOptionsLearningMethod The learning method used to determine when signals should be generated for values that weren't learned.
type SecurityMonitoringRuleNewValueOptionsLearningMethod string

// List of SecurityMonitoringRuleNewValueOptionsLearningMethod.
const (
	SECURITYMONITORINGRULENEWVALUEOPTIONSLEARNINGMETHOD_DURATION  SecurityMonitoringRuleNewValueOptionsLearningMethod = "duration"
	SECURITYMONITORINGRULENEWVALUEOPTIONSLEARNINGMETHOD_THRESHOLD SecurityMonitoringRuleNewValueOptionsLearningMethod = "threshold"
)

var allowedSecurityMonitoringRuleNewValueOptionsLearningMethodEnumValues = []SecurityMonitoringRuleNewValueOptionsLearningMethod{
	SECURITYMONITORINGRULENEWVALUEOPTIONSLEARNINGMETHOD_DURATION,
	SECURITYMONITORINGRULENEWVALUEOPTIONSLEARNINGMETHOD_THRESHOLD,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringRuleNewValueOptionsLearningMethod) GetAllowedValues() []SecurityMonitoringRuleNewValueOptionsLearningMethod {
	return allowedSecurityMonitoringRuleNewValueOptionsLearningMethodEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringRuleNewValueOptionsLearningMethod) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringRuleNewValueOptionsLearningMethod(value)
	return nil
}

// NewSecurityMonitoringRuleNewValueOptionsLearningMethodFromValue returns a pointer to a valid SecurityMonitoringRuleNewValueOptionsLearningMethod
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringRuleNewValueOptionsLearningMethodFromValue(v string) (*SecurityMonitoringRuleNewValueOptionsLearningMethod, error) {
	ev := SecurityMonitoringRuleNewValueOptionsLearningMethod(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringRuleNewValueOptionsLearningMethod: valid values are %v", v, allowedSecurityMonitoringRuleNewValueOptionsLearningMethodEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringRuleNewValueOptionsLearningMethod) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringRuleNewValueOptionsLearningMethodEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringRuleNewValueOptionsLearningMethod value.
func (v SecurityMonitoringRuleNewValueOptionsLearningMethod) Ptr() *SecurityMonitoringRuleNewValueOptionsLearningMethod {
	return &v
}

// NullableSecurityMonitoringRuleNewValueOptionsLearningMethod handles when a null is used for SecurityMonitoringRuleNewValueOptionsLearningMethod.
type NullableSecurityMonitoringRuleNewValueOptionsLearningMethod struct {
	value *SecurityMonitoringRuleNewValueOptionsLearningMethod
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleNewValueOptionsLearningMethod) Get() *SecurityMonitoringRuleNewValueOptionsLearningMethod {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleNewValueOptionsLearningMethod) Set(val *SecurityMonitoringRuleNewValueOptionsLearningMethod) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleNewValueOptionsLearningMethod) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringRuleNewValueOptionsLearningMethod) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleNewValueOptionsLearningMethod initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleNewValueOptionsLearningMethod(val *SecurityMonitoringRuleNewValueOptionsLearningMethod) *NullableSecurityMonitoringRuleNewValueOptionsLearningMethod {
	return &NullableSecurityMonitoringRuleNewValueOptionsLearningMethod{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleNewValueOptionsLearningMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleNewValueOptionsLearningMethod) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
