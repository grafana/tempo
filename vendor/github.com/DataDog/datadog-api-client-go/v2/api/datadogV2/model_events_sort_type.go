// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// EventsSortType The type of sort to use on the calculated value.
type EventsSortType string

// List of EventsSortType.
const (
	EVENTSSORTTYPE_ALPHABETICAL EventsSortType = "alphabetical"
	EVENTSSORTTYPE_MEASURE      EventsSortType = "measure"
)

var allowedEventsSortTypeEnumValues = []EventsSortType{
	EVENTSSORTTYPE_ALPHABETICAL,
	EVENTSSORTTYPE_MEASURE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *EventsSortType) GetAllowedValues() []EventsSortType {
	return allowedEventsSortTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *EventsSortType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = EventsSortType(value)
	return nil
}

// NewEventsSortTypeFromValue returns a pointer to a valid EventsSortType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewEventsSortTypeFromValue(v string) (*EventsSortType, error) {
	ev := EventsSortType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for EventsSortType: valid values are %v", v, allowedEventsSortTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v EventsSortType) IsValid() bool {
	for _, existing := range allowedEventsSortTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to EventsSortType value.
func (v EventsSortType) Ptr() *EventsSortType {
	return &v
}

// NullableEventsSortType handles when a null is used for EventsSortType.
type NullableEventsSortType struct {
	value *EventsSortType
	isSet bool
}

// Get returns the associated value.
func (v NullableEventsSortType) Get() *EventsSortType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableEventsSortType) Set(val *EventsSortType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableEventsSortType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableEventsSortType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableEventsSortType initializes the struct as if Set has been called.
func NewNullableEventsSortType(val *EventsSortType) *NullableEventsSortType {
	return &NullableEventsSortType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableEventsSortType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableEventsSortType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
