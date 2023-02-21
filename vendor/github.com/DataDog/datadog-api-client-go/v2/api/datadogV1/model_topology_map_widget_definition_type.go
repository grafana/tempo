// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// TopologyMapWidgetDefinitionType Type of the topology map widget.
type TopologyMapWidgetDefinitionType string

// List of TopologyMapWidgetDefinitionType.
const (
	TOPOLOGYMAPWIDGETDEFINITIONTYPE_TOPOLOGY_MAP TopologyMapWidgetDefinitionType = "topology_map"
)

var allowedTopologyMapWidgetDefinitionTypeEnumValues = []TopologyMapWidgetDefinitionType{
	TOPOLOGYMAPWIDGETDEFINITIONTYPE_TOPOLOGY_MAP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TopologyMapWidgetDefinitionType) GetAllowedValues() []TopologyMapWidgetDefinitionType {
	return allowedTopologyMapWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TopologyMapWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TopologyMapWidgetDefinitionType(value)
	return nil
}

// NewTopologyMapWidgetDefinitionTypeFromValue returns a pointer to a valid TopologyMapWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTopologyMapWidgetDefinitionTypeFromValue(v string) (*TopologyMapWidgetDefinitionType, error) {
	ev := TopologyMapWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TopologyMapWidgetDefinitionType: valid values are %v", v, allowedTopologyMapWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TopologyMapWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedTopologyMapWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TopologyMapWidgetDefinitionType value.
func (v TopologyMapWidgetDefinitionType) Ptr() *TopologyMapWidgetDefinitionType {
	return &v
}

// NullableTopologyMapWidgetDefinitionType handles when a null is used for TopologyMapWidgetDefinitionType.
type NullableTopologyMapWidgetDefinitionType struct {
	value *TopologyMapWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableTopologyMapWidgetDefinitionType) Get() *TopologyMapWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTopologyMapWidgetDefinitionType) Set(val *TopologyMapWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTopologyMapWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTopologyMapWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTopologyMapWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableTopologyMapWidgetDefinitionType(val *TopologyMapWidgetDefinitionType) *NullableTopologyMapWidgetDefinitionType {
	return &NullableTopologyMapWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTopologyMapWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTopologyMapWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
