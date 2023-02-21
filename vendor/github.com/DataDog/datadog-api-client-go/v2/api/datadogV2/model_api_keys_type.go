// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// APIKeysType API Keys resource type.
type APIKeysType string

// List of APIKeysType.
const (
	APIKEYSTYPE_API_KEYS APIKeysType = "api_keys"
)

var allowedAPIKeysTypeEnumValues = []APIKeysType{
	APIKEYSTYPE_API_KEYS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *APIKeysType) GetAllowedValues() []APIKeysType {
	return allowedAPIKeysTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *APIKeysType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = APIKeysType(value)
	return nil
}

// NewAPIKeysTypeFromValue returns a pointer to a valid APIKeysType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewAPIKeysTypeFromValue(v string) (*APIKeysType, error) {
	ev := APIKeysType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for APIKeysType: valid values are %v", v, allowedAPIKeysTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v APIKeysType) IsValid() bool {
	for _, existing := range allowedAPIKeysTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to APIKeysType value.
func (v APIKeysType) Ptr() *APIKeysType {
	return &v
}

// NullableAPIKeysType handles when a null is used for APIKeysType.
type NullableAPIKeysType struct {
	value *APIKeysType
	isSet bool
}

// Get returns the associated value.
func (v NullableAPIKeysType) Get() *APIKeysType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAPIKeysType) Set(val *APIKeysType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAPIKeysType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableAPIKeysType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAPIKeysType initializes the struct as if Set has been called.
func NewNullableAPIKeysType(val *APIKeysType) *NullableAPIKeysType {
	return &NullableAPIKeysType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAPIKeysType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAPIKeysType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
