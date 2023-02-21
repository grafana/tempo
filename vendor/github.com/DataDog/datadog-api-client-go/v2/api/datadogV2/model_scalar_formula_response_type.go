// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ScalarFormulaResponseType The type of the resource. The value should always be scalar_response.
type ScalarFormulaResponseType string

// List of ScalarFormulaResponseType.
const (
	SCALARFORMULARESPONSETYPE_SCALAR_RESPONSE ScalarFormulaResponseType = "scalar_response"
)

var allowedScalarFormulaResponseTypeEnumValues = []ScalarFormulaResponseType{
	SCALARFORMULARESPONSETYPE_SCALAR_RESPONSE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ScalarFormulaResponseType) GetAllowedValues() []ScalarFormulaResponseType {
	return allowedScalarFormulaResponseTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ScalarFormulaResponseType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ScalarFormulaResponseType(value)
	return nil
}

// NewScalarFormulaResponseTypeFromValue returns a pointer to a valid ScalarFormulaResponseType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewScalarFormulaResponseTypeFromValue(v string) (*ScalarFormulaResponseType, error) {
	ev := ScalarFormulaResponseType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ScalarFormulaResponseType: valid values are %v", v, allowedScalarFormulaResponseTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ScalarFormulaResponseType) IsValid() bool {
	for _, existing := range allowedScalarFormulaResponseTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ScalarFormulaResponseType value.
func (v ScalarFormulaResponseType) Ptr() *ScalarFormulaResponseType {
	return &v
}

// NullableScalarFormulaResponseType handles when a null is used for ScalarFormulaResponseType.
type NullableScalarFormulaResponseType struct {
	value *ScalarFormulaResponseType
	isSet bool
}

// Get returns the associated value.
func (v NullableScalarFormulaResponseType) Get() *ScalarFormulaResponseType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableScalarFormulaResponseType) Set(val *ScalarFormulaResponseType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableScalarFormulaResponseType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableScalarFormulaResponseType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableScalarFormulaResponseType initializes the struct as if Set has been called.
func NewNullableScalarFormulaResponseType(val *ScalarFormulaResponseType) *NullableScalarFormulaResponseType {
	return &NullableScalarFormulaResponseType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableScalarFormulaResponseType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableScalarFormulaResponseType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
