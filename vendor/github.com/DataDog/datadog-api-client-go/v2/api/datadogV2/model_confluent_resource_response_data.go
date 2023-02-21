// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ConfluentResourceResponseData Confluent Cloud resource data.
type ConfluentResourceResponseData struct {
	// Model representation of a Confluent Cloud resource.
	Attributes ConfluentResourceResponseAttributes `json:"attributes"`
	// The ID associated with the Confluent resource.
	Id string `json:"id"`
	// The JSON:API type for this request.
	Type ConfluentResourceType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewConfluentResourceResponseData instantiates a new ConfluentResourceResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewConfluentResourceResponseData(attributes ConfluentResourceResponseAttributes, id string, typeVar ConfluentResourceType) *ConfluentResourceResponseData {
	this := ConfluentResourceResponseData{}
	this.Attributes = attributes
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewConfluentResourceResponseDataWithDefaults instantiates a new ConfluentResourceResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewConfluentResourceResponseDataWithDefaults() *ConfluentResourceResponseData {
	this := ConfluentResourceResponseData{}
	var typeVar ConfluentResourceType = CONFLUENTRESOURCETYPE_CONFLUENT_CLOUD_RESOURCES
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *ConfluentResourceResponseData) GetAttributes() ConfluentResourceResponseAttributes {
	if o == nil {
		var ret ConfluentResourceResponseAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *ConfluentResourceResponseData) GetAttributesOk() (*ConfluentResourceResponseAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *ConfluentResourceResponseData) SetAttributes(v ConfluentResourceResponseAttributes) {
	o.Attributes = v
}

// GetId returns the Id field value.
func (o *ConfluentResourceResponseData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *ConfluentResourceResponseData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *ConfluentResourceResponseData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *ConfluentResourceResponseData) GetType() ConfluentResourceType {
	if o == nil {
		var ret ConfluentResourceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ConfluentResourceResponseData) GetTypeOk() (*ConfluentResourceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ConfluentResourceResponseData) SetType(v ConfluentResourceType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ConfluentResourceResponseData) MarshalJSON() ([]byte, error) {
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
func (o *ConfluentResourceResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *ConfluentResourceResponseAttributes `json:"attributes"`
		Id         *string                              `json:"id"`
		Type       *ConfluentResourceType               `json:"type"`
	}{}
	all := struct {
		Attributes ConfluentResourceResponseAttributes `json:"attributes"`
		Id         string                              `json:"id"`
		Type       ConfluentResourceType               `json:"type"`
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
