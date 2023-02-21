// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentFieldAttributes - Dynamic fields for which selections can be made, with field names as keys.
type IncidentFieldAttributes struct {
	IncidentFieldAttributesSingleValue   *IncidentFieldAttributesSingleValue
	IncidentFieldAttributesMultipleValue *IncidentFieldAttributesMultipleValue

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// IncidentFieldAttributesSingleValueAsIncidentFieldAttributes is a convenience function that returns IncidentFieldAttributesSingleValue wrapped in IncidentFieldAttributes.
func IncidentFieldAttributesSingleValueAsIncidentFieldAttributes(v *IncidentFieldAttributesSingleValue) IncidentFieldAttributes {
	return IncidentFieldAttributes{IncidentFieldAttributesSingleValue: v}
}

// IncidentFieldAttributesMultipleValueAsIncidentFieldAttributes is a convenience function that returns IncidentFieldAttributesMultipleValue wrapped in IncidentFieldAttributes.
func IncidentFieldAttributesMultipleValueAsIncidentFieldAttributes(v *IncidentFieldAttributesMultipleValue) IncidentFieldAttributes {
	return IncidentFieldAttributes{IncidentFieldAttributesMultipleValue: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *IncidentFieldAttributes) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into IncidentFieldAttributesSingleValue
	err = json.Unmarshal(data, &obj.IncidentFieldAttributesSingleValue)
	if err == nil {
		if obj.IncidentFieldAttributesSingleValue != nil && obj.IncidentFieldAttributesSingleValue.UnparsedObject == nil {
			jsonIncidentFieldAttributesSingleValue, _ := json.Marshal(obj.IncidentFieldAttributesSingleValue)
			if string(jsonIncidentFieldAttributesSingleValue) == "{}" { // empty struct
				obj.IncidentFieldAttributesSingleValue = nil
			} else {
				match++
			}
		} else {
			obj.IncidentFieldAttributesSingleValue = nil
		}
	} else {
		obj.IncidentFieldAttributesSingleValue = nil
	}

	// try to unmarshal data into IncidentFieldAttributesMultipleValue
	err = json.Unmarshal(data, &obj.IncidentFieldAttributesMultipleValue)
	if err == nil {
		if obj.IncidentFieldAttributesMultipleValue != nil && obj.IncidentFieldAttributesMultipleValue.UnparsedObject == nil {
			jsonIncidentFieldAttributesMultipleValue, _ := json.Marshal(obj.IncidentFieldAttributesMultipleValue)
			if string(jsonIncidentFieldAttributesMultipleValue) == "{}" { // empty struct
				obj.IncidentFieldAttributesMultipleValue = nil
			} else {
				match++
			}
		} else {
			obj.IncidentFieldAttributesMultipleValue = nil
		}
	} else {
		obj.IncidentFieldAttributesMultipleValue = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.IncidentFieldAttributesSingleValue = nil
		obj.IncidentFieldAttributesMultipleValue = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj IncidentFieldAttributes) MarshalJSON() ([]byte, error) {
	if obj.IncidentFieldAttributesSingleValue != nil {
		return json.Marshal(&obj.IncidentFieldAttributesSingleValue)
	}

	if obj.IncidentFieldAttributesMultipleValue != nil {
		return json.Marshal(&obj.IncidentFieldAttributesMultipleValue)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *IncidentFieldAttributes) GetActualInstance() interface{} {
	if obj.IncidentFieldAttributesSingleValue != nil {
		return obj.IncidentFieldAttributesSingleValue
	}

	if obj.IncidentFieldAttributesMultipleValue != nil {
		return obj.IncidentFieldAttributesMultipleValue
	}

	// all schemas are nil
	return nil
}

// NullableIncidentFieldAttributes handles when a null is used for IncidentFieldAttributes.
type NullableIncidentFieldAttributes struct {
	value *IncidentFieldAttributes
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentFieldAttributes) Get() *IncidentFieldAttributes {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentFieldAttributes) Set(val *IncidentFieldAttributes) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentFieldAttributes) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableIncidentFieldAttributes) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentFieldAttributes initializes the struct as if Set has been called.
func NewNullableIncidentFieldAttributes(val *IncidentFieldAttributes) *NullableIncidentFieldAttributes {
	return &NullableIncidentFieldAttributes{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentFieldAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentFieldAttributes) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
