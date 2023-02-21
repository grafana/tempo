// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsGroupByMissing - The value to use for logs that don't have the facet used to group by
type LogsGroupByMissing struct {
	LogsGroupByMissingString *string
	LogsGroupByMissingNumber *float64

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// LogsGroupByMissingStringAsLogsGroupByMissing is a convenience function that returns string wrapped in LogsGroupByMissing.
func LogsGroupByMissingStringAsLogsGroupByMissing(v *string) LogsGroupByMissing {
	return LogsGroupByMissing{LogsGroupByMissingString: v}
}

// LogsGroupByMissingNumberAsLogsGroupByMissing is a convenience function that returns float64 wrapped in LogsGroupByMissing.
func LogsGroupByMissingNumberAsLogsGroupByMissing(v *float64) LogsGroupByMissing {
	return LogsGroupByMissing{LogsGroupByMissingNumber: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *LogsGroupByMissing) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into LogsGroupByMissingString
	err = json.Unmarshal(data, &obj.LogsGroupByMissingString)
	if err == nil {
		if obj.LogsGroupByMissingString != nil {
			jsonLogsGroupByMissingString, _ := json.Marshal(obj.LogsGroupByMissingString)
			if string(jsonLogsGroupByMissingString) == "{}" { // empty struct
				obj.LogsGroupByMissingString = nil
			} else {
				match++
			}
		} else {
			obj.LogsGroupByMissingString = nil
		}
	} else {
		obj.LogsGroupByMissingString = nil
	}

	// try to unmarshal data into LogsGroupByMissingNumber
	err = json.Unmarshal(data, &obj.LogsGroupByMissingNumber)
	if err == nil {
		if obj.LogsGroupByMissingNumber != nil {
			jsonLogsGroupByMissingNumber, _ := json.Marshal(obj.LogsGroupByMissingNumber)
			if string(jsonLogsGroupByMissingNumber) == "{}" { // empty struct
				obj.LogsGroupByMissingNumber = nil
			} else {
				match++
			}
		} else {
			obj.LogsGroupByMissingNumber = nil
		}
	} else {
		obj.LogsGroupByMissingNumber = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.LogsGroupByMissingString = nil
		obj.LogsGroupByMissingNumber = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj LogsGroupByMissing) MarshalJSON() ([]byte, error) {
	if obj.LogsGroupByMissingString != nil {
		return json.Marshal(&obj.LogsGroupByMissingString)
	}

	if obj.LogsGroupByMissingNumber != nil {
		return json.Marshal(&obj.LogsGroupByMissingNumber)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *LogsGroupByMissing) GetActualInstance() interface{} {
	if obj.LogsGroupByMissingString != nil {
		return obj.LogsGroupByMissingString
	}

	if obj.LogsGroupByMissingNumber != nil {
		return obj.LogsGroupByMissingNumber
	}

	// all schemas are nil
	return nil
}

// NullableLogsGroupByMissing handles when a null is used for LogsGroupByMissing.
type NullableLogsGroupByMissing struct {
	value *LogsGroupByMissing
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsGroupByMissing) Get() *LogsGroupByMissing {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsGroupByMissing) Set(val *LogsGroupByMissing) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsGroupByMissing) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableLogsGroupByMissing) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsGroupByMissing initializes the struct as if Set has been called.
func NewNullableLogsGroupByMissing(val *LogsGroupByMissing) *NullableLogsGroupByMissing {
	return &NullableLogsGroupByMissing{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsGroupByMissing) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsGroupByMissing) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
