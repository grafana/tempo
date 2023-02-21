// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// FastlyAccountResponseData Data object of a Fastly account.
type FastlyAccountResponseData struct {
	// Attributes object of a Fastly account.
	Attributes FastlyAccounResponseAttributes `json:"attributes"`
	// The ID of the Fastly account, a hash of the account name.
	Id string `json:"id"`
	// The JSON:API type for this API. Should always be `fastly-accounts`.
	Type FastlyAccountType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewFastlyAccountResponseData instantiates a new FastlyAccountResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFastlyAccountResponseData(attributes FastlyAccounResponseAttributes, id string, typeVar FastlyAccountType) *FastlyAccountResponseData {
	this := FastlyAccountResponseData{}
	this.Attributes = attributes
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewFastlyAccountResponseDataWithDefaults instantiates a new FastlyAccountResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFastlyAccountResponseDataWithDefaults() *FastlyAccountResponseData {
	this := FastlyAccountResponseData{}
	var typeVar FastlyAccountType = FASTLYACCOUNTTYPE_FASTLY_ACCOUNTS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *FastlyAccountResponseData) GetAttributes() FastlyAccounResponseAttributes {
	if o == nil {
		var ret FastlyAccounResponseAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *FastlyAccountResponseData) GetAttributesOk() (*FastlyAccounResponseAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *FastlyAccountResponseData) SetAttributes(v FastlyAccounResponseAttributes) {
	o.Attributes = v
}

// GetId returns the Id field value.
func (o *FastlyAccountResponseData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *FastlyAccountResponseData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *FastlyAccountResponseData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *FastlyAccountResponseData) GetType() FastlyAccountType {
	if o == nil {
		var ret FastlyAccountType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *FastlyAccountResponseData) GetTypeOk() (*FastlyAccountType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *FastlyAccountResponseData) SetType(v FastlyAccountType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o FastlyAccountResponseData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["id"] = o.Id
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FastlyAccountResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *FastlyAccounResponseAttributes `json:"attributes"`
		Id         *string                         `json:"id"`
		Type       *FastlyAccountType              `json:"type"`
	}{}
	all := struct {
		Attributes FastlyAccounResponseAttributes `json:"attributes"`
		Id         string                         `json:"id"`
		Type       FastlyAccountType              `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("required field attributes missing")
	}
	if required.Id == nil {
		return fmt.Errorf("required field id missing")
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
	if all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
