// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// PartialApplicationKey Partial Datadog application key.
type PartialApplicationKey struct {
	// Attributes of a partial application key.
	Attributes *PartialApplicationKeyAttributes `json:"attributes,omitempty"`
	// ID of the application key.
	Id *string `json:"id,omitempty"`
	// Resources related to the application key.
	Relationships *ApplicationKeyRelationships `json:"relationships,omitempty"`
	// Application Keys resource type.
	Type *ApplicationKeysType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewPartialApplicationKey instantiates a new PartialApplicationKey object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewPartialApplicationKey() *PartialApplicationKey {
	this := PartialApplicationKey{}
	var typeVar ApplicationKeysType = APPLICATIONKEYSTYPE_APPLICATION_KEYS
	this.Type = &typeVar
	return &this
}

// NewPartialApplicationKeyWithDefaults instantiates a new PartialApplicationKey object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewPartialApplicationKeyWithDefaults() *PartialApplicationKey {
	this := PartialApplicationKey{}
	var typeVar ApplicationKeysType = APPLICATIONKEYSTYPE_APPLICATION_KEYS
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *PartialApplicationKey) GetAttributes() PartialApplicationKeyAttributes {
	if o == nil || o.Attributes == nil {
		var ret PartialApplicationKeyAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialApplicationKey) GetAttributesOk() (*PartialApplicationKeyAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *PartialApplicationKey) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given PartialApplicationKeyAttributes and assigns it to the Attributes field.
func (o *PartialApplicationKey) SetAttributes(v PartialApplicationKeyAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *PartialApplicationKey) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialApplicationKey) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *PartialApplicationKey) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *PartialApplicationKey) SetId(v string) {
	o.Id = &v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *PartialApplicationKey) GetRelationships() ApplicationKeyRelationships {
	if o == nil || o.Relationships == nil {
		var ret ApplicationKeyRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialApplicationKey) GetRelationshipsOk() (*ApplicationKeyRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *PartialApplicationKey) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given ApplicationKeyRelationships and assigns it to the Relationships field.
func (o *PartialApplicationKey) SetRelationships(v ApplicationKeyRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *PartialApplicationKey) GetType() ApplicationKeysType {
	if o == nil || o.Type == nil {
		var ret ApplicationKeysType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialApplicationKey) GetTypeOk() (*ApplicationKeysType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *PartialApplicationKey) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given ApplicationKeysType and assigns it to the Type field.
func (o *PartialApplicationKey) SetType(v ApplicationKeysType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o PartialApplicationKey) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.Relationships != nil {
		toSerialize["relationships"] = o.Relationships
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
func (o *PartialApplicationKey) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes    *PartialApplicationKeyAttributes `json:"attributes,omitempty"`
		Id            *string                          `json:"id,omitempty"`
		Relationships *ApplicationKeyRelationships     `json:"relationships,omitempty"`
		Type          *ApplicationKeysType             `json:"type,omitempty"`
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
