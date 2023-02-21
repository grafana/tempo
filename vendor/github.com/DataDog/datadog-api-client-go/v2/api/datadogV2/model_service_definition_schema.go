// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionSchema - Service definition schema.
type ServiceDefinitionSchema struct {
	ServiceDefinitionV1 *ServiceDefinitionV1
	ServiceDefinitionV2 *ServiceDefinitionV2

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// ServiceDefinitionV1AsServiceDefinitionSchema is a convenience function that returns ServiceDefinitionV1 wrapped in ServiceDefinitionSchema.
func ServiceDefinitionV1AsServiceDefinitionSchema(v *ServiceDefinitionV1) ServiceDefinitionSchema {
	return ServiceDefinitionSchema{ServiceDefinitionV1: v}
}

// ServiceDefinitionV2AsServiceDefinitionSchema is a convenience function that returns ServiceDefinitionV2 wrapped in ServiceDefinitionSchema.
func ServiceDefinitionV2AsServiceDefinitionSchema(v *ServiceDefinitionV2) ServiceDefinitionSchema {
	return ServiceDefinitionSchema{ServiceDefinitionV2: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *ServiceDefinitionSchema) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into ServiceDefinitionV1
	err = json.Unmarshal(data, &obj.ServiceDefinitionV1)
	if err == nil {
		if obj.ServiceDefinitionV1 != nil && obj.ServiceDefinitionV1.UnparsedObject == nil {
			jsonServiceDefinitionV1, _ := json.Marshal(obj.ServiceDefinitionV1)
			if string(jsonServiceDefinitionV1) == "{}" { // empty struct
				obj.ServiceDefinitionV1 = nil
			} else {
				match++
			}
		} else {
			obj.ServiceDefinitionV1 = nil
		}
	} else {
		obj.ServiceDefinitionV1 = nil
	}

	// try to unmarshal data into ServiceDefinitionV2
	err = json.Unmarshal(data, &obj.ServiceDefinitionV2)
	if err == nil {
		if obj.ServiceDefinitionV2 != nil && obj.ServiceDefinitionV2.UnparsedObject == nil {
			jsonServiceDefinitionV2, _ := json.Marshal(obj.ServiceDefinitionV2)
			if string(jsonServiceDefinitionV2) == "{}" { // empty struct
				obj.ServiceDefinitionV2 = nil
			} else {
				match++
			}
		} else {
			obj.ServiceDefinitionV2 = nil
		}
	} else {
		obj.ServiceDefinitionV2 = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.ServiceDefinitionV1 = nil
		obj.ServiceDefinitionV2 = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj ServiceDefinitionSchema) MarshalJSON() ([]byte, error) {
	if obj.ServiceDefinitionV1 != nil {
		return json.Marshal(&obj.ServiceDefinitionV1)
	}

	if obj.ServiceDefinitionV2 != nil {
		return json.Marshal(&obj.ServiceDefinitionV2)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *ServiceDefinitionSchema) GetActualInstance() interface{} {
	if obj.ServiceDefinitionV1 != nil {
		return obj.ServiceDefinitionV1
	}

	if obj.ServiceDefinitionV2 != nil {
		return obj.ServiceDefinitionV2
	}

	// all schemas are nil
	return nil
}

// NullableServiceDefinitionSchema handles when a null is used for ServiceDefinitionSchema.
type NullableServiceDefinitionSchema struct {
	value *ServiceDefinitionSchema
	isSet bool
}

// Get returns the associated value.
func (v NullableServiceDefinitionSchema) Get() *ServiceDefinitionSchema {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableServiceDefinitionSchema) Set(val *ServiceDefinitionSchema) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableServiceDefinitionSchema) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableServiceDefinitionSchema) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableServiceDefinitionSchema initializes the struct as if Set has been called.
func NewNullableServiceDefinitionSchema(val *ServiceDefinitionSchema) *NullableServiceDefinitionSchema {
	return &NullableServiceDefinitionSchema{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableServiceDefinitionSchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableServiceDefinitionSchema) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
