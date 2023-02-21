// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringRuleResponse - Create a new rule.
type SecurityMonitoringRuleResponse struct {
	SecurityMonitoringStandardRuleResponse *SecurityMonitoringStandardRuleResponse
	SecurityMonitoringSignalRuleResponse   *SecurityMonitoringSignalRuleResponse

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SecurityMonitoringStandardRuleResponseAsSecurityMonitoringRuleResponse is a convenience function that returns SecurityMonitoringStandardRuleResponse wrapped in SecurityMonitoringRuleResponse.
func SecurityMonitoringStandardRuleResponseAsSecurityMonitoringRuleResponse(v *SecurityMonitoringStandardRuleResponse) SecurityMonitoringRuleResponse {
	return SecurityMonitoringRuleResponse{SecurityMonitoringStandardRuleResponse: v}
}

// SecurityMonitoringSignalRuleResponseAsSecurityMonitoringRuleResponse is a convenience function that returns SecurityMonitoringSignalRuleResponse wrapped in SecurityMonitoringRuleResponse.
func SecurityMonitoringSignalRuleResponseAsSecurityMonitoringRuleResponse(v *SecurityMonitoringSignalRuleResponse) SecurityMonitoringRuleResponse {
	return SecurityMonitoringRuleResponse{SecurityMonitoringSignalRuleResponse: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *SecurityMonitoringRuleResponse) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SecurityMonitoringStandardRuleResponse
	err = json.Unmarshal(data, &obj.SecurityMonitoringStandardRuleResponse)
	if err == nil {
		if obj.SecurityMonitoringStandardRuleResponse != nil && obj.SecurityMonitoringStandardRuleResponse.UnparsedObject == nil {
			jsonSecurityMonitoringStandardRuleResponse, _ := json.Marshal(obj.SecurityMonitoringStandardRuleResponse)
			if string(jsonSecurityMonitoringStandardRuleResponse) == "{}" { // empty struct
				obj.SecurityMonitoringStandardRuleResponse = nil
			} else {
				match++
			}
		} else {
			obj.SecurityMonitoringStandardRuleResponse = nil
		}
	} else {
		obj.SecurityMonitoringStandardRuleResponse = nil
	}

	// try to unmarshal data into SecurityMonitoringSignalRuleResponse
	err = json.Unmarshal(data, &obj.SecurityMonitoringSignalRuleResponse)
	if err == nil {
		if obj.SecurityMonitoringSignalRuleResponse != nil && obj.SecurityMonitoringSignalRuleResponse.UnparsedObject == nil {
			jsonSecurityMonitoringSignalRuleResponse, _ := json.Marshal(obj.SecurityMonitoringSignalRuleResponse)
			if string(jsonSecurityMonitoringSignalRuleResponse) == "{}" { // empty struct
				obj.SecurityMonitoringSignalRuleResponse = nil
			} else {
				match++
			}
		} else {
			obj.SecurityMonitoringSignalRuleResponse = nil
		}
	} else {
		obj.SecurityMonitoringSignalRuleResponse = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.SecurityMonitoringStandardRuleResponse = nil
		obj.SecurityMonitoringSignalRuleResponse = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj SecurityMonitoringRuleResponse) MarshalJSON() ([]byte, error) {
	if obj.SecurityMonitoringStandardRuleResponse != nil {
		return json.Marshal(&obj.SecurityMonitoringStandardRuleResponse)
	}

	if obj.SecurityMonitoringSignalRuleResponse != nil {
		return json.Marshal(&obj.SecurityMonitoringSignalRuleResponse)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *SecurityMonitoringRuleResponse) GetActualInstance() interface{} {
	if obj.SecurityMonitoringStandardRuleResponse != nil {
		return obj.SecurityMonitoringStandardRuleResponse
	}

	if obj.SecurityMonitoringSignalRuleResponse != nil {
		return obj.SecurityMonitoringSignalRuleResponse
	}

	// all schemas are nil
	return nil
}

// NullableSecurityMonitoringRuleResponse handles when a null is used for SecurityMonitoringRuleResponse.
type NullableSecurityMonitoringRuleResponse struct {
	value *SecurityMonitoringRuleResponse
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleResponse) Get() *SecurityMonitoringRuleResponse {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleResponse) Set(val *SecurityMonitoringRuleResponse) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleResponse) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSecurityMonitoringRuleResponse) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleResponse initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleResponse(val *SecurityMonitoringRuleResponse) *NullableSecurityMonitoringRuleResponse {
	return &NullableSecurityMonitoringRuleResponse{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
