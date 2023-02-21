// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// EventType Type of the event.
type EventType string

// List of EventType.
const (
	EVENTTYPE_EVENT EventType = "event"
)

var allowedEventTypeEnumValues = []EventType{
	EVENTTYPE_EVENT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *EventType) GetAllowedValues() []EventType {
	return allowedEventTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *EventType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = EventType(value)
	return nil
}

// NewEventTypeFromValue returns a pointer to a valid EventType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewEventTypeFromValue(v string) (*EventType, error) {
	ev := EventType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for EventType: valid values are %v", v, allowedEventTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v EventType) IsValid() bool {
	for _, existing := range allowedEventTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to EventType value.
func (v EventType) Ptr() *EventType {
	return &v
}

// NullableEventType handles when a null is used for EventType.
type NullableEventType struct {
	value *EventType
	isSet bool
}

// Get returns the associated value.
func (v NullableEventType) Get() *EventType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableEventType) Set(val *EventType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableEventType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableEventType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableEventType initializes the struct as if Set has been called.
func NewNullableEventType(val *EventType) *NullableEventType {
	return &NullableEventType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableEventType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableEventType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
