// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RelationshipToIncidentAttachmentData The attachment relationship data.
type RelationshipToIncidentAttachmentData struct {
	// A unique identifier that represents the attachment.
	Id string `json:"id"`
	// The incident attachment resource type.
	Type IncidentAttachmentType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRelationshipToIncidentAttachmentData instantiates a new RelationshipToIncidentAttachmentData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRelationshipToIncidentAttachmentData(id string, typeVar IncidentAttachmentType) *RelationshipToIncidentAttachmentData {
	this := RelationshipToIncidentAttachmentData{}
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewRelationshipToIncidentAttachmentDataWithDefaults instantiates a new RelationshipToIncidentAttachmentData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRelationshipToIncidentAttachmentDataWithDefaults() *RelationshipToIncidentAttachmentData {
	this := RelationshipToIncidentAttachmentData{}
	var typeVar IncidentAttachmentType = INCIDENTATTACHMENTTYPE_INCIDENT_ATTACHMENTS
	this.Type = typeVar
	return &this
}

// GetId returns the Id field value.
func (o *RelationshipToIncidentAttachmentData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *RelationshipToIncidentAttachmentData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *RelationshipToIncidentAttachmentData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *RelationshipToIncidentAttachmentData) GetType() IncidentAttachmentType {
	if o == nil {
		var ret IncidentAttachmentType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *RelationshipToIncidentAttachmentData) GetTypeOk() (*IncidentAttachmentType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *RelationshipToIncidentAttachmentData) SetType(v IncidentAttachmentType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RelationshipToIncidentAttachmentData) MarshalJSON() ([]byte, error) {
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
func (o *RelationshipToIncidentAttachmentData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Id   *string                 `json:"id"`
		Type *IncidentAttachmentType `json:"type"`
	}{}
	all := struct {
		Id   string                 `json:"id"`
		Type IncidentAttachmentType `json:"type"`
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
