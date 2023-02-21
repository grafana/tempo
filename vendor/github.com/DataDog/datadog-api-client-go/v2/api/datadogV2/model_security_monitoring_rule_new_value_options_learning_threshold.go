// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringRuleNewValueOptionsLearningThreshold A number of occurrences after which signals will be generated for values that weren't learned.
type SecurityMonitoringRuleNewValueOptionsLearningThreshold int32

// List of SecurityMonitoringRuleNewValueOptionsLearningThreshold.
const (
	SECURITYMONITORINGRULENEWVALUEOPTIONSLEARNINGTHRESHOLD_ZERO_OCCURRENCES SecurityMonitoringRuleNewValueOptionsLearningThreshold = 0
	SECURITYMONITORINGRULENEWVALUEOPTIONSLEARNINGTHRESHOLD_ONE_OCCURRENCE   SecurityMonitoringRuleNewValueOptionsLearningThreshold = 1
)

var allowedSecurityMonitoringRuleNewValueOptionsLearningThresholdEnumValues = []SecurityMonitoringRuleNewValueOptionsLearningThreshold{
	SECURITYMONITORINGRULENEWVALUEOPTIONSLEARNINGTHRESHOLD_ZERO_OCCURRENCES,
	SECURITYMONITORINGRULENEWVALUEOPTIONSLEARNINGTHRESHOLD_ONE_OCCURRENCE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringRuleNewValueOptionsLearningThreshold) GetAllowedValues() []SecurityMonitoringRuleNewValueOptionsLearningThreshold {
	return allowedSecurityMonitoringRuleNewValueOptionsLearningThresholdEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringRuleNewValueOptionsLearningThreshold) UnmarshalJSON(src []byte) error {
	var value int32
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringRuleNewValueOptionsLearningThreshold(value)
	return nil
}

// NewSecurityMonitoringRuleNewValueOptionsLearningThresholdFromValue returns a pointer to a valid SecurityMonitoringRuleNewValueOptionsLearningThreshold
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringRuleNewValueOptionsLearningThresholdFromValue(v int32) (*SecurityMonitoringRuleNewValueOptionsLearningThreshold, error) {
	ev := SecurityMonitoringRuleNewValueOptionsLearningThreshold(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringRuleNewValueOptionsLearningThreshold: valid values are %v", v, allowedSecurityMonitoringRuleNewValueOptionsLearningThresholdEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringRuleNewValueOptionsLearningThreshold) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringRuleNewValueOptionsLearningThresholdEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringRuleNewValueOptionsLearningThreshold value.
func (v SecurityMonitoringRuleNewValueOptionsLearningThreshold) Ptr() *SecurityMonitoringRuleNewValueOptionsLearningThreshold {
	return &v
}

// NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold handles when a null is used for SecurityMonitoringRuleNewValueOptionsLearningThreshold.
type NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold struct {
	value *SecurityMonitoringRuleNewValueOptionsLearningThreshold
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold) Get() *SecurityMonitoringRuleNewValueOptionsLearningThreshold {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold) Set(val *SecurityMonitoringRuleNewValueOptionsLearningThreshold) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleNewValueOptionsLearningThreshold initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleNewValueOptionsLearningThreshold(val *SecurityMonitoringRuleNewValueOptionsLearningThreshold) *NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold {
	return &NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleNewValueOptionsLearningThreshold) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
