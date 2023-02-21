// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudflareAccountUpdateRequestAttributes Attributes object for updating a Cloudflare account.
type CloudflareAccountUpdateRequestAttributes struct {
	// The API key of the Cloudflare account.
	ApiKey string `json:"api_key"`
	// The email associated with the Cloudflare account. If an API key is provided (and not a token), this field is also required.
	Email *string `json:"email,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudflareAccountUpdateRequestAttributes instantiates a new CloudflareAccountUpdateRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudflareAccountUpdateRequestAttributes(apiKey string) *CloudflareAccountUpdateRequestAttributes {
	this := CloudflareAccountUpdateRequestAttributes{}
	this.ApiKey = apiKey
	return &this
}

// NewCloudflareAccountUpdateRequestAttributesWithDefaults instantiates a new CloudflareAccountUpdateRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudflareAccountUpdateRequestAttributesWithDefaults() *CloudflareAccountUpdateRequestAttributes {
	this := CloudflareAccountUpdateRequestAttributes{}
	return &this
}

// GetApiKey returns the ApiKey field value.
func (o *CloudflareAccountUpdateRequestAttributes) GetApiKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value
// and a boolean to check if the value has been set.
func (o *CloudflareAccountUpdateRequestAttributes) GetApiKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ApiKey, true
}

// SetApiKey sets field value.
func (o *CloudflareAccountUpdateRequestAttributes) SetApiKey(v string) {
	o.ApiKey = v
}

// GetEmail returns the Email field value if set, zero value otherwise.
func (o *CloudflareAccountUpdateRequestAttributes) GetEmail() string {
	if o == nil || o.Email == nil {
		var ret string
		return ret
	}
	return *o.Email
}

// GetEmailOk returns a tuple with the Email field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudflareAccountUpdateRequestAttributes) GetEmailOk() (*string, bool) {
	if o == nil || o.Email == nil {
		return nil, false
	}
	return o.Email, true
}

// HasEmail returns a boolean if a field has been set.
func (o *CloudflareAccountUpdateRequestAttributes) HasEmail() bool {
	return o != nil && o.Email != nil
}

// SetEmail gets a reference to the given string and assigns it to the Email field.
func (o *CloudflareAccountUpdateRequestAttributes) SetEmail(v string) {
	o.Email = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudflareAccountUpdateRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["api_key"] = o.ApiKey
	if o.Email != nil {
		toSerialize["email"] = o.Email
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudflareAccountUpdateRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ApiKey *string `json:"api_key"`
	}{}
	all := struct {
		ApiKey string  `json:"api_key"`
		Email  *string `json:"email,omitempty"`
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
	o.Email = all.Email
	return nil
}
