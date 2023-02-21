// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// OpsgenieServiceType Opsgenie service resource type.
type OpsgenieServiceType string

// List of OpsgenieServiceType.
const (
	OPSGENIESERVICETYPE_OPSGENIE_SERVICE OpsgenieServiceType = "opsgenie-service"
)

var allowedOpsgenieServiceTypeEnumValues = []OpsgenieServiceType{
	OPSGENIESERVICETYPE_OPSGENIE_SERVICE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *OpsgenieServiceType) GetAllowedValues() []OpsgenieServiceType {
	return allowedOpsgenieServiceTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *OpsgenieServiceType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = OpsgenieServiceType(value)
	return nil
}

// NewOpsgenieServiceTypeFromValue returns a pointer to a valid OpsgenieServiceType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewOpsgenieServiceTypeFromValue(v string) (*OpsgenieServiceType, error) {
	ev := OpsgenieServiceType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for OpsgenieServiceType: valid values are %v", v, allowedOpsgenieServiceTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v OpsgenieServiceType) IsValid() bool {
	for _, existing := range allowedOpsgenieServiceTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to OpsgenieServiceType value.
func (v OpsgenieServiceType) Ptr() *OpsgenieServiceType {
	return &v
}

// NullableOpsgenieServiceType handles when a null is used for OpsgenieServiceType.
type NullableOpsgenieServiceType struct {
	value *OpsgenieServiceType
	isSet bool
}

// Get returns the associated value.
func (v NullableOpsgenieServiceType) Get() *OpsgenieServiceType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableOpsgenieServiceType) Set(val *OpsgenieServiceType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableOpsgenieServiceType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableOpsgenieServiceType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableOpsgenieServiceType initializes the struct as if Set has been called.
func NewNullableOpsgenieServiceType(val *OpsgenieServiceType) *NullableOpsgenieServiceType {
	return &NullableOpsgenieServiceType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableOpsgenieServiceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableOpsgenieServiceType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
