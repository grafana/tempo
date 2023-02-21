// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// FastlyAccountUpdateRequestData Data object for updating a Fastly account.
type FastlyAccountUpdateRequestData struct {
	// Attributes object for updating a Fastly account.
	Attributes *FastlyAccountUpdateRequestAttributes `json:"attributes,omitempty"`
	// The JSON:API type for this API. Should always be `fastly-accounts`.
	Type *FastlyAccountType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewFastlyAccountUpdateRequestData instantiates a new FastlyAccountUpdateRequestData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFastlyAccountUpdateRequestData() *FastlyAccountUpdateRequestData {
	this := FastlyAccountUpdateRequestData{}
	var typeVar FastlyAccountType = FASTLYACCOUNTTYPE_FASTLY_ACCOUNTS
	this.Type = &typeVar
	return &this
}

// NewFastlyAccountUpdateRequestDataWithDefaults instantiates a new FastlyAccountUpdateRequestData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFastlyAccountUpdateRequestDataWithDefaults() *FastlyAccountUpdateRequestData {
	this := FastlyAccountUpdateRequestData{}
	var typeVar FastlyAccountType = FASTLYACCOUNTTYPE_FASTLY_ACCOUNTS
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *FastlyAccountUpdateRequestData) GetAttributes() FastlyAccountUpdateRequestAttributes {
	if o == nil || o.Attributes == nil {
		var ret FastlyAccountUpdateRequestAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FastlyAccountUpdateRequestData) GetAttributesOk() (*FastlyAccountUpdateRequestAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *FastlyAccountUpdateRequestData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given FastlyAccountUpdateRequestAttributes and assigns it to the Attributes field.
func (o *FastlyAccountUpdateRequestData) SetAttributes(v FastlyAccountUpdateRequestAttributes) {
	o.Attributes = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *FastlyAccountUpdateRequestData) GetType() FastlyAccountType {
	if o == nil || o.Type == nil {
		var ret FastlyAccountType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FastlyAccountUpdateRequestData) GetTypeOk() (*FastlyAccountType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *FastlyAccountUpdateRequestData) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given FastlyAccountType and assigns it to the Type field.
func (o *FastlyAccountUpdateRequestData) SetType(v FastlyAccountType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o FastlyAccountUpdateRequestData) MarshalJSON() ([]byte, error) {
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
func (o *FastlyAccountUpdateRequestData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *FastlyAccountUpdateRequestAttributes `json:"attributes,omitempty"`
		Type       *FastlyAccountType                    `json:"type,omitempty"`
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
