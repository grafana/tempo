// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// TimeseriesQuery - An individual timeseries query to one of the basic Datadog data sources.
type TimeseriesQuery struct {
	MetricsTimeseriesQuery *MetricsTimeseriesQuery
	EventsTimeseriesQuery  *EventsTimeseriesQuery

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// MetricsTimeseriesQueryAsTimeseriesQuery is a convenience function that returns MetricsTimeseriesQuery wrapped in TimeseriesQuery.
func MetricsTimeseriesQueryAsTimeseriesQuery(v *MetricsTimeseriesQuery) TimeseriesQuery {
	return TimeseriesQuery{MetricsTimeseriesQuery: v}
}

// EventsTimeseriesQueryAsTimeseriesQuery is a convenience function that returns EventsTimeseriesQuery wrapped in TimeseriesQuery.
func EventsTimeseriesQueryAsTimeseriesQuery(v *EventsTimeseriesQuery) TimeseriesQuery {
	return TimeseriesQuery{EventsTimeseriesQuery: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *TimeseriesQuery) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into MetricsTimeseriesQuery
	err = json.Unmarshal(data, &obj.MetricsTimeseriesQuery)
	if err == nil {
		if obj.MetricsTimeseriesQuery != nil && obj.MetricsTimeseriesQuery.UnparsedObject == nil {
			jsonMetricsTimeseriesQuery, _ := json.Marshal(obj.MetricsTimeseriesQuery)
			if string(jsonMetricsTimeseriesQuery) == "{}" { // empty struct
				obj.MetricsTimeseriesQuery = nil
			} else {
				match++
			}
		} else {
			obj.MetricsTimeseriesQuery = nil
		}
	} else {
		obj.MetricsTimeseriesQuery = nil
	}

	// try to unmarshal data into EventsTimeseriesQuery
	err = json.Unmarshal(data, &obj.EventsTimeseriesQuery)
	if err == nil {
		if obj.EventsTimeseriesQuery != nil && obj.EventsTimeseriesQuery.UnparsedObject == nil {
			jsonEventsTimeseriesQuery, _ := json.Marshal(obj.EventsTimeseriesQuery)
			if string(jsonEventsTimeseriesQuery) == "{}" { // empty struct
				obj.EventsTimeseriesQuery = nil
			} else {
				match++
			}
		} else {
			obj.EventsTimeseriesQuery = nil
		}
	} else {
		obj.EventsTimeseriesQuery = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.MetricsTimeseriesQuery = nil
		obj.EventsTimeseriesQuery = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj TimeseriesQuery) MarshalJSON() ([]byte, error) {
	if obj.MetricsTimeseriesQuery != nil {
		return json.Marshal(&obj.MetricsTimeseriesQuery)
	}

	if obj.EventsTimeseriesQuery != nil {
		return json.Marshal(&obj.EventsTimeseriesQuery)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *TimeseriesQuery) GetActualInstance() interface{} {
	if obj.MetricsTimeseriesQuery != nil {
		return obj.MetricsTimeseriesQuery
	}

	if obj.EventsTimeseriesQuery != nil {
		return obj.EventsTimeseriesQuery
	}

	// all schemas are nil
	return nil
}

// NullableTimeseriesQuery handles when a null is used for TimeseriesQuery.
type NullableTimeseriesQuery struct {
	value *TimeseriesQuery
	isSet bool
}

// Get returns the associated value.
func (v NullableTimeseriesQuery) Get() *TimeseriesQuery {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableTimeseriesQuery) Set(val *TimeseriesQuery) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableTimeseriesQuery) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableTimeseriesQuery) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableTimeseriesQuery initializes the struct as if Set has been called.
func NewNullableTimeseriesQuery(val *TimeseriesQuery) *NullableTimeseriesQuery {
	return &NullableTimeseriesQuery{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableTimeseriesQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableTimeseriesQuery) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
