// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// FastlyAccountUpdateRequestAttributes Attributes object for updating a Fastly account.
type FastlyAccountUpdateRequestAttributes struct {
	// The API key of the Fastly account.
	ApiKey *string `json:"api_key,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewFastlyAccountUpdateRequestAttributes instantiates a new FastlyAccountUpdateRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFastlyAccountUpdateRequestAttributes() *FastlyAccountUpdateRequestAttributes {
	this := FastlyAccountUpdateRequestAttributes{}
	return &this
}

// NewFastlyAccountUpdateRequestAttributesWithDefaults instantiates a new FastlyAccountUpdateRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFastlyAccountUpdateRequestAttributesWithDefaults() *FastlyAccountUpdateRequestAttributes {
	this := FastlyAccountUpdateRequestAttributes{}
	return &this
}

// GetApiKey returns the ApiKey field value if set, zero value otherwise.
func (o *FastlyAccountUpdateRequestAttributes) GetApiKey() string {
	if o == nil || o.ApiKey == nil {
		var ret string
		return ret
	}
	return *o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FastlyAccountUpdateRequestAttributes) GetApiKeyOk() (*string, bool) {
	if o == nil || o.ApiKey == nil {
		return nil, false
	}
	return o.ApiKey, true
}

// HasApiKey returns a boolean if a field has been set.
func (o *FastlyAccountUpdateRequestAttributes) HasApiKey() bool {
	return o != nil && o.ApiKey != nil
}

// SetApiKey gets a reference to the given string and assigns it to the ApiKey field.
func (o *FastlyAccountUpdateRequestAttributes) SetApiKey(v string) {
	o.ApiKey = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o FastlyAccountUpdateRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ApiKey != nil {
		toSerialize["api_key"] = o.ApiKey
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FastlyAccountUpdateRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ApiKey *string `json:"api_key,omitempty"`
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
	o.ApiKey = all.ApiKey
	return nil
}
