// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// UserInvitationData Object to create a user invitation.
type UserInvitationData struct {
	// Relationships data for user invitation.
	Relationships UserInvitationRelationships `json:"relationships"`
	// User invitations type.
	Type UserInvitationsType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserInvitationData instantiates a new UserInvitationData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserInvitationData(relationships UserInvitationRelationships, typeVar UserInvitationsType) *UserInvitationData {
	this := UserInvitationData{}
	this.Relationships = relationships
	this.Type = typeVar
	return &this
}

// NewUserInvitationDataWithDefaults instantiates a new UserInvitationData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserInvitationDataWithDefaults() *UserInvitationData {
	this := UserInvitationData{}
	var typeVar UserInvitationsType = USERINVITATIONSTYPE_USER_INVITATIONS
	this.Type = typeVar
	return &this
}

// GetRelationships returns the Relationships field value.
func (o *UserInvitationData) GetRelationships() UserInvitationRelationships {
	if o == nil {
		var ret UserInvitationRelationships
		return ret
	}
	return o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value
// and a boolean to check if the value has been set.
func (o *UserInvitationData) GetRelationshipsOk() (*UserInvitationRelationships, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Relationships, true
}

// SetRelationships sets field value.
func (o *UserInvitationData) SetRelationships(v UserInvitationRelationships) {
	o.Relationships = v
}

// GetType returns the Type field value.
func (o *UserInvitationData) GetType() UserInvitationsType {
	if o == nil {
		var ret UserInvitationsType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *UserInvitationData) GetTypeOk() (*UserInvitationsType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *UserInvitationData) SetType(v UserInvitationsType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserInvitationData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["relationships"] = o.Relationships
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UserInvitationData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Relationships *UserInvitationRelationships `json:"relationships"`
		Type          *UserInvitationsType         `json:"type"`
	}{}
	all := struct {
		Relationships UserInvitationRelationships `json:"relationships"`
		Type          UserInvitationsType         `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
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
