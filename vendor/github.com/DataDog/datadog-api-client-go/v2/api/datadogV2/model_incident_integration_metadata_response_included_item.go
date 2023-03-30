// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentIntegrationMetadataResponseIncludedItem - An object related to an incident integration metadata that is included in the response.
type IncidentIntegrationMetadataResponseIncludedItem struct {
	User *User

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// UserAsIncidentIntegrationMetadataResponseIncludedItem is a convenience function that returns User wrapped in IncidentIntegrationMetadataResponseIncludedItem.
func UserAsIncidentIntegrationMetadataResponseIncludedItem(v *User) IncidentIntegrationMetadataResponseIncludedItem {
	return IncidentIntegrationMetadataResponseIncludedItem{User: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *IncidentIntegrationMetadataResponseIncludedItem) UnmarshalJSON(data []byte) error {
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

	if match != 1 { // more than 1 match
		// reset to nil
		obj.User = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj IncidentIntegrationMetadataResponseIncludedItem) MarshalJSON() ([]byte, error) {
	if obj.User != nil {
		return json.Marshal(&obj.User)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *IncidentIntegrationMetadataResponseIncludedItem) GetActualInstance() interface{} {
	if obj.User != nil {
		return obj.User
	}

	// all schemas are nil
	return nil
}

// NullableIncidentIntegrationMetadataResponseIncludedItem handles when a null is used for IncidentIntegrationMetadataResponseIncludedItem.
type NullableIncidentIntegrationMetadataResponseIncludedItem struct {
	value *IncidentIntegrationMetadataResponseIncludedItem
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentIntegrationMetadataResponseIncludedItem) Get() *IncidentIntegrationMetadataResponseIncludedItem {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentIntegrationMetadataResponseIncludedItem) Set(val *IncidentIntegrationMetadataResponseIncludedItem) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentIntegrationMetadataResponseIncludedItem) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableIncidentIntegrationMetadataResponseIncludedItem) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentIntegrationMetadataResponseIncludedItem initializes the struct as if Set has been called.
func NewNullableIncidentIntegrationMetadataResponseIncludedItem(val *IncidentIntegrationMetadataResponseIncludedItem) *NullableIncidentIntegrationMetadataResponseIncludedItem {
	return &NullableIncidentIntegrationMetadataResponseIncludedItem{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentIntegrationMetadataResponseIncludedItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentIntegrationMetadataResponseIncludedItem) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
