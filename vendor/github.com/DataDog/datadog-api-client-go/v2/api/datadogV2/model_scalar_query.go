// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ScalarQuery - An individual scalar query to one of the basic Datadog data sources.
type ScalarQuery struct {
	MetricsScalarQuery *MetricsScalarQuery
	EventsScalarQuery  *EventsScalarQuery

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// MetricsScalarQueryAsScalarQuery is a convenience function that returns MetricsScalarQuery wrapped in ScalarQuery.
func MetricsScalarQueryAsScalarQuery(v *MetricsScalarQuery) ScalarQuery {
	return ScalarQuery{MetricsScalarQuery: v}
}

// EventsScalarQueryAsScalarQuery is a convenience function that returns EventsScalarQuery wrapped in ScalarQuery.
func EventsScalarQueryAsScalarQuery(v *EventsScalarQuery) ScalarQuery {
	return ScalarQuery{EventsScalarQuery: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *ScalarQuery) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into MetricsScalarQuery
	err = json.Unmarshal(data, &obj.MetricsScalarQuery)
	if err == nil {
		if obj.MetricsScalarQuery != nil && obj.MetricsScalarQuery.UnparsedObject == nil {
			jsonMetricsScalarQuery, _ := json.Marshal(obj.MetricsScalarQuery)
			if string(jsonMetricsScalarQuery) == "{}" { // empty struct
				obj.MetricsScalarQuery = nil
			} else {
				match++
			}
		} else {
			obj.MetricsScalarQuery = nil
		}
	} else {
		obj.MetricsScalarQuery = nil
	}

	// try to unmarshal data into EventsScalarQuery
	err = json.Unmarshal(data, &obj.EventsScalarQuery)
	if err == nil {
		if obj.EventsScalarQuery != nil && obj.EventsScalarQuery.UnparsedObject == nil {
			jsonEventsScalarQuery, _ := json.Marshal(obj.EventsScalarQuery)
			if string(jsonEventsScalarQuery) == "{}" { // empty struct
				obj.EventsScalarQuery = nil
			} else {
				match++
			}
		} else {
			obj.EventsScalarQuery = nil
		}
	} else {
		obj.EventsScalarQuery = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.MetricsScalarQuery = nil
		obj.EventsScalarQuery = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj ScalarQuery) MarshalJSON() ([]byte, error) {
	if obj.MetricsScalarQuery != nil {
		return json.Marshal(&obj.MetricsScalarQuery)
	}

	if obj.EventsScalarQuery != nil {
		return json.Marshal(&obj.EventsScalarQuery)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *ScalarQuery) GetActualInstance() interface{} {
	if obj.MetricsScalarQuery != nil {
		return obj.MetricsScalarQuery
	}

	if obj.EventsScalarQuery != nil {
		return obj.EventsScalarQuery
	}

	// all schemas are nil
	return nil
}

// NullableScalarQuery handles when a null is used for ScalarQuery.
type NullableScalarQuery struct {
	value *ScalarQuery
	isSet bool
}

// Get returns the associated value.
func (v NullableScalarQuery) Get() *ScalarQuery {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableScalarQuery) Set(val *ScalarQuery) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableScalarQuery) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableScalarQuery) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableScalarQuery initializes the struct as if Set has been called.
func NewNullableScalarQuery(val *ScalarQuery) *NullableScalarQuery {
	return &NullableScalarQuery{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableScalarQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableScalarQuery) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
