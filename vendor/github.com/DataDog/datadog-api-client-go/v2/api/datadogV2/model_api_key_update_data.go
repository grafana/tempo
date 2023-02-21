// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// APIKeyUpdateData Object used to update an API key.
type APIKeyUpdateData struct {
	// Attributes used to update an API Key.
	Attributes APIKeyUpdateAttributes `json:"attributes"`
	// ID of the API key.
	Id string `json:"id"`
	// API Keys resource type.
	Type APIKeysType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAPIKeyUpdateData instantiates a new APIKeyUpdateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAPIKeyUpdateData(attributes APIKeyUpdateAttributes, id string, typeVar APIKeysType) *APIKeyUpdateData {
	this := APIKeyUpdateData{}
	this.Attributes = attributes
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewAPIKeyUpdateDataWithDefaults instantiates a new APIKeyUpdateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAPIKeyUpdateDataWithDefaults() *APIKeyUpdateData {
	this := APIKeyUpdateData{}
	var typeVar APIKeysType = APIKEYSTYPE_API_KEYS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *APIKeyUpdateData) GetAttributes() APIKeyUpdateAttributes {
	if o == nil {
		var ret APIKeyUpdateAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *APIKeyUpdateData) GetAttributesOk() (*APIKeyUpdateAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *APIKeyUpdateData) SetAttributes(v APIKeyUpdateAttributes) {
	o.Attributes = v
}

// GetId returns the Id field value.
func (o *APIKeyUpdateData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *APIKeyUpdateData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *APIKeyUpdateData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *APIKeyUpdateData) GetType() APIKeysType {
	if o == nil {
		var ret APIKeysType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *APIKeyUpdateData) GetTypeOk() (*APIKeysType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *APIKeyUpdateData) SetType(v APIKeysType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o APIKeyUpdateData) MarshalJSON() ([]byte, error) {
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
func (o *APIKeyUpdateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *APIKeyUpdateAttributes `json:"attributes"`
		Id         *string                 `json:"id"`
		Type       *APIKeysType            `json:"type"`
	}{}
	all := struct {
		Attributes APIKeyUpdateAttributes `json:"attributes"`
		Id         string                 `json:"id"`
		Type       APIKeysType            `json:"type"`
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
