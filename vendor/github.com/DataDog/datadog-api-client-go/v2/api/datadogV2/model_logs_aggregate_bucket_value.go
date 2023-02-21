// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsAggregateBucketValue - A bucket value, can be either a timeseries or a single value
type LogsAggregateBucketValue struct {
	LogsAggregateBucketValueSingleString *string
	LogsAggregateBucketValueSingleNumber *float64
	LogsAggregateBucketValueTimeseries   *LogsAggregateBucketValueTimeseries

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// LogsAggregateBucketValueSingleStringAsLogsAggregateBucketValue is a convenience function that returns string wrapped in LogsAggregateBucketValue.
func LogsAggregateBucketValueSingleStringAsLogsAggregateBucketValue(v *string) LogsAggregateBucketValue {
	return LogsAggregateBucketValue{LogsAggregateBucketValueSingleString: v}
}

// LogsAggregateBucketValueSingleNumberAsLogsAggregateBucketValue is a convenience function that returns float64 wrapped in LogsAggregateBucketValue.
func LogsAggregateBucketValueSingleNumberAsLogsAggregateBucketValue(v *float64) LogsAggregateBucketValue {
	return LogsAggregateBucketValue{LogsAggregateBucketValueSingleNumber: v}
}

// LogsAggregateBucketValueTimeseriesAsLogsAggregateBucketValue is a convenience function that returns LogsAggregateBucketValueTimeseries wrapped in LogsAggregateBucketValue.
func LogsAggregateBucketValueTimeseriesAsLogsAggregateBucketValue(v *LogsAggregateBucketValueTimeseries) LogsAggregateBucketValue {
	return LogsAggregateBucketValue{LogsAggregateBucketValueTimeseries: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *LogsAggregateBucketValue) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into LogsAggregateBucketValueSingleString
	err = json.Unmarshal(data, &obj.LogsAggregateBucketValueSingleString)
	if err == nil {
		if obj.LogsAggregateBucketValueSingleString != nil {
			jsonLogsAggregateBucketValueSingleString, _ := json.Marshal(obj.LogsAggregateBucketValueSingleString)
			if string(jsonLogsAggregateBucketValueSingleString) == "{}" { // empty struct
				obj.LogsAggregateBucketValueSingleString = nil
			} else {
				match++
			}
		} else {
			obj.LogsAggregateBucketValueSingleString = nil
		}
	} else {
		obj.LogsAggregateBucketValueSingleString = nil
	}

	// try to unmarshal data into LogsAggregateBucketValueSingleNumber
	err = json.Unmarshal(data, &obj.LogsAggregateBucketValueSingleNumber)
	if err == nil {
		if obj.LogsAggregateBucketValueSingleNumber != nil {
			jsonLogsAggregateBucketValueSingleNumber, _ := json.Marshal(obj.LogsAggregateBucketValueSingleNumber)
			if string(jsonLogsAggregateBucketValueSingleNumber) == "{}" { // empty struct
				obj.LogsAggregateBucketValueSingleNumber = nil
			} else {
				match++
			}
		} else {
			obj.LogsAggregateBucketValueSingleNumber = nil
		}
	} else {
		obj.LogsAggregateBucketValueSingleNumber = nil
	}

	// try to unmarshal data into LogsAggregateBucketValueTimeseries
	err = json.Unmarshal(data, &obj.LogsAggregateBucketValueTimeseries)
	if err == nil {
		if obj.LogsAggregateBucketValueTimeseries != nil {
			jsonLogsAggregateBucketValueTimeseries, _ := json.Marshal(obj.LogsAggregateBucketValueTimeseries)
			if string(jsonLogsAggregateBucketValueTimeseries) == "{}" { // empty struct
				obj.LogsAggregateBucketValueTimeseries = nil
			} else {
				match++
			}
		} else {
			obj.LogsAggregateBucketValueTimeseries = nil
		}
	} else {
		obj.LogsAggregateBucketValueTimeseries = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.LogsAggregateBucketValueSingleString = nil
		obj.LogsAggregateBucketValueSingleNumber = nil
		obj.LogsAggregateBucketValueTimeseries = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj LogsAggregateBucketValue) MarshalJSON() ([]byte, error) {
	if obj.LogsAggregateBucketValueSingleString != nil {
		return json.Marshal(&obj.LogsAggregateBucketValueSingleString)
	}

	if obj.LogsAggregateBucketValueSingleNumber != nil {
		return json.Marshal(&obj.LogsAggregateBucketValueSingleNumber)
	}

	if obj.LogsAggregateBucketValueTimeseries != nil {
		return json.Marshal(&obj.LogsAggregateBucketValueTimeseries)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *LogsAggregateBucketValue) GetActualInstance() interface{} {
	if obj.LogsAggregateBucketValueSingleString != nil {
		return obj.LogsAggregateBucketValueSingleString
	}

	if obj.LogsAggregateBucketValueSingleNumber != nil {
		return obj.LogsAggregateBucketValueSingleNumber
	}

	if obj.LogsAggregateBucketValueTimeseries != nil {
		return obj.LogsAggregateBucketValueTimeseries
	}

	// all schemas are nil
	return nil
}

// NullableLogsAggregateBucketValue handles when a null is used for LogsAggregateBucketValue.
type NullableLogsAggregateBucketValue struct {
	value *LogsAggregateBucketValue
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsAggregateBucketValue) Get() *LogsAggregateBucketValue {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsAggregateBucketValue) Set(val *LogsAggregateBucketValue) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsAggregateBucketValue) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableLogsAggregateBucketValue) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsAggregateBucketValue initializes the struct as if Set has been called.
func NewNullableLogsAggregateBucketValue(val *LogsAggregateBucketValue) *NullableLogsAggregateBucketValue {
	return &NullableLogsAggregateBucketValue{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsAggregateBucketValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsAggregateBucketValue) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
