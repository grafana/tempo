// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentAttachmentData A single incident attachment.
type IncidentAttachmentData struct {
	// The attributes object for an attachment.
	Attributes IncidentAttachmentAttributes `json:"attributes"`
	// A unique identifier that represents the incident attachment.
	Id string `json:"id"`
	// The incident attachment's relationships.
	Relationships IncidentAttachmentRelationships `json:"relationships"`
	// The incident attachment resource type.
	Type IncidentAttachmentType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentAttachmentData instantiates a new IncidentAttachmentData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentAttachmentData(attributes IncidentAttachmentAttributes, id string, relationships IncidentAttachmentRelationships, typeVar IncidentAttachmentType) *IncidentAttachmentData {
	this := IncidentAttachmentData{}
	this.Attributes = attributes
	this.Id = id
	this.Relationships = relationships
	this.Type = typeVar
	return &this
}

// NewIncidentAttachmentDataWithDefaults instantiates a new IncidentAttachmentData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentAttachmentDataWithDefaults() *IncidentAttachmentData {
	this := IncidentAttachmentData{}
	var typeVar IncidentAttachmentType = INCIDENTATTACHMENTTYPE_INCIDENT_ATTACHMENTS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *IncidentAttachmentData) GetAttributes() IncidentAttachmentAttributes {
	if o == nil {
		var ret IncidentAttachmentAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentData) GetAttributesOk() (*IncidentAttachmentAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *IncidentAttachmentData) SetAttributes(v IncidentAttachmentAttributes) {
	o.Attributes = v
}

// GetId returns the Id field value.
func (o *IncidentAttachmentData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *IncidentAttachmentData) SetId(v string) {
	o.Id = v
}

// GetRelationships returns the Relationships field value.
func (o *IncidentAttachmentData) GetRelationships() IncidentAttachmentRelationships {
	if o == nil {
		var ret IncidentAttachmentRelationships
		return ret
	}
	return o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentData) GetRelationshipsOk() (*IncidentAttachmentRelationships, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Relationships, true
}

// SetRelationships sets field value.
func (o *IncidentAttachmentData) SetRelationships(v IncidentAttachmentRelationships) {
	o.Relationships = v
}

// GetType returns the Type field value.
func (o *IncidentAttachmentData) GetType() IncidentAttachmentType {
	if o == nil {
		var ret IncidentAttachmentType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentData) GetTypeOk() (*IncidentAttachmentType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IncidentAttachmentData) SetType(v IncidentAttachmentType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentAttachmentData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["id"] = o.Id
	toSerialize["relationships"] = o.Relationships
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentAttachmentData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes    *IncidentAttachmentAttributes    `json:"attributes"`
		Id            *string                          `json:"id"`
		Relationships *IncidentAttachmentRelationships `json:"relationships"`
		Type          *IncidentAttachmentType          `json:"type"`
	}{}
	all := struct {
		Attributes    IncidentAttachmentAttributes    `json:"attributes"`
		Id            string                          `json:"id"`
		Relationships IncidentAttachmentRelationships `json:"relationships"`
		Type          IncidentAttachmentType          `json:"type"`
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
	if required.Relationships == nil {
		return fmt.Errorf("required field relationships missing")
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
	o.Attributes = all.Attributes
	o.Id = all.Id
	if all.Relationships.UnparsedObject != nil && o.UnparsedObject == nil {
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
