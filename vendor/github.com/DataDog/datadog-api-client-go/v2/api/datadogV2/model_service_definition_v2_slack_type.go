// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV2SlackType Contact type.
type ServiceDefinitionV2SlackType string

// List of ServiceDefinitionV2SlackType.
const (
	SERVICEDEFINITIONV2SLACKTYPE_SLACK ServiceDefinitionV2SlackType = "slack"
)

var allowedServiceDefinitionV2SlackTypeEnumValues = []ServiceDefinitionV2SlackType{
	SERVICEDEFINITIONV2SLACKTYPE_SLACK,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ServiceDefinitionV2SlackType) GetAllowedValues() []ServiceDefinitionV2SlackType {
	return allowedServiceDefinitionV2SlackTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ServiceDefinitionV2SlackType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ServiceDefinitionV2SlackType(value)
	return nil
}

// NewServiceDefinitionV2SlackTypeFromValue returns a pointer to a valid ServiceDefinitionV2SlackType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewServiceDefinitionV2SlackTypeFromValue(v string) (*ServiceDefinitionV2SlackType, error) {
	ev := ServiceDefinitionV2SlackType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ServiceDefinitionV2SlackType: valid values are %v", v, allowedServiceDefinitionV2SlackTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ServiceDefinitionV2SlackType) IsValid() bool {
	for _, existing := range allowedServiceDefinitionV2SlackTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ServiceDefinitionV2SlackType value.
func (v ServiceDefinitionV2SlackType) Ptr() *ServiceDefinitionV2SlackType {
	return &v
}

// NullableServiceDefinitionV2SlackType handles when a null is used for ServiceDefinitionV2SlackType.
type NullableServiceDefinitionV2SlackType struct {
	value *ServiceDefinitionV2SlackType
	isSet bool
}

// Get returns the associated value.
func (v NullableServiceDefinitionV2SlackType) Get() *ServiceDefinitionV2SlackType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableServiceDefinitionV2SlackType) Set(val *ServiceDefinitionV2SlackType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableServiceDefinitionV2SlackType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableServiceDefinitionV2SlackType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableServiceDefinitionV2SlackType initializes the struct as if Set has been called.
func NewNullableServiceDefinitionV2SlackType(val *ServiceDefinitionV2SlackType) *NullableServiceDefinitionV2SlackType {
	return &NullableServiceDefinitionV2SlackType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableServiceDefinitionV2SlackType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableServiceDefinitionV2SlackType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
