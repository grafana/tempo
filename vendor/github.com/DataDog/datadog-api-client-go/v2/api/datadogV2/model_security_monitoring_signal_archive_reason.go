// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringSignalArchiveReason Reason a signal is archived.
type SecurityMonitoringSignalArchiveReason string

// List of SecurityMonitoringSignalArchiveReason.
const (
	SECURITYMONITORINGSIGNALARCHIVEREASON_NONE                   SecurityMonitoringSignalArchiveReason = "none"
	SECURITYMONITORINGSIGNALARCHIVEREASON_FALSE_POSITIVE         SecurityMonitoringSignalArchiveReason = "false_positive"
	SECURITYMONITORINGSIGNALARCHIVEREASON_TESTING_OR_MAINTENANCE SecurityMonitoringSignalArchiveReason = "testing_or_maintenance"
	SECURITYMONITORINGSIGNALARCHIVEREASON_OTHER                  SecurityMonitoringSignalArchiveReason = "other"
)

var allowedSecurityMonitoringSignalArchiveReasonEnumValues = []SecurityMonitoringSignalArchiveReason{
	SECURITYMONITORINGSIGNALARCHIVEREASON_NONE,
	SECURITYMONITORINGSIGNALARCHIVEREASON_FALSE_POSITIVE,
	SECURITYMONITORINGSIGNALARCHIVEREASON_TESTING_OR_MAINTENANCE,
	SECURITYMONITORINGSIGNALARCHIVEREASON_OTHER,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringSignalArchiveReason) GetAllowedValues() []SecurityMonitoringSignalArchiveReason {
	return allowedSecurityMonitoringSignalArchiveReasonEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringSignalArchiveReason) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringSignalArchiveReason(value)
	return nil
}

// NewSecurityMonitoringSignalArchiveReasonFromValue returns a pointer to a valid SecurityMonitoringSignalArchiveReason
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringSignalArchiveReasonFromValue(v string) (*SecurityMonitoringSignalArchiveReason, error) {
	ev := SecurityMonitoringSignalArchiveReason(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringSignalArchiveReason: valid values are %v", v, allowedSecurityMonitoringSignalArchiveReasonEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringSignalArchiveReason) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringSignalArchiveReasonEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringSignalArchiveReason value.
func (v SecurityMonitoringSignalArchiveReason) Ptr() *SecurityMonitoringSignalArchiveReason {
	return &v
}

// NullableSecurityMonitoringSignalArchiveReason handles when a null is used for SecurityMonitoringSignalArchiveReason.
type NullableSecurityMonitoringSignalArchiveReason struct {
	value *SecurityMonitoringSignalArchiveReason
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringSignalArchiveReason) Get() *SecurityMonitoringSignalArchiveReason {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringSignalArchiveReason) Set(val *SecurityMonitoringSignalArchiveReason) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringSignalArchiveReason) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringSignalArchiveReason) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringSignalArchiveReason initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringSignalArchiveReason(val *SecurityMonitoringSignalArchiveReason) *NullableSecurityMonitoringSignalArchiveReason {
	return &NullableSecurityMonitoringSignalArchiveReason{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringSignalArchiveReason) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringSignalArchiveReason) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
