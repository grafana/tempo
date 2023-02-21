// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV2EmailType Contact type.
type ServiceDefinitionV2EmailType string

// List of ServiceDefinitionV2EmailType.
const (
	SERVICEDEFINITIONV2EMAILTYPE_EMAIL ServiceDefinitionV2EmailType = "email"
)

var allowedServiceDefinitionV2EmailTypeEnumValues = []ServiceDefinitionV2EmailType{
	SERVICEDEFINITIONV2EMAILTYPE_EMAIL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ServiceDefinitionV2EmailType) GetAllowedValues() []ServiceDefinitionV2EmailType {
	return allowedServiceDefinitionV2EmailTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ServiceDefinitionV2EmailType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ServiceDefinitionV2EmailType(value)
	return nil
}

// NewServiceDefinitionV2EmailTypeFromValue returns a pointer to a valid ServiceDefinitionV2EmailType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewServiceDefinitionV2EmailTypeFromValue(v string) (*ServiceDefinitionV2EmailType, error) {
	ev := ServiceDefinitionV2EmailType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ServiceDefinitionV2EmailType: valid values are %v", v, allowedServiceDefinitionV2EmailTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ServiceDefinitionV2EmailType) IsValid() bool {
	for _, existing := range allowedServiceDefinitionV2EmailTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ServiceDefinitionV2EmailType value.
func (v ServiceDefinitionV2EmailType) Ptr() *ServiceDefinitionV2EmailType {
	return &v
}

// NullableServiceDefinitionV2EmailType handles when a null is used for ServiceDefinitionV2EmailType.
type NullableServiceDefinitionV2EmailType struct {
	value *ServiceDefinitionV2EmailType
	isSet bool
}

// Get returns the associated value.
func (v NullableServiceDefinitionV2EmailType) Get() *ServiceDefinitionV2EmailType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableServiceDefinitionV2EmailType) Set(val *ServiceDefinitionV2EmailType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableServiceDefinitionV2EmailType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableServiceDefinitionV2EmailType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableServiceDefinitionV2EmailType initializes the struct as if Set has been called.
func NewNullableServiceDefinitionV2EmailType(val *ServiceDefinitionV2EmailType) *NullableServiceDefinitionV2EmailType {
	return &NullableServiceDefinitionV2EmailType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableServiceDefinitionV2EmailType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableServiceDefinitionV2EmailType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
