// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ConfluentAccountType The JSON:API type for this API. Should always be `confluent-cloud-accounts`.
type ConfluentAccountType string

// List of ConfluentAccountType.
const (
	CONFLUENTACCOUNTTYPE_CONFLUENT_CLOUD_ACCOUNTS ConfluentAccountType = "confluent-cloud-accounts"
)

var allowedConfluentAccountTypeEnumValues = []ConfluentAccountType{
	CONFLUENTACCOUNTTYPE_CONFLUENT_CLOUD_ACCOUNTS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ConfluentAccountType) GetAllowedValues() []ConfluentAccountType {
	return allowedConfluentAccountTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ConfluentAccountType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ConfluentAccountType(value)
	return nil
}

// NewConfluentAccountTypeFromValue returns a pointer to a valid ConfluentAccountType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewConfluentAccountTypeFromValue(v string) (*ConfluentAccountType, error) {
	ev := ConfluentAccountType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ConfluentAccountType: valid values are %v", v, allowedConfluentAccountTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ConfluentAccountType) IsValid() bool {
	for _, existing := range allowedConfluentAccountTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ConfluentAccountType value.
func (v ConfluentAccountType) Ptr() *ConfluentAccountType {
	return &v
}

// NullableConfluentAccountType handles when a null is used for ConfluentAccountType.
type NullableConfluentAccountType struct {
	value *ConfluentAccountType
	isSet bool
}

// Get returns the associated value.
func (v NullableConfluentAccountType) Get() *ConfluentAccountType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableConfluentAccountType) Set(val *ConfluentAccountType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableConfluentAccountType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableConfluentAccountType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableConfluentAccountType initializes the struct as if Set has been called.
func NewNullableConfluentAccountType(val *ConfluentAccountType) *NullableConfluentAccountType {
	return &NullableConfluentAccountType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableConfluentAccountType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableConfluentAccountType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
