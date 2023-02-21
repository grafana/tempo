// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RolesSort Sorting options for roles.
type RolesSort string

// List of RolesSort.
const (
	ROLESSORT_NAME_ASCENDING         RolesSort = "name"
	ROLESSORT_NAME_DESCENDING        RolesSort = "-name"
	ROLESSORT_MODIFIED_AT_ASCENDING  RolesSort = "modified_at"
	ROLESSORT_MODIFIED_AT_DESCENDING RolesSort = "-modified_at"
	ROLESSORT_USER_COUNT_ASCENDING   RolesSort = "user_count"
	ROLESSORT_USER_COUNT_DESCENDING  RolesSort = "-user_count"
)

var allowedRolesSortEnumValues = []RolesSort{
	ROLESSORT_NAME_ASCENDING,
	ROLESSORT_NAME_DESCENDING,
	ROLESSORT_MODIFIED_AT_ASCENDING,
	ROLESSORT_MODIFIED_AT_DESCENDING,
	ROLESSORT_USER_COUNT_ASCENDING,
	ROLESSORT_USER_COUNT_DESCENDING,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RolesSort) GetAllowedValues() []RolesSort {
	return allowedRolesSortEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RolesSort) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RolesSort(value)
	return nil
}

// NewRolesSortFromValue returns a pointer to a valid RolesSort
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRolesSortFromValue(v string) (*RolesSort, error) {
	ev := RolesSort(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RolesSort: valid values are %v", v, allowedRolesSortEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RolesSort) IsValid() bool {
	for _, existing := range allowedRolesSortEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RolesSort value.
func (v RolesSort) Ptr() *RolesSort {
	return &v
}

// NullableRolesSort handles when a null is used for RolesSort.
type NullableRolesSort struct {
	value *RolesSort
	isSet bool
}

// Get returns the associated value.
func (v NullableRolesSort) Get() *RolesSort {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRolesSort) Set(val *RolesSort) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRolesSort) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRolesSort) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRolesSort initializes the struct as if Set has been called.
func NewNullableRolesSort(val *RolesSort) *NullableRolesSort {
	return &NullableRolesSort{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRolesSort) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRolesSort) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
