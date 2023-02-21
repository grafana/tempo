// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// AuthNMapping The AuthN Mapping object returned by API.
type AuthNMapping struct {
	// Attributes of AuthN Mapping.
	Attributes *AuthNMappingAttributes `json:"attributes,omitempty"`
	// ID of the AuthN Mapping.
	Id string `json:"id"`
	// All relationships associated with AuthN Mapping.
	Relationships *AuthNMappingRelationships `json:"relationships,omitempty"`
	// AuthN Mappings resource type.
	Type AuthNMappingsType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAuthNMapping instantiates a new AuthNMapping object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAuthNMapping(id string, typeVar AuthNMappingsType) *AuthNMapping {
	this := AuthNMapping{}
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewAuthNMappingWithDefaults instantiates a new AuthNMapping object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAuthNMappingWithDefaults() *AuthNMapping {
	this := AuthNMapping{}
	var typeVar AuthNMappingsType = AUTHNMAPPINGSTYPE_AUTHN_MAPPINGS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *AuthNMapping) GetAttributes() AuthNMappingAttributes {
	if o == nil || o.Attributes == nil {
		var ret AuthNMappingAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMapping) GetAttributesOk() (*AuthNMappingAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *AuthNMapping) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given AuthNMappingAttributes and assigns it to the Attributes field.
func (o *AuthNMapping) SetAttributes(v AuthNMappingAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value.
func (o *AuthNMapping) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *AuthNMapping) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *AuthNMapping) SetId(v string) {
	o.Id = v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *AuthNMapping) GetRelationships() AuthNMappingRelationships {
	if o == nil || o.Relationships == nil {
		var ret AuthNMappingRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMapping) GetRelationshipsOk() (*AuthNMappingRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *AuthNMapping) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given AuthNMappingRelationships and assigns it to the Relationships field.
func (o *AuthNMapping) SetRelationships(v AuthNMappingRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *AuthNMapping) GetType() AuthNMappingsType {
	if o == nil {
		var ret AuthNMappingsType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *AuthNMapping) GetTypeOk() (*AuthNMappingsType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *AuthNMapping) SetType(v AuthNMappingsType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o AuthNMapping) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	toSerialize["id"] = o.Id
	if o.Relationships != nil {
		toSerialize["relationships"] = o.Relationships
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AuthNMapping) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Id   *string            `json:"id"`
		Type *AuthNMappingsType `json:"type"`
	}{}
	all := struct {
		Attributes    *AuthNMappingAttributes    `json:"attributes,omitempty"`
		Id            string                     `json:"id"`
		Relationships *AuthNMappingRelationships `json:"relationships,omitempty"`
		Type          AuthNMappingsType          `json:"type"`
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
	if all.Relationships != nil && all.Relationships.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Relationships = all.Relationships
	o.Type = all.Type
	return nil
}
