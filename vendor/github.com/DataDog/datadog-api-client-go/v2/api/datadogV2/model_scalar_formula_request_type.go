// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ScalarFormulaRequestType The type of the resource. The value should always be scalar_request.
type ScalarFormulaRequestType string

// List of ScalarFormulaRequestType.
const (
	SCALARFORMULAREQUESTTYPE_SCALAR_REQUEST ScalarFormulaRequestType = "scalar_request"
)

var allowedScalarFormulaRequestTypeEnumValues = []ScalarFormulaRequestType{
	SCALARFORMULAREQUESTTYPE_SCALAR_REQUEST,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ScalarFormulaRequestType) GetAllowedValues() []ScalarFormulaRequestType {
	return allowedScalarFormulaRequestTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ScalarFormulaRequestType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ScalarFormulaRequestType(value)
	return nil
}

// NewScalarFormulaRequestTypeFromValue returns a pointer to a valid ScalarFormulaRequestType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewScalarFormulaRequestTypeFromValue(v string) (*ScalarFormulaRequestType, error) {
	ev := ScalarFormulaRequestType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ScalarFormulaRequestType: valid values are %v", v, allowedScalarFormulaRequestTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ScalarFormulaRequestType) IsValid() bool {
	for _, existing := range allowedScalarFormulaRequestTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ScalarFormulaRequestType value.
func (v ScalarFormulaRequestType) Ptr() *ScalarFormulaRequestType {
	return &v
}

// NullableScalarFormulaRequestType handles when a null is used for ScalarFormulaRequestType.
type NullableScalarFormulaRequestType struct {
	value *ScalarFormulaRequestType
	isSet bool
}

// Get returns the associated value.
func (v NullableScalarFormulaRequestType) Get() *ScalarFormulaRequestType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableScalarFormulaRequestType) Set(val *ScalarFormulaRequestType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableScalarFormulaRequestType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableScalarFormulaRequestType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableScalarFormulaRequestType initializes the struct as if Set has been called.
func NewNullableScalarFormulaRequestType(val *ScalarFormulaRequestType) *NullableScalarFormulaRequestType {
	return &NullableScalarFormulaRequestType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableScalarFormulaRequestType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableScalarFormulaRequestType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
