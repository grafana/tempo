// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV2LinkType Link type.
type ServiceDefinitionV2LinkType string

// List of ServiceDefinitionV2LinkType.
const (
	SERVICEDEFINITIONV2LINKTYPE_DOC       ServiceDefinitionV2LinkType = "doc"
	SERVICEDEFINITIONV2LINKTYPE_WIKI      ServiceDefinitionV2LinkType = "wiki"
	SERVICEDEFINITIONV2LINKTYPE_RUNBOOK   ServiceDefinitionV2LinkType = "runbook"
	SERVICEDEFINITIONV2LINKTYPE_URL       ServiceDefinitionV2LinkType = "url"
	SERVICEDEFINITIONV2LINKTYPE_REPO      ServiceDefinitionV2LinkType = "repo"
	SERVICEDEFINITIONV2LINKTYPE_DASHBOARD ServiceDefinitionV2LinkType = "dashboard"
	SERVICEDEFINITIONV2LINKTYPE_ONCALL    ServiceDefinitionV2LinkType = "oncall"
	SERVICEDEFINITIONV2LINKTYPE_CODE      ServiceDefinitionV2LinkType = "code"
	SERVICEDEFINITIONV2LINKTYPE_LINK      ServiceDefinitionV2LinkType = "link"
)

var allowedServiceDefinitionV2LinkTypeEnumValues = []ServiceDefinitionV2LinkType{
	SERVICEDEFINITIONV2LINKTYPE_DOC,
	SERVICEDEFINITIONV2LINKTYPE_WIKI,
	SERVICEDEFINITIONV2LINKTYPE_RUNBOOK,
	SERVICEDEFINITIONV2LINKTYPE_URL,
	SERVICEDEFINITIONV2LINKTYPE_REPO,
	SERVICEDEFINITIONV2LINKTYPE_DASHBOARD,
	SERVICEDEFINITIONV2LINKTYPE_ONCALL,
	SERVICEDEFINITIONV2LINKTYPE_CODE,
	SERVICEDEFINITIONV2LINKTYPE_LINK,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ServiceDefinitionV2LinkType) GetAllowedValues() []ServiceDefinitionV2LinkType {
	return allowedServiceDefinitionV2LinkTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ServiceDefinitionV2LinkType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ServiceDefinitionV2LinkType(value)
	return nil
}

// NewServiceDefinitionV2LinkTypeFromValue returns a pointer to a valid ServiceDefinitionV2LinkType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewServiceDefinitionV2LinkTypeFromValue(v string) (*ServiceDefinitionV2LinkType, error) {
	ev := ServiceDefinitionV2LinkType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ServiceDefinitionV2LinkType: valid values are %v", v, allowedServiceDefinitionV2LinkTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ServiceDefinitionV2LinkType) IsValid() bool {
	for _, existing := range allowedServiceDefinitionV2LinkTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ServiceDefinitionV2LinkType value.
func (v ServiceDefinitionV2LinkType) Ptr() *ServiceDefinitionV2LinkType {
	return &v
}

// NullableServiceDefinitionV2LinkType handles when a null is used for ServiceDefinitionV2LinkType.
type NullableServiceDefinitionV2LinkType struct {
	value *ServiceDefinitionV2LinkType
	isSet bool
}

// Get returns the associated value.
func (v NullableServiceDefinitionV2LinkType) Get() *ServiceDefinitionV2LinkType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableServiceDefinitionV2LinkType) Set(val *ServiceDefinitionV2LinkType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableServiceDefinitionV2LinkType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableServiceDefinitionV2LinkType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableServiceDefinitionV2LinkType initializes the struct as if Set has been called.
func NewNullableServiceDefinitionV2LinkType(val *ServiceDefinitionV2LinkType) *NullableServiceDefinitionV2LinkType {
	return &NullableServiceDefinitionV2LinkType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableServiceDefinitionV2LinkType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableServiceDefinitionV2LinkType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
