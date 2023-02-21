// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionsCreateRequest - Create service definitions request.
type ServiceDefinitionsCreateRequest struct {
	ServiceDefinitionV2  *ServiceDefinitionV2
	ServiceDefinitionRaw *string

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// ServiceDefinitionV2AsServiceDefinitionsCreateRequest is a convenience function that returns ServiceDefinitionV2 wrapped in ServiceDefinitionsCreateRequest.
func ServiceDefinitionV2AsServiceDefinitionsCreateRequest(v *ServiceDefinitionV2) ServiceDefinitionsCreateRequest {
	return ServiceDefinitionsCreateRequest{ServiceDefinitionV2: v}
}

// ServiceDefinitionRawAsServiceDefinitionsCreateRequest is a convenience function that returns string wrapped in ServiceDefinitionsCreateRequest.
func ServiceDefinitionRawAsServiceDefinitionsCreateRequest(v *string) ServiceDefinitionsCreateRequest {
	return ServiceDefinitionsCreateRequest{ServiceDefinitionRaw: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *ServiceDefinitionsCreateRequest) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
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

	// try to unmarshal data into ServiceDefinitionRaw
	err = json.Unmarshal(data, &obj.ServiceDefinitionRaw)
	if err == nil {
		if obj.ServiceDefinitionRaw != nil {
			jsonServiceDefinitionRaw, _ := json.Marshal(obj.ServiceDefinitionRaw)
			if string(jsonServiceDefinitionRaw) == "{}" { // empty struct
				obj.ServiceDefinitionRaw = nil
			} else {
				match++
			}
		} else {
			obj.ServiceDefinitionRaw = nil
		}
	} else {
		obj.ServiceDefinitionRaw = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.ServiceDefinitionV2 = nil
		obj.ServiceDefinitionRaw = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj ServiceDefinitionsCreateRequest) MarshalJSON() ([]byte, error) {
	if obj.ServiceDefinitionV2 != nil {
		return json.Marshal(&obj.ServiceDefinitionV2)
	}

	if obj.ServiceDefinitionRaw != nil {
		return json.Marshal(&obj.ServiceDefinitionRaw)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *ServiceDefinitionsCreateRequest) GetActualInstance() interface{} {
	if obj.ServiceDefinitionV2 != nil {
		return obj.ServiceDefinitionV2
	}

	if obj.ServiceDefinitionRaw != nil {
		return obj.ServiceDefinitionRaw
	}

	// all schemas are nil
	return nil
}

// NullableServiceDefinitionsCreateRequest handles when a null is used for ServiceDefinitionsCreateRequest.
type NullableServiceDefinitionsCreateRequest struct {
	value *ServiceDefinitionsCreateRequest
	isSet bool
}

// Get returns the associated value.
func (v NullableServiceDefinitionsCreateRequest) Get() *ServiceDefinitionsCreateRequest {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableServiceDefinitionsCreateRequest) Set(val *ServiceDefinitionsCreateRequest) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableServiceDefinitionsCreateRequest) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableServiceDefinitionsCreateRequest) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableServiceDefinitionsCreateRequest initializes the struct as if Set has been called.
func NewNullableServiceDefinitionsCreateRequest(val *ServiceDefinitionsCreateRequest) *NullableServiceDefinitionsCreateRequest {
	return &NullableServiceDefinitionsCreateRequest{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableServiceDefinitionsCreateRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableServiceDefinitionsCreateRequest) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
