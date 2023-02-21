// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentTimelineCellCreateAttributes - The timeline cell's attributes for a create request.
type IncidentTimelineCellCreateAttributes struct {
	IncidentTimelineCellMarkdownCreateAttributes *IncidentTimelineCellMarkdownCreateAttributes

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// IncidentTimelineCellMarkdownCreateAttributesAsIncidentTimelineCellCreateAttributes is a convenience function that returns IncidentTimelineCellMarkdownCreateAttributes wrapped in IncidentTimelineCellCreateAttributes.
func IncidentTimelineCellMarkdownCreateAttributesAsIncidentTimelineCellCreateAttributes(v *IncidentTimelineCellMarkdownCreateAttributes) IncidentTimelineCellCreateAttributes {
	return IncidentTimelineCellCreateAttributes{IncidentTimelineCellMarkdownCreateAttributes: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *IncidentTimelineCellCreateAttributes) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into IncidentTimelineCellMarkdownCreateAttributes
	err = json.Unmarshal(data, &obj.IncidentTimelineCellMarkdownCreateAttributes)
	if err == nil {
		if obj.IncidentTimelineCellMarkdownCreateAttributes != nil && obj.IncidentTimelineCellMarkdownCreateAttributes.UnparsedObject == nil {
			jsonIncidentTimelineCellMarkdownCreateAttributes, _ := json.Marshal(obj.IncidentTimelineCellMarkdownCreateAttributes)
			if string(jsonIncidentTimelineCellMarkdownCreateAttributes) == "{}" { // empty struct
				obj.IncidentTimelineCellMarkdownCreateAttributes = nil
			} else {
				match++
			}
		} else {
			obj.IncidentTimelineCellMarkdownCreateAttributes = nil
		}
	} else {
		obj.IncidentTimelineCellMarkdownCreateAttributes = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.IncidentTimelineCellMarkdownCreateAttributes = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj IncidentTimelineCellCreateAttributes) MarshalJSON() ([]byte, error) {
	if obj.IncidentTimelineCellMarkdownCreateAttributes != nil {
		return json.Marshal(&obj.IncidentTimelineCellMarkdownCreateAttributes)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *IncidentTimelineCellCreateAttributes) GetActualInstance() interface{} {
	if obj.IncidentTimelineCellMarkdownCreateAttributes != nil {
		return obj.IncidentTimelineCellMarkdownCreateAttributes
	}

	// all schemas are nil
	return nil
}

// NullableIncidentTimelineCellCreateAttributes handles when a null is used for IncidentTimelineCellCreateAttributes.
type NullableIncidentTimelineCellCreateAttributes struct {
	value *IncidentTimelineCellCreateAttributes
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentTimelineCellCreateAttributes) Get() *IncidentTimelineCellCreateAttributes {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentTimelineCellCreateAttributes) Set(val *IncidentTimelineCellCreateAttributes) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentTimelineCellCreateAttributes) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableIncidentTimelineCellCreateAttributes) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentTimelineCellCreateAttributes initializes the struct as if Set has been called.
func NewNullableIncidentTimelineCellCreateAttributes(val *IncidentTimelineCellCreateAttributes) *NullableIncidentTimelineCellCreateAttributes {
	return &NullableIncidentTimelineCellCreateAttributes{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentTimelineCellCreateAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentTimelineCellCreateAttributes) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
