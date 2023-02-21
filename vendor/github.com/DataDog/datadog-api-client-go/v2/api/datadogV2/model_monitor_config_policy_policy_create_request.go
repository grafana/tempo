// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MonitorConfigPolicyPolicyCreateRequest - Configuration for the policy.
type MonitorConfigPolicyPolicyCreateRequest struct {
	MonitorConfigPolicyTagPolicyCreateRequest *MonitorConfigPolicyTagPolicyCreateRequest

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// MonitorConfigPolicyTagPolicyCreateRequestAsMonitorConfigPolicyPolicyCreateRequest is a convenience function that returns MonitorConfigPolicyTagPolicyCreateRequest wrapped in MonitorConfigPolicyPolicyCreateRequest.
func MonitorConfigPolicyTagPolicyCreateRequestAsMonitorConfigPolicyPolicyCreateRequest(v *MonitorConfigPolicyTagPolicyCreateRequest) MonitorConfigPolicyPolicyCreateRequest {
	return MonitorConfigPolicyPolicyCreateRequest{MonitorConfigPolicyTagPolicyCreateRequest: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *MonitorConfigPolicyPolicyCreateRequest) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into MonitorConfigPolicyTagPolicyCreateRequest
	err = json.Unmarshal(data, &obj.MonitorConfigPolicyTagPolicyCreateRequest)
	if err == nil {
		if obj.MonitorConfigPolicyTagPolicyCreateRequest != nil && obj.MonitorConfigPolicyTagPolicyCreateRequest.UnparsedObject == nil {
			jsonMonitorConfigPolicyTagPolicyCreateRequest, _ := json.Marshal(obj.MonitorConfigPolicyTagPolicyCreateRequest)
			if string(jsonMonitorConfigPolicyTagPolicyCreateRequest) == "{}" { // empty struct
				obj.MonitorConfigPolicyTagPolicyCreateRequest = nil
			} else {
				match++
			}
		} else {
			obj.MonitorConfigPolicyTagPolicyCreateRequest = nil
		}
	} else {
		obj.MonitorConfigPolicyTagPolicyCreateRequest = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.MonitorConfigPolicyTagPolicyCreateRequest = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj MonitorConfigPolicyPolicyCreateRequest) MarshalJSON() ([]byte, error) {
	if obj.MonitorConfigPolicyTagPolicyCreateRequest != nil {
		return json.Marshal(&obj.MonitorConfigPolicyTagPolicyCreateRequest)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *MonitorConfigPolicyPolicyCreateRequest) GetActualInstance() interface{} {
	if obj.MonitorConfigPolicyTagPolicyCreateRequest != nil {
		return obj.MonitorConfigPolicyTagPolicyCreateRequest
	}

	// all schemas are nil
	return nil
}

// NullableMonitorConfigPolicyPolicyCreateRequest handles when a null is used for MonitorConfigPolicyPolicyCreateRequest.
type NullableMonitorConfigPolicyPolicyCreateRequest struct {
	value *MonitorConfigPolicyPolicyCreateRequest
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorConfigPolicyPolicyCreateRequest) Get() *MonitorConfigPolicyPolicyCreateRequest {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorConfigPolicyPolicyCreateRequest) Set(val *MonitorConfigPolicyPolicyCreateRequest) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorConfigPolicyPolicyCreateRequest) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableMonitorConfigPolicyPolicyCreateRequest) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorConfigPolicyPolicyCreateRequest initializes the struct as if Set has been called.
func NewNullableMonitorConfigPolicyPolicyCreateRequest(val *MonitorConfigPolicyPolicyCreateRequest) *NullableMonitorConfigPolicyPolicyCreateRequest {
	return &NullableMonitorConfigPolicyPolicyCreateRequest{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorConfigPolicyPolicyCreateRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorConfigPolicyPolicyCreateRequest) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
