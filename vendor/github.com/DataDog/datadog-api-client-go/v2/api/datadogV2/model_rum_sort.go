// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMSort Sort parameters when querying events.
type RUMSort string

// List of RUMSort.
const (
	RUMSORT_TIMESTAMP_ASCENDING  RUMSort = "timestamp"
	RUMSORT_TIMESTAMP_DESCENDING RUMSort = "-timestamp"
)

var allowedRUMSortEnumValues = []RUMSort{
	RUMSORT_TIMESTAMP_ASCENDING,
	RUMSORT_TIMESTAMP_DESCENDING,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RUMSort) GetAllowedValues() []RUMSort {
	return allowedRUMSortEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RUMSort) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RUMSort(value)
	return nil
}

// NewRUMSortFromValue returns a pointer to a valid RUMSort
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRUMSortFromValue(v string) (*RUMSort, error) {
	ev := RUMSort(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RUMSort: valid values are %v", v, allowedRUMSortEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RUMSort) IsValid() bool {
	for _, existing := range allowedRUMSortEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RUMSort value.
func (v RUMSort) Ptr() *RUMSort {
	return &v
}

// NullableRUMSort handles when a null is used for RUMSort.
type NullableRUMSort struct {
	value *RUMSort
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMSort) Get() *RUMSort {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMSort) Set(val *RUMSort) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMSort) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRUMSort) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMSort initializes the struct as if Set has been called.
func NewNullableRUMSort(val *RUMSort) *NullableRUMSort {
	return &NullableRUMSort{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMSort) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMSort) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
