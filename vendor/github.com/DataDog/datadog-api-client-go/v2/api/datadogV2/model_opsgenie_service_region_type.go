// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// OpsgenieServiceRegionType The region for the Opsgenie service.
type OpsgenieServiceRegionType string

// List of OpsgenieServiceRegionType.
const (
	OPSGENIESERVICEREGIONTYPE_US     OpsgenieServiceRegionType = "us"
	OPSGENIESERVICEREGIONTYPE_EU     OpsgenieServiceRegionType = "eu"
	OPSGENIESERVICEREGIONTYPE_CUSTOM OpsgenieServiceRegionType = "custom"
)

var allowedOpsgenieServiceRegionTypeEnumValues = []OpsgenieServiceRegionType{
	OPSGENIESERVICEREGIONTYPE_US,
	OPSGENIESERVICEREGIONTYPE_EU,
	OPSGENIESERVICEREGIONTYPE_CUSTOM,
}

// GetAllowedValues reeturns the list of possible values.
func (v *OpsgenieServiceRegionType) GetAllowedValues() []OpsgenieServiceRegionType {
	return allowedOpsgenieServiceRegionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *OpsgenieServiceRegionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = OpsgenieServiceRegionType(value)
	return nil
}

// NewOpsgenieServiceRegionTypeFromValue returns a pointer to a valid OpsgenieServiceRegionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewOpsgenieServiceRegionTypeFromValue(v string) (*OpsgenieServiceRegionType, error) {
	ev := OpsgenieServiceRegionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for OpsgenieServiceRegionType: valid values are %v", v, allowedOpsgenieServiceRegionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v OpsgenieServiceRegionType) IsValid() bool {
	for _, existing := range allowedOpsgenieServiceRegionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to OpsgenieServiceRegionType value.
func (v OpsgenieServiceRegionType) Ptr() *OpsgenieServiceRegionType {
	return &v
}

// NullableOpsgenieServiceRegionType handles when a null is used for OpsgenieServiceRegionType.
type NullableOpsgenieServiceRegionType struct {
	value *OpsgenieServiceRegionType
	isSet bool
}

// Get returns the associated value.
func (v NullableOpsgenieServiceRegionType) Get() *OpsgenieServiceRegionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableOpsgenieServiceRegionType) Set(val *OpsgenieServiceRegionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableOpsgenieServiceRegionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableOpsgenieServiceRegionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableOpsgenieServiceRegionType initializes the struct as if Set has been called.
func NewNullableOpsgenieServiceRegionType(val *OpsgenieServiceRegionType) *NullableOpsgenieServiceRegionType {
	return &NullableOpsgenieServiceRegionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableOpsgenieServiceRegionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableOpsgenieServiceRegionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
