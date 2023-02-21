// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RoleRelationships Relationships of the role object.
type RoleRelationships struct {
	// Relationship to multiple permissions objects.
	Permissions *RelationshipToPermissions `json:"permissions,omitempty"`
	// Relationship to users.
	Users *RelationshipToUsers `json:"users,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRoleRelationships instantiates a new RoleRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRoleRelationships() *RoleRelationships {
	this := RoleRelationships{}
	return &this
}

// NewRoleRelationshipsWithDefaults instantiates a new RoleRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRoleRelationshipsWithDefaults() *RoleRelationships {
	this := RoleRelationships{}
	return &this
}

// GetPermissions returns the Permissions field value if set, zero value otherwise.
func (o *RoleRelationships) GetPermissions() RelationshipToPermissions {
	if o == nil || o.Permissions == nil {
		var ret RelationshipToPermissions
		return ret
	}
	return *o.Permissions
}

// GetPermissionsOk returns a tuple with the Permissions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleRelationships) GetPermissionsOk() (*RelationshipToPermissions, bool) {
	if o == nil || o.Permissions == nil {
		return nil, false
	}
	return o.Permissions, true
}

// HasPermissions returns a boolean if a field has been set.
func (o *RoleRelationships) HasPermissions() bool {
	return o != nil && o.Permissions != nil
}

// SetPermissions gets a reference to the given RelationshipToPermissions and assigns it to the Permissions field.
func (o *RoleRelationships) SetPermissions(v RelationshipToPermissions) {
	o.Permissions = &v
}

// GetUsers returns the Users field value if set, zero value otherwise.
func (o *RoleRelationships) GetUsers() RelationshipToUsers {
	if o == nil || o.Users == nil {
		var ret RelationshipToUsers
		return ret
	}
	return *o.Users
}

// GetUsersOk returns a tuple with the Users field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleRelationships) GetUsersOk() (*RelationshipToUsers, bool) {
	if o == nil || o.Users == nil {
		return nil, false
	}
	return o.Users, true
}

// HasUsers returns a boolean if a field has been set.
func (o *RoleRelationships) HasUsers() bool {
	return o != nil && o.Users != nil
}

// SetUsers gets a reference to the given RelationshipToUsers and assigns it to the Users field.
func (o *RoleRelationships) SetUsers(v RelationshipToUsers) {
	o.Users = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RoleRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Permissions != nil {
		toSerialize["permissions"] = o.Permissions
	}
	if o.Users != nil {
		toSerialize["users"] = o.Users
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RoleRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Permissions *RelationshipToPermissions `json:"permissions,omitempty"`
		Users       *RelationshipToUsers       `json:"users,omitempty"`
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
	if all.Permissions != nil && all.Permissions.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Permissions = all.Permissions
	if all.Users != nil && all.Users.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Users = all.Users
	return nil
}
