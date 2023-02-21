// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CostByOrgType Type of cost data.
type CostByOrgType string

// List of CostByOrgType.
const (
	COSTBYORGTYPE_COST_BY_ORG CostByOrgType = "cost_by_org"
)

var allowedCostByOrgTypeEnumValues = []CostByOrgType{
	COSTBYORGTYPE_COST_BY_ORG,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CostByOrgType) GetAllowedValues() []CostByOrgType {
	return allowedCostByOrgTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CostByOrgType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CostByOrgType(value)
	return nil
}

// NewCostByOrgTypeFromValue returns a pointer to a valid CostByOrgType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCostByOrgTypeFromValue(v string) (*CostByOrgType, error) {
	ev := CostByOrgType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CostByOrgType: valid values are %v", v, allowedCostByOrgTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CostByOrgType) IsValid() bool {
	for _, existing := range allowedCostByOrgTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CostByOrgType value.
func (v CostByOrgType) Ptr() *CostByOrgType {
	return &v
}

// NullableCostByOrgType handles when a null is used for CostByOrgType.
type NullableCostByOrgType struct {
	value *CostByOrgType
	isSet bool
}

// Get returns the associated value.
func (v NullableCostByOrgType) Get() *CostByOrgType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCostByOrgType) Set(val *CostByOrgType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCostByOrgType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCostByOrgType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCostByOrgType initializes the struct as if Set has been called.
func NewNullableCostByOrgType(val *CostByOrgType) *NullableCostByOrgType {
	return &NullableCostByOrgType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCostByOrgType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCostByOrgType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
