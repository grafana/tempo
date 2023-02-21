// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudflareAccountCreateRequestAttributes Attributes object for creating a Cloudflare account.
type CloudflareAccountCreateRequestAttributes struct {
	// The API key (or token) for the Cloudflare account.
	ApiKey string `json:"api_key"`
	// The email associated with the Cloudflare account. If an API key is provided (and not a token), this field is also required.
	Email *string `json:"email,omitempty"`
	// The name of the Cloudflare account.
	Name string `json:"name"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudflareAccountCreateRequestAttributes instantiates a new CloudflareAccountCreateRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudflareAccountCreateRequestAttributes(apiKey string, name string) *CloudflareAccountCreateRequestAttributes {
	this := CloudflareAccountCreateRequestAttributes{}
	this.ApiKey = apiKey
	this.Name = name
	return &this
}

// NewCloudflareAccountCreateRequestAttributesWithDefaults instantiates a new CloudflareAccountCreateRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudflareAccountCreateRequestAttributesWithDefaults() *CloudflareAccountCreateRequestAttributes {
	this := CloudflareAccountCreateRequestAttributes{}
	return &this
}

// GetApiKey returns the ApiKey field value.
func (o *CloudflareAccountCreateRequestAttributes) GetApiKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value
// and a boolean to check if the value has been set.
func (o *CloudflareAccountCreateRequestAttributes) GetApiKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ApiKey, true
}

// SetApiKey sets field value.
func (o *CloudflareAccountCreateRequestAttributes) SetApiKey(v string) {
	o.ApiKey = v
}

// GetEmail returns the Email field value if set, zero value otherwise.
func (o *CloudflareAccountCreateRequestAttributes) GetEmail() string {
	if o == nil || o.Email == nil {
		var ret string
		return ret
	}
	return *o.Email
}

// GetEmailOk returns a tuple with the Email field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudflareAccountCreateRequestAttributes) GetEmailOk() (*string, bool) {
	if o == nil || o.Email == nil {
		return nil, false
	}
	return o.Email, true
}

// HasEmail returns a boolean if a field has been set.
func (o *CloudflareAccountCreateRequestAttributes) HasEmail() bool {
	return o != nil && o.Email != nil
}

// SetEmail gets a reference to the given string and assigns it to the Email field.
func (o *CloudflareAccountCreateRequestAttributes) SetEmail(v string) {
	o.Email = &v
}

// GetName returns the Name field value.
func (o *CloudflareAccountCreateRequestAttributes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *CloudflareAccountCreateRequestAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *CloudflareAccountCreateRequestAttributes) SetName(v string) {
	o.Name = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudflareAccountCreateRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["api_key"] = o.ApiKey
	if o.Email != nil {
		toSerialize["email"] = o.Email
	}
	toSerialize["name"] = o.Name

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudflareAccountCreateRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ApiKey *string `json:"api_key"`
		Name   *string `json:"name"`
	}{}
	all := struct {
		ApiKey string  `json:"api_key"`
		Email  *string `json:"email,omitempty"`
		Name   string  `json:"name"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ApiKey == nil {
		return fmt.Errorf("required field api_key missing")
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
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
	o.Name = all.Name
	return nil
}
