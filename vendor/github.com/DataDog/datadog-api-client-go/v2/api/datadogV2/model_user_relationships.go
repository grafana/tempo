// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// UserRelationships Relationships of the user object.
type UserRelationships struct {
	// Relationship to roles.
	Roles *RelationshipToRoles `json:"roles,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserRelationships instantiates a new UserRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserRelationships() *UserRelationships {
	this := UserRelationships{}
	return &this
}

// NewUserRelationshipsWithDefaults instantiates a new UserRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserRelationshipsWithDefaults() *UserRelationships {
	this := UserRelationships{}
	return &this
}

// GetRoles returns the Roles field value if set, zero value otherwise.
func (o *UserRelationships) GetRoles() RelationshipToRoles {
	if o == nil || o.Roles == nil {
		var ret RelationshipToRoles
		return ret
	}
	return *o.Roles
}

// GetRolesOk returns a tuple with the Roles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserRelationships) GetRolesOk() (*RelationshipToRoles, bool) {
	if o == nil || o.Roles == nil {
		return nil, false
	}
	return o.Roles, true
}

// HasRoles returns a boolean if a field has been set.
func (o *UserRelationships) HasRoles() bool {
	return o != nil && o.Roles != nil
}

// SetRoles gets a reference to the given RelationshipToRoles and assigns it to the Roles field.
func (o *UserRelationships) SetRoles(v RelationshipToRoles) {
	o.Roles = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Roles != nil {
		toSerialize["roles"] = o.Roles
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UserRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Roles *RelationshipToRoles `json:"roles,omitempty"`
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
	if all.Roles != nil && all.Roles.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Roles = all.Roles
	return nil
}
