// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ScalarColumn - A single column in a scalar query response.
type ScalarColumn struct {
	GroupScalarColumn *GroupScalarColumn
	DataScalarColumn  *DataScalarColumn

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// GroupScalarColumnAsScalarColumn is a convenience function that returns GroupScalarColumn wrapped in ScalarColumn.
func GroupScalarColumnAsScalarColumn(v *GroupScalarColumn) ScalarColumn {
	return ScalarColumn{GroupScalarColumn: v}
}

// DataScalarColumnAsScalarColumn is a convenience function that returns DataScalarColumn wrapped in ScalarColumn.
func DataScalarColumnAsScalarColumn(v *DataScalarColumn) ScalarColumn {
	return ScalarColumn{DataScalarColumn: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *ScalarColumn) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into GroupScalarColumn
	err = json.Unmarshal(data, &obj.GroupScalarColumn)
	if err == nil {
		if obj.GroupScalarColumn != nil && obj.GroupScalarColumn.UnparsedObject == nil {
			jsonGroupScalarColumn, _ := json.Marshal(obj.GroupScalarColumn)
			if string(jsonGroupScalarColumn) == "{}" { // empty struct
				obj.GroupScalarColumn = nil
			} else {
				match++
			}
		} else {
			obj.GroupScalarColumn = nil
		}
	} else {
		obj.GroupScalarColumn = nil
	}

	// try to unmarshal data into DataScalarColumn
	err = json.Unmarshal(data, &obj.DataScalarColumn)
	if err == nil {
		if obj.DataScalarColumn != nil && obj.DataScalarColumn.UnparsedObject == nil {
			jsonDataScalarColumn, _ := json.Marshal(obj.DataScalarColumn)
			if string(jsonDataScalarColumn) == "{}" { // empty struct
				obj.DataScalarColumn = nil
			} else {
				match++
			}
		} else {
			obj.DataScalarColumn = nil
		}
	} else {
		obj.DataScalarColumn = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.GroupScalarColumn = nil
		obj.DataScalarColumn = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj ScalarColumn) MarshalJSON() ([]byte, error) {
	if obj.GroupScalarColumn != nil {
		return json.Marshal(&obj.GroupScalarColumn)
	}

	if obj.DataScalarColumn != nil {
		return json.Marshal(&obj.DataScalarColumn)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *ScalarColumn) GetActualInstance() interface{} {
	if obj.GroupScalarColumn != nil {
		return obj.GroupScalarColumn
	}

	if obj.DataScalarColumn != nil {
		return obj.DataScalarColumn
	}

	// all schemas are nil
	return nil
}

// NullableScalarColumn handles when a null is used for ScalarColumn.
type NullableScalarColumn struct {
	value *ScalarColumn
	isSet bool
}

// Get returns the associated value.
func (v NullableScalarColumn) Get() *ScalarColumn {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableScalarColumn) Set(val *ScalarColumn) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableScalarColumn) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableScalarColumn) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableScalarColumn initializes the struct as if Set has been called.
func NewNullableScalarColumn(val *ScalarColumn) *NullableScalarColumn {
	return &NullableScalarColumn{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableScalarColumn) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableScalarColumn) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
