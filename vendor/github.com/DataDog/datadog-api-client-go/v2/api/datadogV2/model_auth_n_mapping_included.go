// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// AuthNMappingIncluded - Included data in the AuthN Mapping response.
type AuthNMappingIncluded struct {
	SAMLAssertionAttribute *SAMLAssertionAttribute
	Role                   *Role

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SAMLAssertionAttributeAsAuthNMappingIncluded is a convenience function that returns SAMLAssertionAttribute wrapped in AuthNMappingIncluded.
func SAMLAssertionAttributeAsAuthNMappingIncluded(v *SAMLAssertionAttribute) AuthNMappingIncluded {
	return AuthNMappingIncluded{SAMLAssertionAttribute: v}
}

// RoleAsAuthNMappingIncluded is a convenience function that returns Role wrapped in AuthNMappingIncluded.
func RoleAsAuthNMappingIncluded(v *Role) AuthNMappingIncluded {
	return AuthNMappingIncluded{Role: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *AuthNMappingIncluded) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SAMLAssertionAttribute
	err = json.Unmarshal(data, &obj.SAMLAssertionAttribute)
	if err == nil {
		if obj.SAMLAssertionAttribute != nil && obj.SAMLAssertionAttribute.UnparsedObject == nil {
			jsonSAMLAssertionAttribute, _ := json.Marshal(obj.SAMLAssertionAttribute)
			if string(jsonSAMLAssertionAttribute) == "{}" { // empty struct
				obj.SAMLAssertionAttribute = nil
			} else {
				match++
			}
		} else {
			obj.SAMLAssertionAttribute = nil
		}
	} else {
		obj.SAMLAssertionAttribute = nil
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
		obj.SAMLAssertionAttribute = nil
		obj.Role = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj AuthNMappingIncluded) MarshalJSON() ([]byte, error) {
	if obj.SAMLAssertionAttribute != nil {
		return json.Marshal(&obj.SAMLAssertionAttribute)
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
func (obj *AuthNMappingIncluded) GetActualInstance() interface{} {
	if obj.SAMLAssertionAttribute != nil {
		return obj.SAMLAssertionAttribute
	}

	if obj.Role != nil {
		return obj.Role
	}

	// all schemas are nil
	return nil
}

// NullableAuthNMappingIncluded handles when a null is used for AuthNMappingIncluded.
type NullableAuthNMappingIncluded struct {
	value *AuthNMappingIncluded
	isSet bool
}

// Get returns the associated value.
func (v NullableAuthNMappingIncluded) Get() *AuthNMappingIncluded {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAuthNMappingIncluded) Set(val *AuthNMappingIncluded) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAuthNMappingIncluded) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableAuthNMappingIncluded) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAuthNMappingIncluded initializes the struct as if Set has been called.
func NewNullableAuthNMappingIncluded(val *AuthNMappingIncluded) *NullableAuthNMappingIncluded {
	return &NullableAuthNMappingIncluded{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAuthNMappingIncluded) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAuthNMappingIncluded) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
