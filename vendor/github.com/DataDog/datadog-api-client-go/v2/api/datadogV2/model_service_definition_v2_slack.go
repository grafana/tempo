// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV2Slack Service owner's Slack channel.
type ServiceDefinitionV2Slack struct {
	// Slack Channel.
	Contact string `json:"contact"`
	// Contact Slack.
	Name *string `json:"name,omitempty"`
	// Contact type.
	Type ServiceDefinitionV2SlackType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV2Slack instantiates a new ServiceDefinitionV2Slack object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV2Slack(contact string, typeVar ServiceDefinitionV2SlackType) *ServiceDefinitionV2Slack {
	this := ServiceDefinitionV2Slack{}
	this.Contact = contact
	this.Type = typeVar
	return &this
}

// NewServiceDefinitionV2SlackWithDefaults instantiates a new ServiceDefinitionV2Slack object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV2SlackWithDefaults() *ServiceDefinitionV2Slack {
	this := ServiceDefinitionV2Slack{}
	return &this
}

// GetContact returns the Contact field value.
func (o *ServiceDefinitionV2Slack) GetContact() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Contact
}

// GetContactOk returns a tuple with the Contact field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Slack) GetContactOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Contact, true
}

// SetContact sets field value.
func (o *ServiceDefinitionV2Slack) SetContact(v string) {
	o.Contact = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *ServiceDefinitionV2Slack) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Slack) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *ServiceDefinitionV2Slack) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *ServiceDefinitionV2Slack) SetName(v string) {
	o.Name = &v
}

// GetType returns the Type field value.
func (o *ServiceDefinitionV2Slack) GetType() ServiceDefinitionV2SlackType {
	if o == nil {
		var ret ServiceDefinitionV2SlackType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Slack) GetTypeOk() (*ServiceDefinitionV2SlackType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ServiceDefinitionV2Slack) SetType(v ServiceDefinitionV2SlackType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV2Slack) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["contact"] = o.Contact
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionV2Slack) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Contact *string                       `json:"contact"`
		Type    *ServiceDefinitionV2SlackType `json:"type"`
	}{}
	all := struct {
		Contact string                       `json:"contact"`
		Name    *string                      `json:"name,omitempty"`
		Type    ServiceDefinitionV2SlackType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Contact == nil {
		return fmt.Errorf("required field contact missing")
	}
	if required.Type == nil {
		return fmt.Errorf("required field type missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Contact = all.Contact
	o.Name = all.Name
	o.Type = all.Type
	return nil
}
