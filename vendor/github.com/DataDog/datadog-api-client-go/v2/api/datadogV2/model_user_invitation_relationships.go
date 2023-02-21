// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// UserInvitationRelationships Relationships data for user invitation.
type UserInvitationRelationships struct {
	// Relationship to user.
	User RelationshipToUser `json:"user"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserInvitationRelationships instantiates a new UserInvitationRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserInvitationRelationships(user RelationshipToUser) *UserInvitationRelationships {
	this := UserInvitationRelationships{}
	this.User = user
	return &this
}

// NewUserInvitationRelationshipsWithDefaults instantiates a new UserInvitationRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserInvitationRelationshipsWithDefaults() *UserInvitationRelationships {
	this := UserInvitationRelationships{}
	return &this
}

// GetUser returns the User field value.
func (o *UserInvitationRelationships) GetUser() RelationshipToUser {
	if o == nil {
		var ret RelationshipToUser
		return ret
	}
	return o.User
}

// GetUserOk returns a tuple with the User field value
// and a boolean to check if the value has been set.
func (o *UserInvitationRelationships) GetUserOk() (*RelationshipToUser, bool) {
	if o == nil {
		return nil, false
	}
	return &o.User, true
}

// SetUser sets field value.
func (o *UserInvitationRelationships) SetUser(v RelationshipToUser) {
	o.User = v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserInvitationRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["user"] = o.User

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UserInvitationRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		User *RelationshipToUser `json:"user"`
	}{}
	all := struct {
		User RelationshipToUser `json:"user"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.User == nil {
		return fmt.Errorf("required field user missing")
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
	if all.User.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.User = all.User
	return nil
}
