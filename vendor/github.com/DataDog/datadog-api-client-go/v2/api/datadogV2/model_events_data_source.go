// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// EventsDataSource A data source that is powered by the Events Platform.
type EventsDataSource string

// List of EventsDataSource.
const (
	EVENTSDATASOURCE_LOGS EventsDataSource = "logs"
	EVENTSDATASOURCE_RUM  EventsDataSource = "rum"
)

var allowedEventsDataSourceEnumValues = []EventsDataSource{
	EVENTSDATASOURCE_LOGS,
	EVENTSDATASOURCE_RUM,
}

// GetAllowedValues reeturns the list of possible values.
func (v *EventsDataSource) GetAllowedValues() []EventsDataSource {
	return allowedEventsDataSourceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *EventsDataSource) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = EventsDataSource(value)
	return nil
}

// NewEventsDataSourceFromValue returns a pointer to a valid EventsDataSource
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewEventsDataSourceFromValue(v string) (*EventsDataSource, error) {
	ev := EventsDataSource(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for EventsDataSource: valid values are %v", v, allowedEventsDataSourceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v EventsDataSource) IsValid() bool {
	for _, existing := range allowedEventsDataSourceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to EventsDataSource value.
func (v EventsDataSource) Ptr() *EventsDataSource {
	return &v
}

// NullableEventsDataSource handles when a null is used for EventsDataSource.
type NullableEventsDataSource struct {
	value *EventsDataSource
	isSet bool
}

// Get returns the associated value.
func (v NullableEventsDataSource) Get() *EventsDataSource {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableEventsDataSource) Set(val *EventsDataSource) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableEventsDataSource) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableEventsDataSource) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableEventsDataSource initializes the struct as if Set has been called.
func NewNullableEventsDataSource(val *EventsDataSource) *NullableEventsDataSource {
	return &NullableEventsDataSource{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableEventsDataSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableEventsDataSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
