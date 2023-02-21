// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// EventsAggregation The type of aggregation that can be performed on events-based queries.
type EventsAggregation string

// List of EventsAggregation.
const (
	EVENTSAGGREGATION_COUNT       EventsAggregation = "count"
	EVENTSAGGREGATION_CARDINALITY EventsAggregation = "cardinality"
	EVENTSAGGREGATION_PC75        EventsAggregation = "pc75"
	EVENTSAGGREGATION_PC90        EventsAggregation = "pc90"
	EVENTSAGGREGATION_PC95        EventsAggregation = "pc95"
	EVENTSAGGREGATION_PC98        EventsAggregation = "pc98"
	EVENTSAGGREGATION_PC99        EventsAggregation = "pc99"
	EVENTSAGGREGATION_SUM         EventsAggregation = "sum"
	EVENTSAGGREGATION_MIN         EventsAggregation = "min"
	EVENTSAGGREGATION_MAX         EventsAggregation = "max"
	EVENTSAGGREGATION_AVG         EventsAggregation = "avg"
)

var allowedEventsAggregationEnumValues = []EventsAggregation{
	EVENTSAGGREGATION_COUNT,
	EVENTSAGGREGATION_CARDINALITY,
	EVENTSAGGREGATION_PC75,
	EVENTSAGGREGATION_PC90,
	EVENTSAGGREGATION_PC95,
	EVENTSAGGREGATION_PC98,
	EVENTSAGGREGATION_PC99,
	EVENTSAGGREGATION_SUM,
	EVENTSAGGREGATION_MIN,
	EVENTSAGGREGATION_MAX,
	EVENTSAGGREGATION_AVG,
}

// GetAllowedValues reeturns the list of possible values.
func (v *EventsAggregation) GetAllowedValues() []EventsAggregation {
	return allowedEventsAggregationEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *EventsAggregation) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = EventsAggregation(value)
	return nil
}

// NewEventsAggregationFromValue returns a pointer to a valid EventsAggregation
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewEventsAggregationFromValue(v string) (*EventsAggregation, error) {
	ev := EventsAggregation(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for EventsAggregation: valid values are %v", v, allowedEventsAggregationEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v EventsAggregation) IsValid() bool {
	for _, existing := range allowedEventsAggregationEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to EventsAggregation value.
func (v EventsAggregation) Ptr() *EventsAggregation {
	return &v
}

// NullableEventsAggregation handles when a null is used for EventsAggregation.
type NullableEventsAggregation struct {
	value *EventsAggregation
	isSet bool
}

// Get returns the associated value.
func (v NullableEventsAggregation) Get() *EventsAggregation {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableEventsAggregation) Set(val *EventsAggregation) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableEventsAggregation) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableEventsAggregation) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableEventsAggregation initializes the struct as if Set has been called.
func NewNullableEventsAggregation(val *EventsAggregation) *NullableEventsAggregation {
	return &NullableEventsAggregation{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableEventsAggregation) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableEventsAggregation) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
