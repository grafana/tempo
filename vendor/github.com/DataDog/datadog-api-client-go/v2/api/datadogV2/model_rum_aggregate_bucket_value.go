// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RUMAggregateBucketValue - A bucket value, can be either a timeseries or a single value.
type RUMAggregateBucketValue struct {
	RUMAggregateBucketValueSingleString *string
	RUMAggregateBucketValueSingleNumber *float64
	RUMAggregateBucketValueTimeseries   *RUMAggregateBucketValueTimeseries

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// RUMAggregateBucketValueSingleStringAsRUMAggregateBucketValue is a convenience function that returns string wrapped in RUMAggregateBucketValue.
func RUMAggregateBucketValueSingleStringAsRUMAggregateBucketValue(v *string) RUMAggregateBucketValue {
	return RUMAggregateBucketValue{RUMAggregateBucketValueSingleString: v}
}

// RUMAggregateBucketValueSingleNumberAsRUMAggregateBucketValue is a convenience function that returns float64 wrapped in RUMAggregateBucketValue.
func RUMAggregateBucketValueSingleNumberAsRUMAggregateBucketValue(v *float64) RUMAggregateBucketValue {
	return RUMAggregateBucketValue{RUMAggregateBucketValueSingleNumber: v}
}

// RUMAggregateBucketValueTimeseriesAsRUMAggregateBucketValue is a convenience function that returns RUMAggregateBucketValueTimeseries wrapped in RUMAggregateBucketValue.
func RUMAggregateBucketValueTimeseriesAsRUMAggregateBucketValue(v *RUMAggregateBucketValueTimeseries) RUMAggregateBucketValue {
	return RUMAggregateBucketValue{RUMAggregateBucketValueTimeseries: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *RUMAggregateBucketValue) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into RUMAggregateBucketValueSingleString
	err = json.Unmarshal(data, &obj.RUMAggregateBucketValueSingleString)
	if err == nil {
		if obj.RUMAggregateBucketValueSingleString != nil {
			jsonRUMAggregateBucketValueSingleString, _ := json.Marshal(obj.RUMAggregateBucketValueSingleString)
			if string(jsonRUMAggregateBucketValueSingleString) == "{}" { // empty struct
				obj.RUMAggregateBucketValueSingleString = nil
			} else {
				match++
			}
		} else {
			obj.RUMAggregateBucketValueSingleString = nil
		}
	} else {
		obj.RUMAggregateBucketValueSingleString = nil
	}

	// try to unmarshal data into RUMAggregateBucketValueSingleNumber
	err = json.Unmarshal(data, &obj.RUMAggregateBucketValueSingleNumber)
	if err == nil {
		if obj.RUMAggregateBucketValueSingleNumber != nil {
			jsonRUMAggregateBucketValueSingleNumber, _ := json.Marshal(obj.RUMAggregateBucketValueSingleNumber)
			if string(jsonRUMAggregateBucketValueSingleNumber) == "{}" { // empty struct
				obj.RUMAggregateBucketValueSingleNumber = nil
			} else {
				match++
			}
		} else {
			obj.RUMAggregateBucketValueSingleNumber = nil
		}
	} else {
		obj.RUMAggregateBucketValueSingleNumber = nil
	}

	// try to unmarshal data into RUMAggregateBucketValueTimeseries
	err = json.Unmarshal(data, &obj.RUMAggregateBucketValueTimeseries)
	if err == nil {
		if obj.RUMAggregateBucketValueTimeseries != nil {
			jsonRUMAggregateBucketValueTimeseries, _ := json.Marshal(obj.RUMAggregateBucketValueTimeseries)
			if string(jsonRUMAggregateBucketValueTimeseries) == "{}" { // empty struct
				obj.RUMAggregateBucketValueTimeseries = nil
			} else {
				match++
			}
		} else {
			obj.RUMAggregateBucketValueTimeseries = nil
		}
	} else {
		obj.RUMAggregateBucketValueTimeseries = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.RUMAggregateBucketValueSingleString = nil
		obj.RUMAggregateBucketValueSingleNumber = nil
		obj.RUMAggregateBucketValueTimeseries = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj RUMAggregateBucketValue) MarshalJSON() ([]byte, error) {
	if obj.RUMAggregateBucketValueSingleString != nil {
		return json.Marshal(&obj.RUMAggregateBucketValueSingleString)
	}

	if obj.RUMAggregateBucketValueSingleNumber != nil {
		return json.Marshal(&obj.RUMAggregateBucketValueSingleNumber)
	}

	if obj.RUMAggregateBucketValueTimeseries != nil {
		return json.Marshal(&obj.RUMAggregateBucketValueTimeseries)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *RUMAggregateBucketValue) GetActualInstance() interface{} {
	if obj.RUMAggregateBucketValueSingleString != nil {
		return obj.RUMAggregateBucketValueSingleString
	}

	if obj.RUMAggregateBucketValueSingleNumber != nil {
		return obj.RUMAggregateBucketValueSingleNumber
	}

	if obj.RUMAggregateBucketValueTimeseries != nil {
		return obj.RUMAggregateBucketValueTimeseries
	}

	// all schemas are nil
	return nil
}

// NullableRUMAggregateBucketValue handles when a null is used for RUMAggregateBucketValue.
type NullableRUMAggregateBucketValue struct {
	value *RUMAggregateBucketValue
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMAggregateBucketValue) Get() *RUMAggregateBucketValue {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMAggregateBucketValue) Set(val *RUMAggregateBucketValue) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMAggregateBucketValue) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableRUMAggregateBucketValue) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMAggregateBucketValue initializes the struct as if Set has been called.
func NewNullableRUMAggregateBucketValue(val *RUMAggregateBucketValue) *NullableRUMAggregateBucketValue {
	return &NullableRUMAggregateBucketValue{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMAggregateBucketValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMAggregateBucketValue) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
