// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringRuleKeepAlive Once a signal is generated, the signal will remain “open” if a case is matched at least once within
// this keep alive window.
type SecurityMonitoringRuleKeepAlive int32

// List of SecurityMonitoringRuleKeepAlive.
const (
	SECURITYMONITORINGRULEKEEPALIVE_ZERO_MINUTES    SecurityMonitoringRuleKeepAlive = 0
	SECURITYMONITORINGRULEKEEPALIVE_ONE_MINUTE      SecurityMonitoringRuleKeepAlive = 60
	SECURITYMONITORINGRULEKEEPALIVE_FIVE_MINUTES    SecurityMonitoringRuleKeepAlive = 300
	SECURITYMONITORINGRULEKEEPALIVE_TEN_MINUTES     SecurityMonitoringRuleKeepAlive = 600
	SECURITYMONITORINGRULEKEEPALIVE_FIFTEEN_MINUTES SecurityMonitoringRuleKeepAlive = 900
	SECURITYMONITORINGRULEKEEPALIVE_THIRTY_MINUTES  SecurityMonitoringRuleKeepAlive = 1800
	SECURITYMONITORINGRULEKEEPALIVE_ONE_HOUR        SecurityMonitoringRuleKeepAlive = 3600
	SECURITYMONITORINGRULEKEEPALIVE_TWO_HOURS       SecurityMonitoringRuleKeepAlive = 7200
	SECURITYMONITORINGRULEKEEPALIVE_THREE_HOURS     SecurityMonitoringRuleKeepAlive = 10800
	SECURITYMONITORINGRULEKEEPALIVE_SIX_HOURS       SecurityMonitoringRuleKeepAlive = 21600
)

var allowedSecurityMonitoringRuleKeepAliveEnumValues = []SecurityMonitoringRuleKeepAlive{
	SECURITYMONITORINGRULEKEEPALIVE_ZERO_MINUTES,
	SECURITYMONITORINGRULEKEEPALIVE_ONE_MINUTE,
	SECURITYMONITORINGRULEKEEPALIVE_FIVE_MINUTES,
	SECURITYMONITORINGRULEKEEPALIVE_TEN_MINUTES,
	SECURITYMONITORINGRULEKEEPALIVE_FIFTEEN_MINUTES,
	SECURITYMONITORINGRULEKEEPALIVE_THIRTY_MINUTES,
	SECURITYMONITORINGRULEKEEPALIVE_ONE_HOUR,
	SECURITYMONITORINGRULEKEEPALIVE_TWO_HOURS,
	SECURITYMONITORINGRULEKEEPALIVE_THREE_HOURS,
	SECURITYMONITORINGRULEKEEPALIVE_SIX_HOURS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringRuleKeepAlive) GetAllowedValues() []SecurityMonitoringRuleKeepAlive {
	return allowedSecurityMonitoringRuleKeepAliveEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringRuleKeepAlive) UnmarshalJSON(src []byte) error {
	var value int32
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringRuleKeepAlive(value)
	return nil
}

// NewSecurityMonitoringRuleKeepAliveFromValue returns a pointer to a valid SecurityMonitoringRuleKeepAlive
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringRuleKeepAliveFromValue(v int32) (*SecurityMonitoringRuleKeepAlive, error) {
	ev := SecurityMonitoringRuleKeepAlive(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringRuleKeepAlive: valid values are %v", v, allowedSecurityMonitoringRuleKeepAliveEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringRuleKeepAlive) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringRuleKeepAliveEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringRuleKeepAlive value.
func (v SecurityMonitoringRuleKeepAlive) Ptr() *SecurityMonitoringRuleKeepAlive {
	return &v
}

// NullableSecurityMonitoringRuleKeepAlive handles when a null is used for SecurityMonitoringRuleKeepAlive.
type NullableSecurityMonitoringRuleKeepAlive struct {
	value *SecurityMonitoringRuleKeepAlive
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleKeepAlive) Get() *SecurityMonitoringRuleKeepAlive {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleKeepAlive) Set(val *SecurityMonitoringRuleKeepAlive) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleKeepAlive) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringRuleKeepAlive) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleKeepAlive initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleKeepAlive(val *SecurityMonitoringRuleKeepAlive) *NullableSecurityMonitoringRuleKeepAlive {
	return &NullableSecurityMonitoringRuleKeepAlive{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleKeepAlive) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleKeepAlive) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
