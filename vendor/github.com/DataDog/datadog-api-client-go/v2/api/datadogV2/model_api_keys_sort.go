// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// APIKeysSort Sorting options
type APIKeysSort string

// List of APIKeysSort.
const (
	APIKEYSSORT_CREATED_AT_ASCENDING   APIKeysSort = "created_at"
	APIKEYSSORT_CREATED_AT_DESCENDING  APIKeysSort = "-created_at"
	APIKEYSSORT_LAST4_ASCENDING        APIKeysSort = "last4"
	APIKEYSSORT_LAST4_DESCENDING       APIKeysSort = "-last4"
	APIKEYSSORT_MODIFIED_AT_ASCENDING  APIKeysSort = "modified_at"
	APIKEYSSORT_MODIFIED_AT_DESCENDING APIKeysSort = "-modified_at"
	APIKEYSSORT_NAME_ASCENDING         APIKeysSort = "name"
	APIKEYSSORT_NAME_DESCENDING        APIKeysSort = "-name"
)

var allowedAPIKeysSortEnumValues = []APIKeysSort{
	APIKEYSSORT_CREATED_AT_ASCENDING,
	APIKEYSSORT_CREATED_AT_DESCENDING,
	APIKEYSSORT_LAST4_ASCENDING,
	APIKEYSSORT_LAST4_DESCENDING,
	APIKEYSSORT_MODIFIED_AT_ASCENDING,
	APIKEYSSORT_MODIFIED_AT_DESCENDING,
	APIKEYSSORT_NAME_ASCENDING,
	APIKEYSSORT_NAME_DESCENDING,
}

// GetAllowedValues reeturns the list of possible values.
func (v *APIKeysSort) GetAllowedValues() []APIKeysSort {
	return allowedAPIKeysSortEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *APIKeysSort) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = APIKeysSort(value)
	return nil
}

// NewAPIKeysSortFromValue returns a pointer to a valid APIKeysSort
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewAPIKeysSortFromValue(v string) (*APIKeysSort, error) {
	ev := APIKeysSort(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for APIKeysSort: valid values are %v", v, allowedAPIKeysSortEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v APIKeysSort) IsValid() bool {
	for _, existing := range allowedAPIKeysSortEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to APIKeysSort value.
func (v APIKeysSort) Ptr() *APIKeysSort {
	return &v
}

// NullableAPIKeysSort handles when a null is used for APIKeysSort.
type NullableAPIKeysSort struct {
	value *APIKeysSort
	isSet bool
}

// Get returns the associated value.
func (v NullableAPIKeysSort) Get() *APIKeysSort {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAPIKeysSort) Set(val *APIKeysSort) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAPIKeysSort) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableAPIKeysSort) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAPIKeysSort initializes the struct as if Set has been called.
func NewNullableAPIKeysSort(val *APIKeysSort) *NullableAPIKeysSort {
	return &NullableAPIKeysSort{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAPIKeysSort) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAPIKeysSort) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
