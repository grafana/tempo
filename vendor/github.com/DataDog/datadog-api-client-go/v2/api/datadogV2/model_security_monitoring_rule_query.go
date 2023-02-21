// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringRuleQuery - Query for matching rule.
type SecurityMonitoringRuleQuery struct {
	SecurityMonitoringStandardRuleQuery *SecurityMonitoringStandardRuleQuery
	SecurityMonitoringSignalRuleQuery   *SecurityMonitoringSignalRuleQuery

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SecurityMonitoringStandardRuleQueryAsSecurityMonitoringRuleQuery is a convenience function that returns SecurityMonitoringStandardRuleQuery wrapped in SecurityMonitoringRuleQuery.
func SecurityMonitoringStandardRuleQueryAsSecurityMonitoringRuleQuery(v *SecurityMonitoringStandardRuleQuery) SecurityMonitoringRuleQuery {
	return SecurityMonitoringRuleQuery{SecurityMonitoringStandardRuleQuery: v}
}

// SecurityMonitoringSignalRuleQueryAsSecurityMonitoringRuleQuery is a convenience function that returns SecurityMonitoringSignalRuleQuery wrapped in SecurityMonitoringRuleQuery.
func SecurityMonitoringSignalRuleQueryAsSecurityMonitoringRuleQuery(v *SecurityMonitoringSignalRuleQuery) SecurityMonitoringRuleQuery {
	return SecurityMonitoringRuleQuery{SecurityMonitoringSignalRuleQuery: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *SecurityMonitoringRuleQuery) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SecurityMonitoringStandardRuleQuery
	err = json.Unmarshal(data, &obj.SecurityMonitoringStandardRuleQuery)
	if err == nil {
		if obj.SecurityMonitoringStandardRuleQuery != nil && obj.SecurityMonitoringStandardRuleQuery.UnparsedObject == nil {
			jsonSecurityMonitoringStandardRuleQuery, _ := json.Marshal(obj.SecurityMonitoringStandardRuleQuery)
			if string(jsonSecurityMonitoringStandardRuleQuery) == "{}" { // empty struct
				obj.SecurityMonitoringStandardRuleQuery = nil
			} else {
				match++
			}
		} else {
			obj.SecurityMonitoringStandardRuleQuery = nil
		}
	} else {
		obj.SecurityMonitoringStandardRuleQuery = nil
	}

	// try to unmarshal data into SecurityMonitoringSignalRuleQuery
	err = json.Unmarshal(data, &obj.SecurityMonitoringSignalRuleQuery)
	if err == nil {
		if obj.SecurityMonitoringSignalRuleQuery != nil && obj.SecurityMonitoringSignalRuleQuery.UnparsedObject == nil {
			jsonSecurityMonitoringSignalRuleQuery, _ := json.Marshal(obj.SecurityMonitoringSignalRuleQuery)
			if string(jsonSecurityMonitoringSignalRuleQuery) == "{}" { // empty struct
				obj.SecurityMonitoringSignalRuleQuery = nil
			} else {
				match++
			}
		} else {
			obj.SecurityMonitoringSignalRuleQuery = nil
		}
	} else {
		obj.SecurityMonitoringSignalRuleQuery = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.SecurityMonitoringStandardRuleQuery = nil
		obj.SecurityMonitoringSignalRuleQuery = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj SecurityMonitoringRuleQuery) MarshalJSON() ([]byte, error) {
	if obj.SecurityMonitoringStandardRuleQuery != nil {
		return json.Marshal(&obj.SecurityMonitoringStandardRuleQuery)
	}

	if obj.SecurityMonitoringSignalRuleQuery != nil {
		return json.Marshal(&obj.SecurityMonitoringSignalRuleQuery)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *SecurityMonitoringRuleQuery) GetActualInstance() interface{} {
	if obj.SecurityMonitoringStandardRuleQuery != nil {
		return obj.SecurityMonitoringStandardRuleQuery
	}

	if obj.SecurityMonitoringSignalRuleQuery != nil {
		return obj.SecurityMonitoringSignalRuleQuery
	}

	// all schemas are nil
	return nil
}

// NullableSecurityMonitoringRuleQuery handles when a null is used for SecurityMonitoringRuleQuery.
type NullableSecurityMonitoringRuleQuery struct {
	value *SecurityMonitoringRuleQuery
	isSet bool
}

// Get returns the associated value.
func (v NullableSecurityMonitoringRuleQuery) Get() *SecurityMonitoringRuleQuery {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSecurityMonitoringRuleQuery) Set(val *SecurityMonitoringRuleQuery) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSecurityMonitoringRuleQuery) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSecurityMonitoringRuleQuery) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSecurityMonitoringRuleQuery initializes the struct as if Set has been called.
func NewNullableSecurityMonitoringRuleQuery(val *SecurityMonitoringRuleQuery) *NullableSecurityMonitoringRuleQuery {
	return &NullableSecurityMonitoringRuleQuery{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSecurityMonitoringRuleQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSecurityMonitoringRuleQuery) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
