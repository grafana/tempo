// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsGroupByTotal - A resulting object to put the given computes in over all the matching records.
type LogsGroupByTotal struct {
	LogsGroupByTotalBoolean *bool
	LogsGroupByTotalString  *string
	LogsGroupByTotalNumber  *float64

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// LogsGroupByTotalBooleanAsLogsGroupByTotal is a convenience function that returns bool wrapped in LogsGroupByTotal.
func LogsGroupByTotalBooleanAsLogsGroupByTotal(v *bool) LogsGroupByTotal {
	return LogsGroupByTotal{LogsGroupByTotalBoolean: v}
}

// LogsGroupByTotalStringAsLogsGroupByTotal is a convenience function that returns string wrapped in LogsGroupByTotal.
func LogsGroupByTotalStringAsLogsGroupByTotal(v *string) LogsGroupByTotal {
	return LogsGroupByTotal{LogsGroupByTotalString: v}
}

// LogsGroupByTotalNumberAsLogsGroupByTotal is a convenience function that returns float64 wrapped in LogsGroupByTotal.
func LogsGroupByTotalNumberAsLogsGroupByTotal(v *float64) LogsGroupByTotal {
	return LogsGroupByTotal{LogsGroupByTotalNumber: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *LogsGroupByTotal) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into LogsGroupByTotalBoolean
	err = json.Unmarshal(data, &obj.LogsGroupByTotalBoolean)
	if err == nil {
		if obj.LogsGroupByTotalBoolean != nil {
			jsonLogsGroupByTotalBoolean, _ := json.Marshal(obj.LogsGroupByTotalBoolean)
			if string(jsonLogsGroupByTotalBoolean) == "{}" { // empty struct
				obj.LogsGroupByTotalBoolean = nil
			} else {
				match++
			}
		} else {
			obj.LogsGroupByTotalBoolean = nil
		}
	} else {
		obj.LogsGroupByTotalBoolean = nil
	}

	// try to unmarshal data into LogsGroupByTotalString
	err = json.Unmarshal(data, &obj.LogsGroupByTotalString)
	if err == nil {
		if obj.LogsGroupByTotalString != nil {
			jsonLogsGroupByTotalString, _ := json.Marshal(obj.LogsGroupByTotalString)
			if string(jsonLogsGroupByTotalString) == "{}" { // empty struct
				obj.LogsGroupByTotalString = nil
			} else {
				match++
			}
		} else {
			obj.LogsGroupByTotalString = nil
		}
	} else {
		obj.LogsGroupByTotalString = nil
	}

	// try to unmarshal data into LogsGroupByTotalNumber
	err = json.Unmarshal(data, &obj.LogsGroupByTotalNumber)
	if err == nil {
		if obj.LogsGroupByTotalNumber != nil {
			jsonLogsGroupByTotalNumber, _ := json.Marshal(obj.LogsGroupByTotalNumber)
			if string(jsonLogsGroupByTotalNumber) == "{}" { // empty struct
				obj.LogsGroupByTotalNumber = nil
			} else {
				match++
			}
		} else {
			obj.LogsGroupByTotalNumber = nil
		}
	} else {
		obj.LogsGroupByTotalNumber = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.LogsGroupByTotalBoolean = nil
		obj.LogsGroupByTotalString = nil
		obj.LogsGroupByTotalNumber = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj LogsGroupByTotal) MarshalJSON() ([]byte, error) {
	if obj.LogsGroupByTotalBoolean != nil {
		return json.Marshal(&obj.LogsGroupByTotalBoolean)
	}

	if obj.LogsGroupByTotalString != nil {
		return json.Marshal(&obj.LogsGroupByTotalString)
	}

	if obj.LogsGroupByTotalNumber != nil {
		return json.Marshal(&obj.LogsGroupByTotalNumber)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *LogsGroupByTotal) GetActualInstance() interface{} {
	if obj.LogsGroupByTotalBoolean != nil {
		return obj.LogsGroupByTotalBoolean
	}

	if obj.LogsGroupByTotalString != nil {
		return obj.LogsGroupByTotalString
	}

	if obj.LogsGroupByTotalNumber != nil {
		return obj.LogsGroupByTotalNumber
	}

	// all schemas are nil
	return nil
}

// NullableLogsGroupByTotal handles when a null is used for LogsGroupByTotal.
type NullableLogsGroupByTotal struct {
	value *LogsGroupByTotal
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsGroupByTotal) Get() *LogsGroupByTotal {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsGroupByTotal) Set(val *LogsGroupByTotal) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsGroupByTotal) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableLogsGroupByTotal) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsGroupByTotal initializes the struct as if Set has been called.
func NewNullableLogsGroupByTotal(val *LogsGroupByTotal) *NullableLogsGroupByTotal {
	return &NullableLogsGroupByTotal{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsGroupByTotal) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsGroupByTotal) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
