// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringRuleSeverity Severity of the Security Signal.
type SecurityMonitoringRuleSeverity string

// List of SecurityMonitoringRuleSeverity.
const (
	SECURITYMONITORINGRULESEVERITY_INFO     SecurityMonitoringRuleSeverity = "info"
	SECURITYMONITORINGRULESEVERITY_LOW      SecurityMonitoringRuleSeverity = "low"
	SECURITYMONITORINGRULESEVERITY_MEDIUM   SecurityMonitoringRuleSeverity = "medium"
	SECURITYMONITORINGRULESEVERITY_HIGH     SecurityMonitoringRuleSeverity = "high"
	SECURITYMONITORINGRULESEVERITY_CRITICAL SecurityMonitoringRuleSeverity = "critical"
)

var allowedSecurityMonitoringRuleSeverityEnumValues = []SecurityMonitoringRuleSeverity{
	SECURITYMONITORINGRULESEVERITY_INFO,
	SECURITYMONITORINGRULESEVERITY_LOW,
	SECURITYMONITORINGRULESEVERITY_MEDIUM,
	SECURITYMONITORINGRULESEVERITY_HIGH,
	SECURITYMONITORINGRULESEVERITY_CRITICAL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringRuleSeverity) GetAllowedValues() []SecurityMonitoringRuleSeverity {
	return allowedSecurityMonitoringRuleSeverityEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringRuleSeverity) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringRuleSeverity(value)
	return nil
}

// NewSecurityMonitoringRuleSeverityFromValue returns a pointer to a valid SecurityMonitoringRuleSeverity
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringRuleSeverityFromValue(v string) (*SecurityMonitoringRuleSeverity, error) {
	ev := SecurityMonitoringRuleSeverity(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringRuleSeverity: valid values are %v", v, allowedSecurityMonitoringRuleSeverityEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringRuleSeverity) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringRuleSeverityEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringRuleSeverity value.
func (v SecurityMonitoringRuleSeverity) Ptr() *SecurityMonitoringRuleSeverity {
	return &v
}

// NullableSecurityMonitoringRuleSeverity handles when a null is used for SecurityMonitoringRuleSeverity.
type NullableSecurityMonitoringRuleSeverity struct {
	value *SecurityMonitoringRuleSeverity
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleSeverity) Get() *SecurityMonitoringRuleSeverity {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleSeverity) Set(val *SecurityMonitoringRuleSeverity) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleSeverity) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringRuleSeverity) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleSeverity initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleSeverity(val *SecurityMonitoringRuleSeverity) *NullableSecurityMonitoringRuleSeverity {
	return &NullableSecurityMonitoringRuleSeverity{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleSeverity) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleSeverity) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
