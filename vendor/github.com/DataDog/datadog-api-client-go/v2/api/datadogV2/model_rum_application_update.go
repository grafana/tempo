// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMApplicationUpdate RUM application update.
type RUMApplicationUpdate struct {
	// RUM application update attributes.
	Attributes *RUMApplicationUpdateAttributes `json:"attributes,omitempty"`
	// RUM application ID.
	Id string `json:"id"`
	// RUM application update type.
	Type RUMApplicationUpdateType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRUMApplicationUpdate instantiates a new RUMApplicationUpdate object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRUMApplicationUpdate(id string, typeVar RUMApplicationUpdateType) *RUMApplicationUpdate {
	this := RUMApplicationUpdate{}
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewRUMApplicationUpdateWithDefaults instantiates a new RUMApplicationUpdate object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRUMApplicationUpdateWithDefaults() *RUMApplicationUpdate {
	this := RUMApplicationUpdate{}
	var typeVar RUMApplicationUpdateType = RUMAPPLICATIONUPDATETYPE_RUM_APPLICATION_UPDATE
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *RUMApplicationUpdate) GetAttributes() RUMApplicationUpdateAttributes {
	if o == nil || o.Attributes == nil {
		var ret RUMApplicationUpdateAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMApplicationUpdate) GetAttributesOk() (*RUMApplicationUpdateAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *RUMApplicationUpdate) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given RUMApplicationUpdateAttributes and assigns it to the Attributes field.
func (o *RUMApplicationUpdate) SetAttributes(v RUMApplicationUpdateAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value.
func (o *RUMApplicationUpdate) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *RUMApplicationUpdate) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *RUMApplicationUpdate) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *RUMApplicationUpdate) GetType() RUMApplicationUpdateType {
	if o == nil {
		var ret RUMApplicationUpdateType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *RUMApplicationUpdate) GetTypeOk() (*RUMApplicationUpdateType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *RUMApplicationUpdate) SetType(v RUMApplicationUpdateType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RUMApplicationUpdate) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	toSerialize["id"] = o.Id
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RUMApplicationUpdate) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Id   *string                   `json:"id"`
		Type *RUMApplicationUpdateType `json:"type"`
	}{}
	all := struct {
		Attributes *RUMApplicationUpdateAttributes `json:"attributes,omitempty"`
		Id         string                          `json:"id"`
		Type       RUMApplicationUpdateType        `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
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
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
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
