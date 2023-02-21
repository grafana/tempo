// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMApplicationCreate RUM application creation.
type RUMApplicationCreate struct {
	// RUM application creation attributes.
	Attributes RUMApplicationCreateAttributes `json:"attributes"`
	// RUM application creation type.
	Type RUMApplicationCreateType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRUMApplicationCreate instantiates a new RUMApplicationCreate object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRUMApplicationCreate(attributes RUMApplicationCreateAttributes, typeVar RUMApplicationCreateType) *RUMApplicationCreate {
	this := RUMApplicationCreate{}
	this.Attributes = attributes
	this.Type = typeVar
	return &this
}

// NewRUMApplicationCreateWithDefaults instantiates a new RUMApplicationCreate object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRUMApplicationCreateWithDefaults() *RUMApplicationCreate {
	this := RUMApplicationCreate{}
	var typeVar RUMApplicationCreateType = RUMAPPLICATIONCREATETYPE_RUM_APPLICATION_CREATE
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *RUMApplicationCreate) GetAttributes() RUMApplicationCreateAttributes {
	if o == nil {
		var ret RUMApplicationCreateAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *RUMApplicationCreate) GetAttributesOk() (*RUMApplicationCreateAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *RUMApplicationCreate) SetAttributes(v RUMApplicationCreateAttributes) {
	o.Attributes = v
}

// GetType returns the Type field value.
func (o *RUMApplicationCreate) GetType() RUMApplicationCreateType {
	if o == nil {
		var ret RUMApplicationCreateType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *RUMApplicationCreate) GetTypeOk() (*RUMApplicationCreateType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *RUMApplicationCreate) SetType(v RUMApplicationCreateType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RUMApplicationCreate) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RUMApplicationCreate) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *RUMApplicationCreateAttributes `json:"attributes"`
		Type       *RUMApplicationCreateType       `json:"type"`
	}{}
	all := struct {
		Attributes RUMApplicationCreateAttributes `json:"attributes"`
		Type       RUMApplicationCreateType       `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("required field attributes missing")
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
	o.Type = all.Type
	return nil
}
