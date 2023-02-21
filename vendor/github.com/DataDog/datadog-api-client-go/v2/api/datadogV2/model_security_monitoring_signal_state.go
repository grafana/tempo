// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringSignalState The new triage state of the signal.
type SecurityMonitoringSignalState string

// List of SecurityMonitoringSignalState.
const (
	SECURITYMONITORINGSIGNALSTATE_OPEN         SecurityMonitoringSignalState = "open"
	SECURITYMONITORINGSIGNALSTATE_ARCHIVED     SecurityMonitoringSignalState = "archived"
	SECURITYMONITORINGSIGNALSTATE_UNDER_REVIEW SecurityMonitoringSignalState = "under_review"
)

var allowedSecurityMonitoringSignalStateEnumValues = []SecurityMonitoringSignalState{
	SECURITYMONITORINGSIGNALSTATE_OPEN,
	SECURITYMONITORINGSIGNALSTATE_ARCHIVED,
	SECURITYMONITORINGSIGNALSTATE_UNDER_REVIEW,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringSignalState) GetAllowedValues() []SecurityMonitoringSignalState {
	return allowedSecurityMonitoringSignalStateEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringSignalState) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringSignalState(value)
	return nil
}

// NewSecurityMonitoringSignalStateFromValue returns a pointer to a valid SecurityMonitoringSignalState
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringSignalStateFromValue(v string) (*SecurityMonitoringSignalState, error) {
	ev := SecurityMonitoringSignalState(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringSignalState: valid values are %v", v, allowedSecurityMonitoringSignalStateEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringSignalState) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringSignalStateEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringSignalState value.
func (v SecurityMonitoringSignalState) Ptr() *SecurityMonitoringSignalState {
	return &v
}

// NullableSecurityMonitoringSignalState handles when a null is used for SecurityMonitoringSignalState.
type NullableSecurityMonitoringSignalState struct {
	value *SecurityMonitoringSignalState
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringSignalState) Get() *SecurityMonitoringSignalState {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringSignalState) Set(val *SecurityMonitoringSignalState) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringSignalState) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringSignalState) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringSignalState initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringSignalState(val *SecurityMonitoringSignalState) *NullableSecurityMonitoringSignalState {
	return &NullableSecurityMonitoringSignalState{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringSignalState) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringSignalState) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
