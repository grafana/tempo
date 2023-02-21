// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionV1Contact Contact information about the service.
type ServiceDefinitionV1Contact struct {
	// Service owner’s email.
	Email *string `json:"email,omitempty"`
	// Service owner’s Slack channel.
	Slack *string `json:"slack,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV1Contact instantiates a new ServiceDefinitionV1Contact object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV1Contact() *ServiceDefinitionV1Contact {
	this := ServiceDefinitionV1Contact{}
	return &this
}

// NewServiceDefinitionV1ContactWithDefaults instantiates a new ServiceDefinitionV1Contact object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV1ContactWithDefaults() *ServiceDefinitionV1Contact {
	this := ServiceDefinitionV1Contact{}
	return &this
}

// GetEmail returns the Email field value if set, zero value otherwise.
func (o *ServiceDefinitionV1Contact) GetEmail() string {
	if o == nil || o.Email == nil {
		var ret string
		return ret
	}
	return *o.Email
}

// GetEmailOk returns a tuple with the Email field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Contact) GetEmailOk() (*string, bool) {
	if o == nil || o.Email == nil {
		return nil, false
	}
	return o.Email, true
}

// HasEmail returns a boolean if a field has been set.
func (o *ServiceDefinitionV1Contact) HasEmail() bool {
	return o != nil && o.Email != nil
}

// SetEmail gets a reference to the given string and assigns it to the Email field.
func (o *ServiceDefinitionV1Contact) SetEmail(v string) {
	o.Email = &v
}

// GetSlack returns the Slack field value if set, zero value otherwise.
func (o *ServiceDefinitionV1Contact) GetSlack() string {
	if o == nil || o.Slack == nil {
		var ret string
		return ret
	}
	return *o.Slack
}

// GetSlackOk returns a tuple with the Slack field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Contact) GetSlackOk() (*string, bool) {
	if o == nil || o.Slack == nil {
		return nil, false
	}
	return o.Slack, true
}

// HasSlack returns a boolean if a field has been set.
func (o *ServiceDefinitionV1Contact) HasSlack() bool {
	return o != nil && o.Slack != nil
}

// SetSlack gets a reference to the given string and assigns it to the Slack field.
func (o *ServiceDefinitionV1Contact) SetSlack(v string) {
	o.Slack = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV1Contact) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Email != nil {
		toSerialize["email"] = o.Email
	}
	if o.Slack != nil {
		toSerialize["slack"] = o.Slack
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionV1Contact) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Email *string `json:"email,omitempty"`
		Slack *string `json:"slack,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Email = all.Email
	o.Slack = all.Slack
	return nil
}
