// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV2Email Service owner's email.
type ServiceDefinitionV2Email struct {
	// Contact value.
	Contact string `json:"contact"`
	// Contact email.
	Name *string `json:"name,omitempty"`
	// Contact type.
	Type ServiceDefinitionV2EmailType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV2Email instantiates a new ServiceDefinitionV2Email object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV2Email(contact string, typeVar ServiceDefinitionV2EmailType) *ServiceDefinitionV2Email {
	this := ServiceDefinitionV2Email{}
	this.Contact = contact
	this.Type = typeVar
	return &this
}

// NewServiceDefinitionV2EmailWithDefaults instantiates a new ServiceDefinitionV2Email object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV2EmailWithDefaults() *ServiceDefinitionV2Email {
	this := ServiceDefinitionV2Email{}
	return &this
}

// GetContact returns the Contact field value.
func (o *ServiceDefinitionV2Email) GetContact() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Contact
}

// GetContactOk returns a tuple with the Contact field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Email) GetContactOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Contact, true
}

// SetContact sets field value.
func (o *ServiceDefinitionV2Email) SetContact(v string) {
	o.Contact = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *ServiceDefinitionV2Email) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Email) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *ServiceDefinitionV2Email) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *ServiceDefinitionV2Email) SetName(v string) {
	o.Name = &v
}

// GetType returns the Type field value.
func (o *ServiceDefinitionV2Email) GetType() ServiceDefinitionV2EmailType {
	if o == nil {
		var ret ServiceDefinitionV2EmailType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Email) GetTypeOk() (*ServiceDefinitionV2EmailType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ServiceDefinitionV2Email) SetType(v ServiceDefinitionV2EmailType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV2Email) MarshalJSON() ([]byte, error) {
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
func (o *ServiceDefinitionV2Email) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Contact *string                       `json:"contact"`
		Type    *ServiceDefinitionV2EmailType `json:"type"`
	}{}
	all := struct {
		Contact string                       `json:"contact"`
		Name    *string                      `json:"name,omitempty"`
		Type    ServiceDefinitionV2EmailType `json:"type"`
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
