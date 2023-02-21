// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringRuleQueryAggregation The aggregation type.
type SecurityMonitoringRuleQueryAggregation string

// List of SecurityMonitoringRuleQueryAggregation.
const (
	SECURITYMONITORINGRULEQUERYAGGREGATION_COUNT       SecurityMonitoringRuleQueryAggregation = "count"
	SECURITYMONITORINGRULEQUERYAGGREGATION_CARDINALITY SecurityMonitoringRuleQueryAggregation = "cardinality"
	SECURITYMONITORINGRULEQUERYAGGREGATION_SUM         SecurityMonitoringRuleQueryAggregation = "sum"
	SECURITYMONITORINGRULEQUERYAGGREGATION_MAX         SecurityMonitoringRuleQueryAggregation = "max"
	SECURITYMONITORINGRULEQUERYAGGREGATION_NEW_VALUE   SecurityMonitoringRuleQueryAggregation = "new_value"
	SECURITYMONITORINGRULEQUERYAGGREGATION_GEO_DATA    SecurityMonitoringRuleQueryAggregation = "geo_data"
	SECURITYMONITORINGRULEQUERYAGGREGATION_EVENT_COUNT SecurityMonitoringRuleQueryAggregation = "event_count"
	SECURITYMONITORINGRULEQUERYAGGREGATION_NONE        SecurityMonitoringRuleQueryAggregation = "none"
)

var allowedSecurityMonitoringRuleQueryAggregationEnumValues = []SecurityMonitoringRuleQueryAggregation{
	SECURITYMONITORINGRULEQUERYAGGREGATION_COUNT,
	SECURITYMONITORINGRULEQUERYAGGREGATION_CARDINALITY,
	SECURITYMONITORINGRULEQUERYAGGREGATION_SUM,
	SECURITYMONITORINGRULEQUERYAGGREGATION_MAX,
	SECURITYMONITORINGRULEQUERYAGGREGATION_NEW_VALUE,
	SECURITYMONITORINGRULEQUERYAGGREGATION_GEO_DATA,
	SECURITYMONITORINGRULEQUERYAGGREGATION_EVENT_COUNT,
	SECURITYMONITORINGRULEQUERYAGGREGATION_NONE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SecurityMonitoringRuleQueryAggregation) GetAllowedValues() []SecurityMonitoringRuleQueryAggregation {
	return allowedSecurityMonitoringRuleQueryAggregationEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SecurityMonitoringRuleQueryAggregation) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SecurityMonitoringRuleQueryAggregation(value)
	return nil
}

// NewSecurityMonitoringRuleQueryAggregationFromValue returns a pointer to a valid SecurityMonitoringRuleQueryAggregation
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSecurityMonitoringRuleQueryAggregationFromValue(v string) (*SecurityMonitoringRuleQueryAggregation, error) {
	ev := SecurityMonitoringRuleQueryAggregation(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SecurityMonitoringRuleQueryAggregation: valid values are %v", v, allowedSecurityMonitoringRuleQueryAggregationEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SecurityMonitoringRuleQueryAggregation) IsValid() bool {
	for _, existing := range allowedSecurityMonitoringRuleQueryAggregationEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SecurityMonitoringRuleQueryAggregation value.
func (v SecurityMonitoringRuleQueryAggregation) Ptr() *SecurityMonitoringRuleQueryAggregation {
	return &v
}

// NullableSecurityMonitoringRuleQueryAggregation handles when a null is used for SecurityMonitoringRuleQueryAggregation.
type NullableSecurityMonitoringRuleQueryAggregation struct {
	value *SecurityMonitoringRuleQueryAggregation
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleQueryAggregation) Get() *SecurityMonitoringRuleQueryAggregation {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleQueryAggregation) Set(val *SecurityMonitoringRuleQueryAggregation) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleQueryAggregation) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSecurityMonitoringRuleQueryAggregation) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleQueryAggregation initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleQueryAggregation(val *SecurityMonitoringRuleQueryAggregation) *NullableSecurityMonitoringRuleQueryAggregation {
	return &NullableSecurityMonitoringRuleQueryAggregation{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleQueryAggregation) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleQueryAggregation) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
