// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ConfluentAccountResponseAttributes The attributes of a Confluent account.
type ConfluentAccountResponseAttributes struct {
	// The API key associated with your Confluent account.
	ApiKey string `json:"api_key"`
	// A list of Confluent resources associated with the Confluent account.
	Resources []ConfluentResourceResponseAttributes `json:"resources,omitempty"`
	// A list of strings representing tags. Can be a single key, or key-value pairs separated by a colon.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewConfluentAccountResponseAttributes instantiates a new ConfluentAccountResponseAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewConfluentAccountResponseAttributes(apiKey string) *ConfluentAccountResponseAttributes {
	this := ConfluentAccountResponseAttributes{}
	this.ApiKey = apiKey
	return &this
}

// NewConfluentAccountResponseAttributesWithDefaults instantiates a new ConfluentAccountResponseAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewConfluentAccountResponseAttributesWithDefaults() *ConfluentAccountResponseAttributes {
	this := ConfluentAccountResponseAttributes{}
	return &this
}

// GetApiKey returns the ApiKey field value.
func (o *ConfluentAccountResponseAttributes) GetApiKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value
// and a boolean to check if the value has been set.
func (o *ConfluentAccountResponseAttributes) GetApiKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ApiKey, true
}

// SetApiKey sets field value.
func (o *ConfluentAccountResponseAttributes) SetApiKey(v string) {
	o.ApiKey = v
}

// GetResources returns the Resources field value if set, zero value otherwise.
func (o *ConfluentAccountResponseAttributes) GetResources() []ConfluentResourceResponseAttributes {
	if o == nil || o.Resources == nil {
		var ret []ConfluentResourceResponseAttributes
		return ret
	}
	return o.Resources
}

// GetResourcesOk returns a tuple with the Resources field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ConfluentAccountResponseAttributes) GetResourcesOk() (*[]ConfluentResourceResponseAttributes, bool) {
	if o == nil || o.Resources == nil {
		return nil, false
	}
	return &o.Resources, true
}

// HasResources returns a boolean if a field has been set.
func (o *ConfluentAccountResponseAttributes) HasResources() bool {
	return o != nil && o.Resources != nil
}

// SetResources gets a reference to the given []ConfluentResourceResponseAttributes and assigns it to the Resources field.
func (o *ConfluentAccountResponseAttributes) SetResources(v []ConfluentResourceResponseAttributes) {
	o.Resources = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ConfluentAccountResponseAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ConfluentAccountResponseAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ConfluentAccountResponseAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ConfluentAccountResponseAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ConfluentAccountResponseAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["api_key"] = o.ApiKey
	if o.Resources != nil {
		toSerialize["resources"] = o.Resources
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ConfluentAccountResponseAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ApiKey *string `json:"api_key"`
	}{}
	all := struct {
		ApiKey    string                                `json:"api_key"`
		Resources []ConfluentResourceResponseAttributes `json:"resources,omitempty"`
		Tags      []string                              `json:"tags,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ApiKey == nil {
		return fmt.Errorf("required field api_key missing")
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
	o.ApiKey = all.ApiKey
	o.Resources = all.Resources
	o.Tags = all.Tags
	return nil
}
