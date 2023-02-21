// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CloudflareAccountUpdateRequestData Data object for updating a Cloudflare account.
type CloudflareAccountUpdateRequestData struct {
	// Attributes object for updating a Cloudflare account.
	Attributes *CloudflareAccountUpdateRequestAttributes `json:"attributes,omitempty"`
	// The JSON:API type for this API. Should always be `cloudflare-accounts`.
	Type *CloudflareAccountType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudflareAccountUpdateRequestData instantiates a new CloudflareAccountUpdateRequestData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudflareAccountUpdateRequestData() *CloudflareAccountUpdateRequestData {
	this := CloudflareAccountUpdateRequestData{}
	var typeVar CloudflareAccountType = CLOUDFLAREACCOUNTTYPE_CLOUDFLARE_ACCOUNTS
	this.Type = &typeVar
	return &this
}

// NewCloudflareAccountUpdateRequestDataWithDefaults instantiates a new CloudflareAccountUpdateRequestData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudflareAccountUpdateRequestDataWithDefaults() *CloudflareAccountUpdateRequestData {
	this := CloudflareAccountUpdateRequestData{}
	var typeVar CloudflareAccountType = CLOUDFLAREACCOUNTTYPE_CLOUDFLARE_ACCOUNTS
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *CloudflareAccountUpdateRequestData) GetAttributes() CloudflareAccountUpdateRequestAttributes {
	if o == nil || o.Attributes == nil {
		var ret CloudflareAccountUpdateRequestAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudflareAccountUpdateRequestData) GetAttributesOk() (*CloudflareAccountUpdateRequestAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *CloudflareAccountUpdateRequestData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given CloudflareAccountUpdateRequestAttributes and assigns it to the Attributes field.
func (o *CloudflareAccountUpdateRequestData) SetAttributes(v CloudflareAccountUpdateRequestAttributes) {
	o.Attributes = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *CloudflareAccountUpdateRequestData) GetType() CloudflareAccountType {
	if o == nil || o.Type == nil {
		var ret CloudflareAccountType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudflareAccountUpdateRequestData) GetTypeOk() (*CloudflareAccountType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *CloudflareAccountUpdateRequestData) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given CloudflareAccountType and assigns it to the Type field.
func (o *CloudflareAccountUpdateRequestData) SetType(v CloudflareAccountType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudflareAccountUpdateRequestData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudflareAccountUpdateRequestData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *CloudflareAccountUpdateRequestAttributes `json:"attributes,omitempty"`
		Type       *CloudflareAccountType                    `json:"type,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Type = all.Type
	return nil
}
