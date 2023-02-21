// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// EventStatusType If an alert event is enabled, its status is one of the following:
// `failure`, `error`, `warning`, `info`, `success`, `user_update`,
// `recommendation`, or `snapshot`.
type EventStatusType string

// List of EventStatusType.
const (
	EVENTSTATUSTYPE_FAILURE        EventStatusType = "failure"
	EVENTSTATUSTYPE_ERROR          EventStatusType = "error"
	EVENTSTATUSTYPE_WARNING        EventStatusType = "warning"
	EVENTSTATUSTYPE_INFO           EventStatusType = "info"
	EVENTSTATUSTYPE_SUCCESS        EventStatusType = "success"
	EVENTSTATUSTYPE_USER_UPDATE    EventStatusType = "user_update"
	EVENTSTATUSTYPE_RECOMMENDATION EventStatusType = "recommendation"
	EVENTSTATUSTYPE_SNAPSHOT       EventStatusType = "snapshot"
)

var allowedEventStatusTypeEnumValues = []EventStatusType{
	EVENTSTATUSTYPE_FAILURE,
	EVENTSTATUSTYPE_ERROR,
	EVENTSTATUSTYPE_WARNING,
	EVENTSTATUSTYPE_INFO,
	EVENTSTATUSTYPE_SUCCESS,
	EVENTSTATUSTYPE_USER_UPDATE,
	EVENTSTATUSTYPE_RECOMMENDATION,
	EVENTSTATUSTYPE_SNAPSHOT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *EventStatusType) GetAllowedValues() []EventStatusType {
	return allowedEventStatusTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *EventStatusType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = EventStatusType(value)
	return nil
}

// NewEventStatusTypeFromValue returns a pointer to a valid EventStatusType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewEventStatusTypeFromValue(v string) (*EventStatusType, error) {
	ev := EventStatusType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for EventStatusType: valid values are %v", v, allowedEventStatusTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v EventStatusType) IsValid() bool {
	for _, existing := range allowedEventStatusTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to EventStatusType value.
func (v EventStatusType) Ptr() *EventStatusType {
	return &v
}

// NullableEventStatusType handles when a null is used for EventStatusType.
type NullableEventStatusType struct {
	value *EventStatusType
	isSet bool
}

// Get returns the associated value.
func (v NullableEventStatusType) Get() *EventStatusType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableEventStatusType) Set(val *EventStatusType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableEventStatusType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableEventStatusType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableEventStatusType initializes the struct as if Set has been called.
func NewNullableEventStatusType(val *EventStatusType) *NullableEventStatusType {
	return &NullableEventStatusType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableEventStatusType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableEventStatusType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
