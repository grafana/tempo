// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ConfluentAccountUpdateRequestAttributes Attributes object for updating a Confluent account.
type ConfluentAccountUpdateRequestAttributes struct {
	// The API key associated with your Confluent account.
	ApiKey string `json:"api_key"`
	// The API secret associated with your Confluent account.
	ApiSecret string `json:"api_secret"`
	// A list of strings representing tags. Can be a single key, or key-value pairs separated by a colon.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewConfluentAccountUpdateRequestAttributes instantiates a new ConfluentAccountUpdateRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewConfluentAccountUpdateRequestAttributes(apiKey string, apiSecret string) *ConfluentAccountUpdateRequestAttributes {
	this := ConfluentAccountUpdateRequestAttributes{}
	this.ApiKey = apiKey
	this.ApiSecret = apiSecret
	return &this
}

// NewConfluentAccountUpdateRequestAttributesWithDefaults instantiates a new ConfluentAccountUpdateRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewConfluentAccountUpdateRequestAttributesWithDefaults() *ConfluentAccountUpdateRequestAttributes {
	this := ConfluentAccountUpdateRequestAttributes{}
	return &this
}

// GetApiKey returns the ApiKey field value.
func (o *ConfluentAccountUpdateRequestAttributes) GetApiKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value
// and a boolean to check if the value has been set.
func (o *ConfluentAccountUpdateRequestAttributes) GetApiKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ApiKey, true
}

// SetApiKey sets field value.
func (o *ConfluentAccountUpdateRequestAttributes) SetApiKey(v string) {
	o.ApiKey = v
}

// GetApiSecret returns the ApiSecret field value.
func (o *ConfluentAccountUpdateRequestAttributes) GetApiSecret() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ApiSecret
}

// GetApiSecretOk returns a tuple with the ApiSecret field value
// and a boolean to check if the value has been set.
func (o *ConfluentAccountUpdateRequestAttributes) GetApiSecretOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ApiSecret, true
}

// SetApiSecret sets field value.
func (o *ConfluentAccountUpdateRequestAttributes) SetApiSecret(v string) {
	o.ApiSecret = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ConfluentAccountUpdateRequestAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ConfluentAccountUpdateRequestAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ConfluentAccountUpdateRequestAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ConfluentAccountUpdateRequestAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ConfluentAccountUpdateRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["api_key"] = o.ApiKey
	toSerialize["api_secret"] = o.ApiSecret
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ConfluentAccountUpdateRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ApiKey    *string `json:"api_key"`
		ApiSecret *string `json:"api_secret"`
	}{}
	all := struct {
		ApiKey    string   `json:"api_key"`
		ApiSecret string   `json:"api_secret"`
		Tags      []string `json:"tags,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ApiKey == nil {
		return fmt.Errorf("required field api_key missing")
	}
	if required.ApiSecret == nil {
		return fmt.Errorf("required field api_secret missing")
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
	o.ApiSecret = all.ApiSecret
	o.Tags = all.Tags
	return nil
}
