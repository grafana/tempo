// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ApplicationKeysSort Sorting options
type ApplicationKeysSort string

// List of ApplicationKeysSort.
const (
	APPLICATIONKEYSSORT_CREATED_AT_ASCENDING  ApplicationKeysSort = "created_at"
	APPLICATIONKEYSSORT_CREATED_AT_DESCENDING ApplicationKeysSort = "-created_at"
	APPLICATIONKEYSSORT_LAST4_ASCENDING       ApplicationKeysSort = "last4"
	APPLICATIONKEYSSORT_LAST4_DESCENDING      ApplicationKeysSort = "-last4"
	APPLICATIONKEYSSORT_NAME_ASCENDING        ApplicationKeysSort = "name"
	APPLICATIONKEYSSORT_NAME_DESCENDING       ApplicationKeysSort = "-name"
)

var allowedApplicationKeysSortEnumValues = []ApplicationKeysSort{
	APPLICATIONKEYSSORT_CREATED_AT_ASCENDING,
	APPLICATIONKEYSSORT_CREATED_AT_DESCENDING,
	APPLICATIONKEYSSORT_LAST4_ASCENDING,
	APPLICATIONKEYSSORT_LAST4_DESCENDING,
	APPLICATIONKEYSSORT_NAME_ASCENDING,
	APPLICATIONKEYSSORT_NAME_DESCENDING,
}

// GetAllowedValues reeturns the list of possible values.
func (v *ApplicationKeysSort) GetAllowedValues() []ApplicationKeysSort {
	return allowedApplicationKeysSortEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *ApplicationKeysSort) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = ApplicationKeysSort(value)
	return nil
}

// NewApplicationKeysSortFromValue returns a pointer to a valid ApplicationKeysSort
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewApplicationKeysSortFromValue(v string) (*ApplicationKeysSort, error) {
	ev := ApplicationKeysSort(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for ApplicationKeysSort: valid values are %v", v, allowedApplicationKeysSortEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v ApplicationKeysSort) IsValid() bool {
	for _, existing := range allowedApplicationKeysSortEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to ApplicationKeysSort value.
func (v ApplicationKeysSort) Ptr() *ApplicationKeysSort {
	return &v
}

// NullableApplicationKeysSort handles when a null is used for ApplicationKeysSort.
type NullableApplicationKeysSort struct {
	value *ApplicationKeysSort
	isSet bool
}

// Get returns the associated value.
func (v NullableApplicationKeysSort) Get() *ApplicationKeysSort {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableApplicationKeysSort) Set(val *ApplicationKeysSort) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableApplicationKeysSort) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableApplicationKeysSort) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableApplicationKeysSort initializes the struct as if Set has been called.
func NewNullableApplicationKeysSort(val *ApplicationKeysSort) *NullableApplicationKeysSort {
	return &NullableApplicationKeysSort{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableApplicationKeysSort) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableApplicationKeysSort) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
