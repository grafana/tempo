// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV1Resource Service's external links.
type ServiceDefinitionV1Resource struct {
	// Link name.
	Name string `json:"name"`
	// Link type.
	Type ServiceDefinitionV1ResourceType `json:"type"`
	// Link URL.
	Url string `json:"url"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV1Resource instantiates a new ServiceDefinitionV1Resource object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV1Resource(name string, typeVar ServiceDefinitionV1ResourceType, url string) *ServiceDefinitionV1Resource {
	this := ServiceDefinitionV1Resource{}
	this.Name = name
	this.Type = typeVar
	this.Url = url
	return &this
}

// NewServiceDefinitionV1ResourceWithDefaults instantiates a new ServiceDefinitionV1Resource object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV1ResourceWithDefaults() *ServiceDefinitionV1Resource {
	this := ServiceDefinitionV1Resource{}
	return &this
}

// GetName returns the Name field value.
func (o *ServiceDefinitionV1Resource) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Resource) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *ServiceDefinitionV1Resource) SetName(v string) {
	o.Name = v
}

// GetType returns the Type field value.
func (o *ServiceDefinitionV1Resource) GetType() ServiceDefinitionV1ResourceType {
	if o == nil {
		var ret ServiceDefinitionV1ResourceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Resource) GetTypeOk() (*ServiceDefinitionV1ResourceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ServiceDefinitionV1Resource) SetType(v ServiceDefinitionV1ResourceType) {
	o.Type = v
}

// GetUrl returns the Url field value.
func (o *ServiceDefinitionV1Resource) GetUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Url
}

// GetUrlOk returns a tuple with the Url field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Resource) GetUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Url, true
}

// SetUrl sets field value.
func (o *ServiceDefinitionV1Resource) SetUrl(v string) {
	o.Url = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV1Resource) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["name"] = o.Name
	toSerialize["type"] = o.Type
	toSerialize["url"] = o.Url

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionV1Resource) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string                          `json:"name"`
		Type *ServiceDefinitionV1ResourceType `json:"type"`
		Url  *string                          `json:"url"`
	}{}
	all := struct {
		Name string                          `json:"name"`
		Type ServiceDefinitionV1ResourceType `json:"type"`
		Url  string                          `json:"url"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
	}
	if required.Type == nil {
		return fmt.Errorf("required field type missing")
	}
	if required.Url == nil {
		return fmt.Errorf("required field url missing")
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
	o.Name = all.Name
	o.Type = all.Type
	o.Url = all.Url
	return nil
}
