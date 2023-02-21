// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentAttachmentRelationships The incident attachment's relationships.
type IncidentAttachmentRelationships struct {
	// Relationship to user.
	LastModifiedByUser *RelationshipToUser `json:"last_modified_by_user,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentAttachmentRelationships instantiates a new IncidentAttachmentRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentAttachmentRelationships() *IncidentAttachmentRelationships {
	this := IncidentAttachmentRelationships{}
	return &this
}

// NewIncidentAttachmentRelationshipsWithDefaults instantiates a new IncidentAttachmentRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentAttachmentRelationshipsWithDefaults() *IncidentAttachmentRelationships {
	this := IncidentAttachmentRelationships{}
	return &this
}

// GetLastModifiedByUser returns the LastModifiedByUser field value if set, zero value otherwise.
func (o *IncidentAttachmentRelationships) GetLastModifiedByUser() RelationshipToUser {
	if o == nil || o.LastModifiedByUser == nil {
		var ret RelationshipToUser
		return ret
	}
	return *o.LastModifiedByUser
}

// GetLastModifiedByUserOk returns a tuple with the LastModifiedByUser field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentRelationships) GetLastModifiedByUserOk() (*RelationshipToUser, bool) {
	if o == nil || o.LastModifiedByUser == nil {
		return nil, false
	}
	return o.LastModifiedByUser, true
}

// HasLastModifiedByUser returns a boolean if a field has been set.
func (o *IncidentAttachmentRelationships) HasLastModifiedByUser() bool {
	return o != nil && o.LastModifiedByUser != nil
}

// SetLastModifiedByUser gets a reference to the given RelationshipToUser and assigns it to the LastModifiedByUser field.
func (o *IncidentAttachmentRelationships) SetLastModifiedByUser(v RelationshipToUser) {
	o.LastModifiedByUser = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentAttachmentRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.LastModifiedByUser != nil {
		toSerialize["last_modified_by_user"] = o.LastModifiedByUser
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentAttachmentRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		LastModifiedByUser *RelationshipToUser `json:"last_modified_by_user,omitempty"`
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
	if all.LastModifiedByUser != nil && all.LastModifiedByUser.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LastModifiedByUser = all.LastModifiedByUser
	return nil
}
