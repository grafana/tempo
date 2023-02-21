// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ApplicationKeyResponseIncludedItem - An object related to an application key.
type ApplicationKeyResponseIncludedItem struct {
	User *User
	Role *Role

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// UserAsApplicationKeyResponseIncludedItem is a convenience function that returns User wrapped in ApplicationKeyResponseIncludedItem.
func UserAsApplicationKeyResponseIncludedItem(v *User) ApplicationKeyResponseIncludedItem {
	return ApplicationKeyResponseIncludedItem{User: v}
}

// RoleAsApplicationKeyResponseIncludedItem is a convenience function that returns Role wrapped in ApplicationKeyResponseIncludedItem.
func RoleAsApplicationKeyResponseIncludedItem(v *Role) ApplicationKeyResponseIncludedItem {
	return ApplicationKeyResponseIncludedItem{Role: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *ApplicationKeyResponseIncludedItem) UnmarshalJSON(data []byte) error {
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

	// try to unmarshal data into Role
	err = json.Unmarshal(data, &obj.Role)
	if err == nil {
		if obj.Role != nil && obj.Role.UnparsedObject == nil {
			jsonRole, _ := json.Marshal(obj.Role)
			if string(jsonRole) == "{}" { // empty struct
				obj.Role = nil
			} else {
				match++
			}
		} else {
			obj.Role = nil
		}
	} else {
		obj.Role = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.User = nil
		obj.Role = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj ApplicationKeyResponseIncludedItem) MarshalJSON() ([]byte, error) {
	if obj.User != nil {
		return json.Marshal(&obj.User)
	}

	if obj.Role != nil {
		return json.Marshal(&obj.Role)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *ApplicationKeyResponseIncludedItem) GetActualInstance() interface{} {
	if obj.User != nil {
		return obj.User
	}

	if obj.Role != nil {
		return obj.Role
	}

	// all schemas are nil
	return nil
}

// NullableApplicationKeyResponseIncludedItem handles when a null is used for ApplicationKeyResponseIncludedItem.
type NullableApplicationKeyResponseIncludedItem struct {
	value *ApplicationKeyResponseIncludedItem
	isSet bool
}

// Get returns the associated value.
func (v NullableApplicationKeyResponseIncludedItem) Get() *ApplicationKeyResponseIncludedItem {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableApplicationKeyResponseIncludedItem) Set(val *ApplicationKeyResponseIncludedItem) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableApplicationKeyResponseIncludedItem) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableApplicationKeyResponseIncludedItem) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableApplicationKeyResponseIncludedItem initializes the struct as if Set has been called.
func NewNullableApplicationKeyResponseIncludedItem(val *ApplicationKeyResponseIncludedItem) *NullableApplicationKeyResponseIncludedItem {
	return &NullableApplicationKeyResponseIncludedItem{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableApplicationKeyResponseIncludedItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableApplicationKeyResponseIncludedItem) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
