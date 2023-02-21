// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudConfigurationRuleType The rule type.
type CloudConfigurationRuleType string

// List of CloudConfigurationRuleType.
const (
	CLOUDCONFIGURATIONRULETYPE_CLOUD_CONFIGURATION CloudConfigurationRuleType = "cloud_configuration"
)

var allowedCloudConfigurationRuleTypeEnumValues = []CloudConfigurationRuleType{
	CLOUDCONFIGURATIONRULETYPE_CLOUD_CONFIGURATION,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CloudConfigurationRuleType) GetAllowedValues() []CloudConfigurationRuleType {
	return allowedCloudConfigurationRuleTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CloudConfigurationRuleType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CloudConfigurationRuleType(value)
	return nil
}

// NewCloudConfigurationRuleTypeFromValue returns a pointer to a valid CloudConfigurationRuleType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCloudConfigurationRuleTypeFromValue(v string) (*CloudConfigurationRuleType, error) {
	ev := CloudConfigurationRuleType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CloudConfigurationRuleType: valid values are %v", v, allowedCloudConfigurationRuleTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CloudConfigurationRuleType) IsValid() bool {
	for _, existing := range allowedCloudConfigurationRuleTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CloudConfigurationRuleType value.
func (v CloudConfigurationRuleType) Ptr() *CloudConfigurationRuleType {
	return &v
}

// NullableCloudConfigurationRuleType handles when a null is used for CloudConfigurationRuleType.
type NullableCloudConfigurationRuleType struct {
	value *CloudConfigurationRuleType
	isSet bool
}

// Get returns the associated value.
func (v NullableCloudConfigurationRuleType) Get() *CloudConfigurationRuleType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCloudConfigurationRuleType) Set(val *CloudConfigurationRuleType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCloudConfigurationRuleType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCloudConfigurationRuleType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCloudConfigurationRuleType initializes the struct as if Set has been called.
func NewNullableCloudConfigurationRuleType(val *CloudConfigurationRuleType) *NullableCloudConfigurationRuleType {
	return &NullableCloudConfigurationRuleType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCloudConfigurationRuleType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCloudConfigurationRuleType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
