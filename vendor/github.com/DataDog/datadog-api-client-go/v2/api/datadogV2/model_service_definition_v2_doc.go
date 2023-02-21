// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV2Doc Service documents.
type ServiceDefinitionV2Doc struct {
	// Document name.
	Name string `json:"name"`
	// Document provider.
	Provider *string `json:"provider,omitempty"`
	// Document URL.
	Url string `json:"url"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV2Doc instantiates a new ServiceDefinitionV2Doc object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV2Doc(name string, url string) *ServiceDefinitionV2Doc {
	this := ServiceDefinitionV2Doc{}
	this.Name = name
	this.Url = url
	return &this
}

// NewServiceDefinitionV2DocWithDefaults instantiates a new ServiceDefinitionV2Doc object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV2DocWithDefaults() *ServiceDefinitionV2Doc {
	this := ServiceDefinitionV2Doc{}
	return &this
}

// GetName returns the Name field value.
func (o *ServiceDefinitionV2Doc) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Doc) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *ServiceDefinitionV2Doc) SetName(v string) {
	o.Name = v
}

// GetProvider returns the Provider field value if set, zero value otherwise.
func (o *ServiceDefinitionV2Doc) GetProvider() string {
	if o == nil || o.Provider == nil {
		var ret string
		return ret
	}
	return *o.Provider
}

// GetProviderOk returns a tuple with the Provider field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Doc) GetProviderOk() (*string, bool) {
	if o == nil || o.Provider == nil {
		return nil, false
	}
	return o.Provider, true
}

// HasProvider returns a boolean if a field has been set.
func (o *ServiceDefinitionV2Doc) HasProvider() bool {
	return o != nil && o.Provider != nil
}

// SetProvider gets a reference to the given string and assigns it to the Provider field.
func (o *ServiceDefinitionV2Doc) SetProvider(v string) {
	o.Provider = &v
}

// GetUrl returns the Url field value.
func (o *ServiceDefinitionV2Doc) GetUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Url
}

// GetUrlOk returns a tuple with the Url field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Doc) GetUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Url, true
}

// SetUrl sets field value.
func (o *ServiceDefinitionV2Doc) SetUrl(v string) {
	o.Url = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV2Doc) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["name"] = o.Name
	if o.Provider != nil {
		toSerialize["provider"] = o.Provider
	}
	toSerialize["url"] = o.Url

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionV2Doc) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
		Url  *string `json:"url"`
	}{}
	all := struct {
		Name     string  `json:"name"`
		Provider *string `json:"provider,omitempty"`
		Url      string  `json:"url"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
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
	o.Name = all.Name
	o.Provider = all.Provider
	o.Url = all.Url
	return nil
}
