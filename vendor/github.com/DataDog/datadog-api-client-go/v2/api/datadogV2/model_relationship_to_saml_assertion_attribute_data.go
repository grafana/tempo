// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RelationshipToSAMLAssertionAttributeData Data of AuthN Mapping relationship to SAML Assertion Attribute.
type RelationshipToSAMLAssertionAttributeData struct {
	// The ID of the SAML assertion attribute.
	Id string `json:"id"`
	// SAML assertion attributes resource type.
	Type SAMLAssertionAttributesType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRelationshipToSAMLAssertionAttributeData instantiates a new RelationshipToSAMLAssertionAttributeData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRelationshipToSAMLAssertionAttributeData(id string, typeVar SAMLAssertionAttributesType) *RelationshipToSAMLAssertionAttributeData {
	this := RelationshipToSAMLAssertionAttributeData{}
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewRelationshipToSAMLAssertionAttributeDataWithDefaults instantiates a new RelationshipToSAMLAssertionAttributeData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRelationshipToSAMLAssertionAttributeDataWithDefaults() *RelationshipToSAMLAssertionAttributeData {
	this := RelationshipToSAMLAssertionAttributeData{}
	var typeVar SAMLAssertionAttributesType = SAMLASSERTIONATTRIBUTESTYPE_SAML_ASSERTION_ATTRIBUTES
	this.Type = typeVar
	return &this
}

// GetId returns the Id field value.
func (o *RelationshipToSAMLAssertionAttributeData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *RelationshipToSAMLAssertionAttributeData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *RelationshipToSAMLAssertionAttributeData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *RelationshipToSAMLAssertionAttributeData) GetType() SAMLAssertionAttributesType {
	if o == nil {
		var ret SAMLAssertionAttributesType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *RelationshipToSAMLAssertionAttributeData) GetTypeOk() (*SAMLAssertionAttributesType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *RelationshipToSAMLAssertionAttributeData) SetType(v SAMLAssertionAttributesType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RelationshipToSAMLAssertionAttributeData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["id"] = o.Id
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RelationshipToSAMLAssertionAttributeData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Id   *string                      `json:"id"`
		Type *SAMLAssertionAttributesType `json:"type"`
	}{}
	all := struct {
		Id   string                      `json:"id"`
		Type SAMLAssertionAttributesType `json:"type"`
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
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
