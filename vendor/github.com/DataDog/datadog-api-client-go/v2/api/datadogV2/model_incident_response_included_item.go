// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentResponseIncludedItem - An object related to an incident that is included in the response.
type IncidentResponseIncludedItem struct {
	User                   *User
	IncidentAttachmentData *IncidentAttachmentData

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// UserAsIncidentResponseIncludedItem is a convenience function that returns User wrapped in IncidentResponseIncludedItem.
func UserAsIncidentResponseIncludedItem(v *User) IncidentResponseIncludedItem {
	return IncidentResponseIncludedItem{User: v}
}

// IncidentAttachmentDataAsIncidentResponseIncludedItem is a convenience function that returns IncidentAttachmentData wrapped in IncidentResponseIncludedItem.
func IncidentAttachmentDataAsIncidentResponseIncludedItem(v *IncidentAttachmentData) IncidentResponseIncludedItem {
	return IncidentResponseIncludedItem{IncidentAttachmentData: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *IncidentResponseIncludedItem) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into User
	err = json.Unmarshal(data, &obj.User)
	if err == nil {
		if obj.User != nil && obj.User.UnparsedObject == nil {
			jsonUser, _ := json.Marshal(obj.User)
			if string(jsonUser) == "{}" { // empty struct
				obj.User = nil
			} else {
				match++
			}
		} else {
			obj.User = nil
		}
	} else {
		obj.User = nil
	}

	// try to unmarshal data into IncidentAttachmentData
	err = json.Unmarshal(data, &obj.IncidentAttachmentData)
	if err == nil {
		if obj.IncidentAttachmentData != nil && obj.IncidentAttachmentData.UnparsedObject == nil {
			jsonIncidentAttachmentData, _ := json.Marshal(obj.IncidentAttachmentData)
			if string(jsonIncidentAttachmentData) == "{}" { // empty struct
				obj.IncidentAttachmentData = nil
			} else {
				match++
			}
		} else {
			obj.IncidentAttachmentData = nil
		}
	} else {
		obj.IncidentAttachmentData = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.User = nil
		obj.IncidentAttachmentData = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj IncidentResponseIncludedItem) MarshalJSON() ([]byte, error) {
	if obj.User != nil {
		return json.Marshal(&obj.User)
	}

	if obj.IncidentAttachmentData != nil {
		return json.Marshal(&obj.IncidentAttachmentData)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *IncidentResponseIncludedItem) GetActualInstance() interface{} {
	if obj.User != nil {
		return obj.User
	}

	if obj.IncidentAttachmentData != nil {
		return obj.IncidentAttachmentData
	}

	// all schemas are nil
	return nil
}

// NullableIncidentResponseIncludedItem handles when a null is used for IncidentResponseIncludedItem.
type NullableIncidentResponseIncludedItem struct {
	value *IncidentResponseIncludedItem
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentResponseIncludedItem) Get() *IncidentResponseIncludedItem {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentResponseIncludedItem) Set(val *IncidentResponseIncludedItem) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentResponseIncludedItem) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableIncidentResponseIncludedItem) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentResponseIncludedItem initializes the struct as if Set has been called.
func NewNullableIncidentResponseIncludedItem(val *IncidentResponseIncludedItem) *NullableIncidentResponseIncludedItem {
	return &NullableIncidentResponseIncludedItem{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentResponseIncludedItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentResponseIncludedItem) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
