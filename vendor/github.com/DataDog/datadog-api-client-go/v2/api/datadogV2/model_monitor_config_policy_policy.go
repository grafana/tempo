// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MonitorConfigPolicyPolicy - Configuration for the policy.
type MonitorConfigPolicyPolicy struct {
	MonitorConfigPolicyTagPolicy *MonitorConfigPolicyTagPolicy

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// MonitorConfigPolicyTagPolicyAsMonitorConfigPolicyPolicy is a convenience function that returns MonitorConfigPolicyTagPolicy wrapped in MonitorConfigPolicyPolicy.
func MonitorConfigPolicyTagPolicyAsMonitorConfigPolicyPolicy(v *MonitorConfigPolicyTagPolicy) MonitorConfigPolicyPolicy {
	return MonitorConfigPolicyPolicy{MonitorConfigPolicyTagPolicy: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *MonitorConfigPolicyPolicy) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into MonitorConfigPolicyTagPolicy
	err = json.Unmarshal(data, &obj.MonitorConfigPolicyTagPolicy)
	if err == nil {
		if obj.MonitorConfigPolicyTagPolicy != nil && obj.MonitorConfigPolicyTagPolicy.UnparsedObject == nil {
			jsonMonitorConfigPolicyTagPolicy, _ := json.Marshal(obj.MonitorConfigPolicyTagPolicy)
			if string(jsonMonitorConfigPolicyTagPolicy) == "{}" { // empty struct
				obj.MonitorConfigPolicyTagPolicy = nil
			} else {
				match++
			}
		} else {
			obj.MonitorConfigPolicyTagPolicy = nil
		}
	} else {
		obj.MonitorConfigPolicyTagPolicy = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.MonitorConfigPolicyTagPolicy = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj MonitorConfigPolicyPolicy) MarshalJSON() ([]byte, error) {
	if obj.MonitorConfigPolicyTagPolicy != nil {
		return json.Marshal(&obj.MonitorConfigPolicyTagPolicy)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *MonitorConfigPolicyPolicy) GetActualInstance() interface{} {
	if obj.MonitorConfigPolicyTagPolicy != nil {
		return obj.MonitorConfigPolicyTagPolicy
	}

	// all schemas are nil
	return nil
}

// NullableMonitorConfigPolicyPolicy handles when a null is used for MonitorConfigPolicyPolicy.
type NullableMonitorConfigPolicyPolicy struct {
	value *MonitorConfigPolicyPolicy
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorConfigPolicyPolicy) Get() *MonitorConfigPolicyPolicy {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorConfigPolicyPolicy) Set(val *MonitorConfigPolicyPolicy) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorConfigPolicyPolicy) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableMonitorConfigPolicyPolicy) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorConfigPolicyPolicy initializes the struct as if Set has been called.
func NewNullableMonitorConfigPolicyPolicy(val *MonitorConfigPolicyPolicy) *NullableMonitorConfigPolicyPolicy {
	return &NullableMonitorConfigPolicyPolicy{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorConfigPolicyPolicy) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorConfigPolicyPolicy) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
