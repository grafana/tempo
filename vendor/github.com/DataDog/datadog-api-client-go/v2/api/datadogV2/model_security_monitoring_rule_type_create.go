// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringRuleTypeCreate The rule type.
type SecurityMonitoringRuleTypeCreate string

// List of SecurityMonitoringRuleTypeCreate.
const (
	SECURITYMONITORINGRULETYPECREATE_LOG_DETECTION     SecurityMonitoringRuleTypeCreate = "log_detection"
	SECURITYMONITORINGRULETYPECREATE_WORKLOAD_SECURITY SecurityMonitoringRuleTypeCreate = "workload_security"
)

var allowedSecurityMonitoringRuleTypeCreateEnumValues = []SecurityMonitoringRuleTypeCreate{
	SECURITYMONITORINGRULETYPECREATE_LOG_DETECTION,
	SECURITYMONITORINGRULETYPECREATE_WORKLOAD_SECURITY,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringRuleTypeCreate) GetAllowedValues() []SecurityMonitoringRuleTypeCreate {
	return allowedSecurityMonitoringRuleTypeCreateEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringRuleTypeCreate) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringRuleTypeCreate(value)
	return nil
}

// NewSecurityMonitoringRuleTypeCreateFromValue returns a pointer to a valid SecurityMonitoringRuleTypeCreate
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringRuleTypeCreateFromValue(v string) (*SecurityMonitoringRuleTypeCreate, error) {
	ev := SecurityMonitoringRuleTypeCreate(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringRuleTypeCreate: valid values are %v", v, allowedSecurityMonitoringRuleTypeCreateEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringRuleTypeCreate) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringRuleTypeCreateEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringRuleTypeCreate value.
func (v SecurityMonitoringRuleTypeCreate) Ptr() *SecurityMonitoringRuleTypeCreate {
	return &v
}

// NullableSecurityMonitoringRuleTypeCreate handles when a null is used for SecurityMonitoringRuleTypeCreate.
type NullableSecurityMonitoringRuleTypeCreate struct {
	value *SecurityMonitoringRuleTypeCreate
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleTypeCreate) Get() *SecurityMonitoringRuleTypeCreate {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleTypeCreate) Set(val *SecurityMonitoringRuleTypeCreate) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleTypeCreate) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringRuleTypeCreate) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleTypeCreate initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleTypeCreate(val *SecurityMonitoringRuleTypeCreate) *NullableSecurityMonitoringRuleTypeCreate {
	return &NullableSecurityMonitoringRuleTypeCreate{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleTypeCreate) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleTypeCreate) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
