// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// TopologyRequestType Widget request type.
type TopologyRequestType string

// List of TopologyRequestType.
const (
	TOPOLOGYREQUESTTYPE_TOPOLOGY TopologyRequestType = "topology"
)

var allowedTopologyRequestTypeEnumValues = []TopologyRequestType{
	TOPOLOGYREQUESTTYPE_TOPOLOGY,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TopologyRequestType) GetAllowedValues() []TopologyRequestType {
	return allowedTopologyRequestTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TopologyRequestType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TopologyRequestType(value)
	return nil
}

// NewTopologyRequestTypeFromValue returns a pointer to a valid TopologyRequestType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTopologyRequestTypeFromValue(v string) (*TopologyRequestType, error) {
	ev := TopologyRequestType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TopologyRequestType: valid values are %v", v, allowedTopologyRequestTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TopologyRequestType) IsValid() bool {
	for _, existing := range allowedTopologyRequestTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TopologyRequestType value.
func (v TopologyRequestType) Ptr() *TopologyRequestType {
	return &v
}

// NullableTopologyRequestType handles when a null is used for TopologyRequestType.
type NullableTopologyRequestType struct {
	value *TopologyRequestType
	isSet bool
}

// Get returns the associated value.
func (v NullableTopologyRequestType) Get() *TopologyRequestType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTopologyRequestType) Set(val *TopologyRequestType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTopologyRequestType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTopologyRequestType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTopologyRequestType initializes the struct as if Set has been called.
func NewNullableTopologyRequestType(val *TopologyRequestType) *NullableTopologyRequestType {
	return &NullableTopologyRequestType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTopologyRequestType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTopologyRequestType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
