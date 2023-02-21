// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RUMGroupByMissing - The value to use for logs that don't have the facet used to group by.
type RUMGroupByMissing struct {
	RUMGroupByMissingString *string
	RUMGroupByMissingNumber *float64

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// RUMGroupByMissingStringAsRUMGroupByMissing is a convenience function that returns string wrapped in RUMGroupByMissing.
func RUMGroupByMissingStringAsRUMGroupByMissing(v *string) RUMGroupByMissing {
	return RUMGroupByMissing{RUMGroupByMissingString: v}
}

// RUMGroupByMissingNumberAsRUMGroupByMissing is a convenience function that returns float64 wrapped in RUMGroupByMissing.
func RUMGroupByMissingNumberAsRUMGroupByMissing(v *float64) RUMGroupByMissing {
	return RUMGroupByMissing{RUMGroupByMissingNumber: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *RUMGroupByMissing) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into RUMGroupByMissingString
	err = json.Unmarshal(data, &obj.RUMGroupByMissingString)
	if err == nil {
		if obj.RUMGroupByMissingString != nil {
			jsonRUMGroupByMissingString, _ := json.Marshal(obj.RUMGroupByMissingString)
			if string(jsonRUMGroupByMissingString) == "{}" { // empty struct
				obj.RUMGroupByMissingString = nil
			} else {
				match++
			}
		} else {
			obj.RUMGroupByMissingString = nil
		}
	} else {
		obj.RUMGroupByMissingString = nil
	}

	// try to unmarshal data into RUMGroupByMissingNumber
	err = json.Unmarshal(data, &obj.RUMGroupByMissingNumber)
	if err == nil {
		if obj.RUMGroupByMissingNumber != nil {
			jsonRUMGroupByMissingNumber, _ := json.Marshal(obj.RUMGroupByMissingNumber)
			if string(jsonRUMGroupByMissingNumber) == "{}" { // empty struct
				obj.RUMGroupByMissingNumber = nil
			} else {
				match++
			}
		} else {
			obj.RUMGroupByMissingNumber = nil
		}
	} else {
		obj.RUMGroupByMissingNumber = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.RUMGroupByMissingString = nil
		obj.RUMGroupByMissingNumber = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj RUMGroupByMissing) MarshalJSON() ([]byte, error) {
	if obj.RUMGroupByMissingString != nil {
		return json.Marshal(&obj.RUMGroupByMissingString)
	}

	if obj.RUMGroupByMissingNumber != nil {
		return json.Marshal(&obj.RUMGroupByMissingNumber)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *RUMGroupByMissing) GetActualInstance() interface{} {
	if obj.RUMGroupByMissingString != nil {
		return obj.RUMGroupByMissingString
	}

	if obj.RUMGroupByMissingNumber != nil {
		return obj.RUMGroupByMissingNumber
	}

	// all schemas are nil
	return nil
}

// NullableRUMGroupByMissing handles when a null is used for RUMGroupByMissing.
type NullableRUMGroupByMissing struct {
	value *RUMGroupByMissing
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMGroupByMissing) Get() *RUMGroupByMissing {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMGroupByMissing) Set(val *RUMGroupByMissing) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMGroupByMissing) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableRUMGroupByMissing) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMGroupByMissing initializes the struct as if Set has been called.
func NewNullableRUMGroupByMissing(val *RUMGroupByMissing) *NullableRUMGroupByMissing {
	return &NullableRUMGroupByMissing{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMGroupByMissing) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMGroupByMissing) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
