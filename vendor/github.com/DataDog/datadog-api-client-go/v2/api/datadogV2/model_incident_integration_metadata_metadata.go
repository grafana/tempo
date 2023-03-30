// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentIntegrationMetadataMetadata - Incident integration metadata's metadata attribute.
type IncidentIntegrationMetadataMetadata struct {
	SlackIntegrationMetadata *SlackIntegrationMetadata
	JiraIntegrationMetadata  *JiraIntegrationMetadata

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SlackIntegrationMetadataAsIncidentIntegrationMetadataMetadata is a convenience function that returns SlackIntegrationMetadata wrapped in IncidentIntegrationMetadataMetadata.
func SlackIntegrationMetadataAsIncidentIntegrationMetadataMetadata(v *SlackIntegrationMetadata) IncidentIntegrationMetadataMetadata {
	return IncidentIntegrationMetadataMetadata{SlackIntegrationMetadata: v}
}

// JiraIntegrationMetadataAsIncidentIntegrationMetadataMetadata is a convenience function that returns JiraIntegrationMetadata wrapped in IncidentIntegrationMetadataMetadata.
func JiraIntegrationMetadataAsIncidentIntegrationMetadataMetadata(v *JiraIntegrationMetadata) IncidentIntegrationMetadataMetadata {
	return IncidentIntegrationMetadataMetadata{JiraIntegrationMetadata: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *IncidentIntegrationMetadataMetadata) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SlackIntegrationMetadata
	err = json.Unmarshal(data, &obj.SlackIntegrationMetadata)
	if err == nil {
		if obj.SlackIntegrationMetadata != nil && obj.SlackIntegrationMetadata.UnparsedObject == nil {
			jsonSlackIntegrationMetadata, _ := json.Marshal(obj.SlackIntegrationMetadata)
			if string(jsonSlackIntegrationMetadata) == "{}" { // empty struct
				obj.SlackIntegrationMetadata = nil
			} else {
				match++
			}
		} else {
			obj.SlackIntegrationMetadata = nil
		}
	} else {
		obj.SlackIntegrationMetadata = nil
	}

	// try to unmarshal data into JiraIntegrationMetadata
	err = json.Unmarshal(data, &obj.JiraIntegrationMetadata)
	if err == nil {
		if obj.JiraIntegrationMetadata != nil && obj.JiraIntegrationMetadata.UnparsedObject == nil {
			jsonJiraIntegrationMetadata, _ := json.Marshal(obj.JiraIntegrationMetadata)
			if string(jsonJiraIntegrationMetadata) == "{}" { // empty struct
				obj.JiraIntegrationMetadata = nil
			} else {
				match++
			}
		} else {
			obj.JiraIntegrationMetadata = nil
		}
	} else {
		obj.JiraIntegrationMetadata = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.SlackIntegrationMetadata = nil
		obj.JiraIntegrationMetadata = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj IncidentIntegrationMetadataMetadata) MarshalJSON() ([]byte, error) {
	if obj.SlackIntegrationMetadata != nil {
		return json.Marshal(&obj.SlackIntegrationMetadata)
	}

	if obj.JiraIntegrationMetadata != nil {
		return json.Marshal(&obj.JiraIntegrationMetadata)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *IncidentIntegrationMetadataMetadata) GetActualInstance() interface{} {
	if obj.SlackIntegrationMetadata != nil {
		return obj.SlackIntegrationMetadata
	}

	if obj.JiraIntegrationMetadata != nil {
		return obj.JiraIntegrationMetadata
	}

	// all schemas are nil
	return nil
}

// NullableIncidentIntegrationMetadataMetadata handles when a null is used for IncidentIntegrationMetadataMetadata.
type NullableIncidentIntegrationMetadataMetadata struct {
	value *IncidentIntegrationMetadataMetadata
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentIntegrationMetadataMetadata) Get() *IncidentIntegrationMetadataMetadata {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentIntegrationMetadataMetadata) Set(val *IncidentIntegrationMetadataMetadata) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentIntegrationMetadataMetadata) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableIncidentIntegrationMetadataMetadata) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentIntegrationMetadataMetadata initializes the struct as if Set has been called.
func NewNullableIncidentIntegrationMetadataMetadata(val *IncidentIntegrationMetadataMetadata) *NullableIncidentIntegrationMetadataMetadata {
	return &NullableIncidentIntegrationMetadataMetadata{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentIntegrationMetadataMetadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentIntegrationMetadataMetadata) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
