// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// FastlyServiceType The JSON:API type for this API. Should always be `fastly-services`.
type FastlyServiceType string

// List of FastlyServiceType.
const (
	FASTLYSERVICETYPE_FASTLY_SERVICES FastlyServiceType = "fastly-services"
)

var allowedFastlyServiceTypeEnumValues = []FastlyServiceType{
	FASTLYSERVICETYPE_FASTLY_SERVICES,
}

// GetAllowedValues reeturns the list of possible values.
func (v *FastlyServiceType) GetAllowedValues() []FastlyServiceType {
	return allowedFastlyServiceTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *FastlyServiceType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = FastlyServiceType(value)
	return nil
}

// NewFastlyServiceTypeFromValue returns a pointer to a valid FastlyServiceType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewFastlyServiceTypeFromValue(v string) (*FastlyServiceType, error) {
	ev := FastlyServiceType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for FastlyServiceType: valid values are %v", v, allowedFastlyServiceTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v FastlyServiceType) IsValid() bool {
	for _, existing := range allowedFastlyServiceTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to FastlyServiceType value.
func (v FastlyServiceType) Ptr() *FastlyServiceType {
	return &v
}

// NullableFastlyServiceType handles when a null is used for FastlyServiceType.
type NullableFastlyServiceType struct {
	value *FastlyServiceType
	isSet bool
}

// Get returns the associated value.
func (v NullableFastlyServiceType) Get() *FastlyServiceType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableFastlyServiceType) Set(val *FastlyServiceType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableFastlyServiceType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableFastlyServiceType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableFastlyServiceType initializes the struct as if Set has been called.
func NewNullableFastlyServiceType(val *FastlyServiceType) *NullableFastlyServiceType {
	return &NullableFastlyServiceType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableFastlyServiceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableFastlyServiceType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
