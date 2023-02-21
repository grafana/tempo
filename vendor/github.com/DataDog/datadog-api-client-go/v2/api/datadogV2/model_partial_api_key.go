// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// PartialAPIKey Partial Datadog API key.
type PartialAPIKey struct {
	// Attributes of a partial API key.
	Attributes *PartialAPIKeyAttributes `json:"attributes,omitempty"`
	// ID of the API key.
	Id *string `json:"id,omitempty"`
	// Resources related to the API key.
	Relationships *APIKeyRelationships `json:"relationships,omitempty"`
	// API Keys resource type.
	Type *APIKeysType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewPartialAPIKey instantiates a new PartialAPIKey object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewPartialAPIKey() *PartialAPIKey {
	this := PartialAPIKey{}
	var typeVar APIKeysType = APIKEYSTYPE_API_KEYS
	this.Type = &typeVar
	return &this
}

// NewPartialAPIKeyWithDefaults instantiates a new PartialAPIKey object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewPartialAPIKeyWithDefaults() *PartialAPIKey {
	this := PartialAPIKey{}
	var typeVar APIKeysType = APIKEYSTYPE_API_KEYS
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *PartialAPIKey) GetAttributes() PartialAPIKeyAttributes {
	if o == nil || o.Attributes == nil {
		var ret PartialAPIKeyAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialAPIKey) GetAttributesOk() (*PartialAPIKeyAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *PartialAPIKey) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given PartialAPIKeyAttributes and assigns it to the Attributes field.
func (o *PartialAPIKey) SetAttributes(v PartialAPIKeyAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *PartialAPIKey) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialAPIKey) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *PartialAPIKey) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *PartialAPIKey) SetId(v string) {
	o.Id = &v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *PartialAPIKey) GetRelationships() APIKeyRelationships {
	if o == nil || o.Relationships == nil {
		var ret APIKeyRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialAPIKey) GetRelationshipsOk() (*APIKeyRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *PartialAPIKey) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given APIKeyRelationships and assigns it to the Relationships field.
func (o *PartialAPIKey) SetRelationships(v APIKeyRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *PartialAPIKey) GetType() APIKeysType {
	if o == nil || o.Type == nil {
		var ret APIKeysType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialAPIKey) GetTypeOk() (*APIKeysType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *PartialAPIKey) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given APIKeysType and assigns it to the Type field.
func (o *PartialAPIKey) SetType(v APIKeysType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o PartialAPIKey) MarshalJSON() ([]byte, error) {
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
func (o *PartialAPIKey) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes    *PartialAPIKeyAttributes `json:"attributes,omitempty"`
		Id            *string                  `json:"id,omitempty"`
		Relationships *APIKeyRelationships     `json:"relationships,omitempty"`
		Type          *APIKeysType             `json:"type,omitempty"`
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
