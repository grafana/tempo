// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentResponseData Incident data from a response.
type IncidentResponseData struct {
	// The incident's attributes from a response.
	Attributes *IncidentResponseAttributes `json:"attributes,omitempty"`
	// The incident's ID.
	Id string `json:"id"`
	// The incident's relationships from a response.
	Relationships *IncidentResponseRelationships `json:"relationships,omitempty"`
	// Incident resource type.
	Type IncidentType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentResponseData instantiates a new IncidentResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentResponseData(id string, typeVar IncidentType) *IncidentResponseData {
	this := IncidentResponseData{}
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewIncidentResponseDataWithDefaults instantiates a new IncidentResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentResponseDataWithDefaults() *IncidentResponseData {
	this := IncidentResponseData{}
	var typeVar IncidentType = INCIDENTTYPE_INCIDENTS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *IncidentResponseData) GetAttributes() IncidentResponseAttributes {
	if o == nil || o.Attributes == nil {
		var ret IncidentResponseAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseData) GetAttributesOk() (*IncidentResponseAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *IncidentResponseData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given IncidentResponseAttributes and assigns it to the Attributes field.
func (o *IncidentResponseData) SetAttributes(v IncidentResponseAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value.
func (o *IncidentResponseData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *IncidentResponseData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *IncidentResponseData) SetId(v string) {
	o.Id = v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *IncidentResponseData) GetRelationships() IncidentResponseRelationships {
	if o == nil || o.Relationships == nil {
		var ret IncidentResponseRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseData) GetRelationshipsOk() (*IncidentResponseRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *IncidentResponseData) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given IncidentResponseRelationships and assigns it to the Relationships field.
func (o *IncidentResponseData) SetRelationships(v IncidentResponseRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *IncidentResponseData) GetType() IncidentType {
	if o == nil {
		var ret IncidentType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IncidentResponseData) GetTypeOk() (*IncidentType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IncidentResponseData) SetType(v IncidentType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentResponseData) MarshalJSON() ([]byte, error) {
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
func (o *IncidentResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Id   *string       `json:"id"`
		Type *IncidentType `json:"type"`
	}{}
	all := struct {
		Attributes    *IncidentResponseAttributes    `json:"attributes,omitempty"`
		Id            string                         `json:"id"`
		Relationships *IncidentResponseRelationships `json:"relationships,omitempty"`
		Type          IncidentType                   `json:"type"`
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
