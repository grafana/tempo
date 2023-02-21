// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringRuleCreatePayload - Create a new rule.
type SecurityMonitoringRuleCreatePayload struct {
	SecurityMonitoringStandardRuleCreatePayload *SecurityMonitoringStandardRuleCreatePayload
	SecurityMonitoringSignalRuleCreatePayload   *SecurityMonitoringSignalRuleCreatePayload
	CloudConfigurationRuleCreatePayload         *CloudConfigurationRuleCreatePayload

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SecurityMonitoringStandardRuleCreatePayloadAsSecurityMonitoringRuleCreatePayload is a convenience function that returns SecurityMonitoringStandardRuleCreatePayload wrapped in SecurityMonitoringRuleCreatePayload.
func SecurityMonitoringStandardRuleCreatePayloadAsSecurityMonitoringRuleCreatePayload(v *SecurityMonitoringStandardRuleCreatePayload) SecurityMonitoringRuleCreatePayload {
	return SecurityMonitoringRuleCreatePayload{SecurityMonitoringStandardRuleCreatePayload: v}
}

// SecurityMonitoringSignalRuleCreatePayloadAsSecurityMonitoringRuleCreatePayload is a convenience function that returns SecurityMonitoringSignalRuleCreatePayload wrapped in SecurityMonitoringRuleCreatePayload.
func SecurityMonitoringSignalRuleCreatePayloadAsSecurityMonitoringRuleCreatePayload(v *SecurityMonitoringSignalRuleCreatePayload) SecurityMonitoringRuleCreatePayload {
	return SecurityMonitoringRuleCreatePayload{SecurityMonitoringSignalRuleCreatePayload: v}
}

// CloudConfigurationRuleCreatePayloadAsSecurityMonitoringRuleCreatePayload is a convenience function that returns CloudConfigurationRuleCreatePayload wrapped in SecurityMonitoringRuleCreatePayload.
func CloudConfigurationRuleCreatePayloadAsSecurityMonitoringRuleCreatePayload(v *CloudConfigurationRuleCreatePayload) SecurityMonitoringRuleCreatePayload {
	return SecurityMonitoringRuleCreatePayload{CloudConfigurationRuleCreatePayload: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *SecurityMonitoringRuleCreatePayload) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SecurityMonitoringStandardRuleCreatePayload
	err = json.Unmarshal(data, &obj.SecurityMonitoringStandardRuleCreatePayload)
	if err == nil {
		if obj.SecurityMonitoringStandardRuleCreatePayload != nil && obj.SecurityMonitoringStandardRuleCreatePayload.UnparsedObject == nil {
			jsonSecurityMonitoringStandardRuleCreatePayload, _ := json.Marshal(obj.SecurityMonitoringStandardRuleCreatePayload)
			if string(jsonSecurityMonitoringStandardRuleCreatePayload) == "{}" { // empty struct
				obj.SecurityMonitoringStandardRuleCreatePayload = nil
			} else {
				match++
			}
		} else {
			obj.SecurityMonitoringStandardRuleCreatePayload = nil
		}
	} else {
		obj.SecurityMonitoringStandardRuleCreatePayload = nil
	}

	// try to unmarshal data into SecurityMonitoringSignalRuleCreatePayload
	err = json.Unmarshal(data, &obj.SecurityMonitoringSignalRuleCreatePayload)
	if err == nil {
		if obj.SecurityMonitoringSignalRuleCreatePayload != nil && obj.SecurityMonitoringSignalRuleCreatePayload.UnparsedObject == nil {
			jsonSecurityMonitoringSignalRuleCreatePayload, _ := json.Marshal(obj.SecurityMonitoringSignalRuleCreatePayload)
			if string(jsonSecurityMonitoringSignalRuleCreatePayload) == "{}" { // empty struct
				obj.SecurityMonitoringSignalRuleCreatePayload = nil
			} else {
				match++
			}
		} else {
			obj.SecurityMonitoringSignalRuleCreatePayload = nil
		}
	} else {
		obj.SecurityMonitoringSignalRuleCreatePayload = nil
	}

	// try to unmarshal data into CloudConfigurationRuleCreatePayload
	err = json.Unmarshal(data, &obj.CloudConfigurationRuleCreatePayload)
	if err == nil {
		if obj.CloudConfigurationRuleCreatePayload != nil && obj.CloudConfigurationRuleCreatePayload.UnparsedObject == nil {
			jsonCloudConfigurationRuleCreatePayload, _ := json.Marshal(obj.CloudConfigurationRuleCreatePayload)
			if string(jsonCloudConfigurationRuleCreatePayload) == "{}" { // empty struct
				obj.CloudConfigurationRuleCreatePayload = nil
			} else {
				match++
			}
		} else {
			obj.CloudConfigurationRuleCreatePayload = nil
		}
	} else {
		obj.CloudConfigurationRuleCreatePayload = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.SecurityMonitoringStandardRuleCreatePayload = nil
		obj.SecurityMonitoringSignalRuleCreatePayload = nil
		obj.CloudConfigurationRuleCreatePayload = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj SecurityMonitoringRuleCreatePayload) MarshalJSON() ([]byte, error) {
	if obj.SecurityMonitoringStandardRuleCreatePayload != nil {
		return json.Marshal(&obj.SecurityMonitoringStandardRuleCreatePayload)
	}

	if obj.SecurityMonitoringSignalRuleCreatePayload != nil {
		return json.Marshal(&obj.SecurityMonitoringSignalRuleCreatePayload)
	}

	if obj.CloudConfigurationRuleCreatePayload != nil {
		return json.Marshal(&obj.CloudConfigurationRuleCreatePayload)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *SecurityMonitoringRuleCreatePayload) GetActualInstance() interface{} {
	if obj.SecurityMonitoringStandardRuleCreatePayload != nil {
		return obj.SecurityMonitoringStandardRuleCreatePayload
	}

	if obj.SecurityMonitoringSignalRuleCreatePayload != nil {
		return obj.SecurityMonitoringSignalRuleCreatePayload
	}

	if obj.CloudConfigurationRuleCreatePayload != nil {
		return obj.CloudConfigurationRuleCreatePayload
	}

	// all schemas are nil
	return nil
}

// NullableSecurityMonitoringRuleCreatePayload handles when a null is used for SecurityMonitoringRuleCreatePayload.
type NullableSecurityMonitoringRuleCreatePayload struct {
	value *SecurityMonitoringRuleCreatePayload
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleCreatePayload) Get() *SecurityMonitoringRuleCreatePayload {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleCreatePayload) Set(val *SecurityMonitoringRuleCreatePayload) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleCreatePayload) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSecurityMonitoringRuleCreatePayload) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleCreatePayload initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleCreatePayload(val *SecurityMonitoringRuleCreatePayload) *NullableSecurityMonitoringRuleCreatePayload {
	return &NullableSecurityMonitoringRuleCreatePayload{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleCreatePayload) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleCreatePayload) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
