// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// TopologyQueryDataSource Name of the data source
type TopologyQueryDataSource string

// List of TopologyQueryDataSource.
const (
	TOPOLOGYQUERYDATASOURCE_DATA_STREAMS TopologyQueryDataSource = "data_streams"
	TOPOLOGYQUERYDATASOURCE_SERVICE_MAP  TopologyQueryDataSource = "service_map"
)

var allowedTopologyQueryDataSourceEnumValues = []TopologyQueryDataSource{
	TOPOLOGYQUERYDATASOURCE_DATA_STREAMS,
	TOPOLOGYQUERYDATASOURCE_SERVICE_MAP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *TopologyQueryDataSource) GetAllowedValues() []TopologyQueryDataSource {
	return allowedTopologyQueryDataSourceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *TopologyQueryDataSource) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = TopologyQueryDataSource(value)
	return nil
}

// NewTopologyQueryDataSourceFromValue returns a pointer to a valid TopologyQueryDataSource
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewTopologyQueryDataSourceFromValue(v string) (*TopologyQueryDataSource, error) {
	ev := TopologyQueryDataSource(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for TopologyQueryDataSource: valid values are %v", v, allowedTopologyQueryDataSourceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v TopologyQueryDataSource) IsValid() bool {
	for _, existing := range allowedTopologyQueryDataSourceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to TopologyQueryDataSource value.
func (v TopologyQueryDataSource) Ptr() *TopologyQueryDataSource {
	return &v
}

// NullableTopologyQueryDataSource handles when a null is used for TopologyQueryDataSource.
type NullableTopologyQueryDataSource struct {
	value *TopologyQueryDataSource
	isSet bool
}

// Get returns the associated value.
func (v NullableTopologyQueryDataSource) Get() *TopologyQueryDataSource {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTopologyQueryDataSource) Set(val *TopologyQueryDataSource) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTopologyQueryDataSource) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableTopologyQueryDataSource) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTopologyQueryDataSource initializes the struct as if Set has been called.
func NewNullableTopologyQueryDataSource(val *TopologyQueryDataSource) *NullableTopologyQueryDataSource {
	return &NullableTopologyQueryDataSource{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTopologyQueryDataSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTopologyQueryDataSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
