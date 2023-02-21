// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionV2Contact - Service owner's contacts information.
type ServiceDefinitionV2Contact struct {
	ServiceDefinitionV2Email *ServiceDefinitionV2Email
	ServiceDefinitionV2Slack *ServiceDefinitionV2Slack

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// ServiceDefinitionV2EmailAsServiceDefinitionV2Contact is a convenience function that returns ServiceDefinitionV2Email wrapped in ServiceDefinitionV2Contact.
func ServiceDefinitionV2EmailAsServiceDefinitionV2Contact(v *ServiceDefinitionV2Email) ServiceDefinitionV2Contact {
	return ServiceDefinitionV2Contact{ServiceDefinitionV2Email: v}
}

// ServiceDefinitionV2SlackAsServiceDefinitionV2Contact is a convenience function that returns ServiceDefinitionV2Slack wrapped in ServiceDefinitionV2Contact.
func ServiceDefinitionV2SlackAsServiceDefinitionV2Contact(v *ServiceDefinitionV2Slack) ServiceDefinitionV2Contact {
	return ServiceDefinitionV2Contact{ServiceDefinitionV2Slack: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *ServiceDefinitionV2Contact) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into ServiceDefinitionV2Email
	err = json.Unmarshal(data, &obj.ServiceDefinitionV2Email)
	if err == nil {
		if obj.ServiceDefinitionV2Email != nil && obj.ServiceDefinitionV2Email.UnparsedObject == nil {
			jsonServiceDefinitionV2Email, _ := json.Marshal(obj.ServiceDefinitionV2Email)
			if string(jsonServiceDefinitionV2Email) == "{}" { // empty struct
				obj.ServiceDefinitionV2Email = nil
			} else {
				match++
			}
		} else {
			obj.ServiceDefinitionV2Email = nil
		}
	} else {
		obj.ServiceDefinitionV2Email = nil
	}

	// try to unmarshal data into ServiceDefinitionV2Slack
	err = json.Unmarshal(data, &obj.ServiceDefinitionV2Slack)
	if err == nil {
		if obj.ServiceDefinitionV2Slack != nil && obj.ServiceDefinitionV2Slack.UnparsedObject == nil {
			jsonServiceDefinitionV2Slack, _ := json.Marshal(obj.ServiceDefinitionV2Slack)
			if string(jsonServiceDefinitionV2Slack) == "{}" { // empty struct
				obj.ServiceDefinitionV2Slack = nil
			} else {
				match++
			}
		} else {
			obj.ServiceDefinitionV2Slack = nil
		}
	} else {
		obj.ServiceDefinitionV2Slack = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.ServiceDefinitionV2Email = nil
		obj.ServiceDefinitionV2Slack = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj ServiceDefinitionV2Contact) MarshalJSON() ([]byte, error) {
	if obj.ServiceDefinitionV2Email != nil {
		return json.Marshal(&obj.ServiceDefinitionV2Email)
	}

	if obj.ServiceDefinitionV2Slack != nil {
		return json.Marshal(&obj.ServiceDefinitionV2Slack)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *ServiceDefinitionV2Contact) GetActualInstance() interface{} {
	if obj.ServiceDefinitionV2Email != nil {
		return obj.ServiceDefinitionV2Email
	}

	if obj.ServiceDefinitionV2Slack != nil {
		return obj.ServiceDefinitionV2Slack
	}

	// all schemas are nil
	return nil
}

// NullableServiceDefinitionV2Contact handles when a null is used for ServiceDefinitionV2Contact.
type NullableServiceDefinitionV2Contact struct {
	value *ServiceDefinitionV2Contact
	isSet bool
}

// Get returns the associated value.
func (v NullableServiceDefinitionV2Contact) Get() *ServiceDefinitionV2Contact {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableServiceDefinitionV2Contact) Set(val *ServiceDefinitionV2Contact) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableServiceDefinitionV2Contact) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableServiceDefinitionV2Contact) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableServiceDefinitionV2Contact initializes the struct as if Set has been called.
func NewNullableServiceDefinitionV2Contact(val *ServiceDefinitionV2Contact) *NullableServiceDefinitionV2Contact {
	return &NullableServiceDefinitionV2Contact{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableServiceDefinitionV2Contact) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableServiceDefinitionV2Contact) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
