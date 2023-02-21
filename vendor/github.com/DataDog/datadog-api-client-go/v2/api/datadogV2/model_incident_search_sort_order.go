// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentSearchSortOrder The ways searched incidents can be sorted.
type IncidentSearchSortOrder string

// List of IncidentSearchSortOrder.
const (
	INCIDENTSEARCHSORTORDER_CREATED_ASCENDING  IncidentSearchSortOrder = "created"
	INCIDENTSEARCHSORTORDER_CREATED_DESCENDING IncidentSearchSortOrder = "-created"
)

var allowedIncidentSearchSortOrderEnumValues = []IncidentSearchSortOrder{
	INCIDENTSEARCHSORTORDER_CREATED_ASCENDING,
	INCIDENTSEARCHSORTORDER_CREATED_DESCENDING,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentSearchSortOrder) GetAllowedValues() []IncidentSearchSortOrder {
	return allowedIncidentSearchSortOrderEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentSearchSortOrder) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentSearchSortOrder(value)
	return nil
}

// NewIncidentSearchSortOrderFromValue returns a pointer to a valid IncidentSearchSortOrder
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentSearchSortOrderFromValue(v string) (*IncidentSearchSortOrder, error) {
	ev := IncidentSearchSortOrder(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentSearchSortOrder: valid values are %v", v, allowedIncidentSearchSortOrderEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentSearchSortOrder) IsValid() bool {
	for _, existing := range allowedIncidentSearchSortOrderEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentSearchSortOrder value.
func (v IncidentSearchSortOrder) Ptr() *IncidentSearchSortOrder {
	return &v
}

// NullableIncidentSearchSortOrder handles when a null is used for IncidentSearchSortOrder.
type NullableIncidentSearchSortOrder struct {
	value *IncidentSearchSortOrder
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentSearchSortOrder) Get() *IncidentSearchSortOrder {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentSearchSortOrder) Set(val *IncidentSearchSortOrder) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentSearchSortOrder) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentSearchSortOrder) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentSearchSortOrder initializes the struct as if Set has been called.
func NewNullableIncidentSearchSortOrder(val *IncidentSearchSortOrder) *NullableIncidentSearchSortOrder {
	return &NullableIncidentSearchSortOrder{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentSearchSortOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentSearchSortOrder) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
