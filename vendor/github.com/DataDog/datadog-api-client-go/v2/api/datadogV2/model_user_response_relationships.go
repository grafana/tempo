// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// UserResponseRelationships Relationships of the user object returned by the API.
type UserResponseRelationships struct {
	// Relationship to an organization.
	Org *RelationshipToOrganization `json:"org,omitempty"`
	// Relationship to organizations.
	OtherOrgs *RelationshipToOrganizations `json:"other_orgs,omitempty"`
	// Relationship to users.
	OtherUsers *RelationshipToUsers `json:"other_users,omitempty"`
	// Relationship to roles.
	Roles *RelationshipToRoles `json:"roles,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserResponseRelationships instantiates a new UserResponseRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserResponseRelationships() *UserResponseRelationships {
	this := UserResponseRelationships{}
	return &this
}

// NewUserResponseRelationshipsWithDefaults instantiates a new UserResponseRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserResponseRelationshipsWithDefaults() *UserResponseRelationships {
	this := UserResponseRelationships{}
	return &this
}

// GetOrg returns the Org field value if set, zero value otherwise.
func (o *UserResponseRelationships) GetOrg() RelationshipToOrganization {
	if o == nil || o.Org == nil {
		var ret RelationshipToOrganization
		return ret
	}
	return *o.Org
}

// GetOrgOk returns a tuple with the Org field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserResponseRelationships) GetOrgOk() (*RelationshipToOrganization, bool) {
	if o == nil || o.Org == nil {
		return nil, false
	}
	return o.Org, true
}

// HasOrg returns a boolean if a field has been set.
func (o *UserResponseRelationships) HasOrg() bool {
	return o != nil && o.Org != nil
}

// SetOrg gets a reference to the given RelationshipToOrganization and assigns it to the Org field.
func (o *UserResponseRelationships) SetOrg(v RelationshipToOrganization) {
	o.Org = &v
}

// GetOtherOrgs returns the OtherOrgs field value if set, zero value otherwise.
func (o *UserResponseRelationships) GetOtherOrgs() RelationshipToOrganizations {
	if o == nil || o.OtherOrgs == nil {
		var ret RelationshipToOrganizations
		return ret
	}
	return *o.OtherOrgs
}

// GetOtherOrgsOk returns a tuple with the OtherOrgs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserResponseRelationships) GetOtherOrgsOk() (*RelationshipToOrganizations, bool) {
	if o == nil || o.OtherOrgs == nil {
		return nil, false
	}
	return o.OtherOrgs, true
}

// HasOtherOrgs returns a boolean if a field has been set.
func (o *UserResponseRelationships) HasOtherOrgs() bool {
	return o != nil && o.OtherOrgs != nil
}

// SetOtherOrgs gets a reference to the given RelationshipToOrganizations and assigns it to the OtherOrgs field.
func (o *UserResponseRelationships) SetOtherOrgs(v RelationshipToOrganizations) {
	o.OtherOrgs = &v
}

// GetOtherUsers returns the OtherUsers field value if set, zero value otherwise.
func (o *UserResponseRelationships) GetOtherUsers() RelationshipToUsers {
	if o == nil || o.OtherUsers == nil {
		var ret RelationshipToUsers
		return ret
	}
	return *o.OtherUsers
}

// GetOtherUsersOk returns a tuple with the OtherUsers field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserResponseRelationships) GetOtherUsersOk() (*RelationshipToUsers, bool) {
	if o == nil || o.OtherUsers == nil {
		return nil, false
	}
	return o.OtherUsers, true
}

// HasOtherUsers returns a boolean if a field has been set.
func (o *UserResponseRelationships) HasOtherUsers() bool {
	return o != nil && o.OtherUsers != nil
}

// SetOtherUsers gets a reference to the given RelationshipToUsers and assigns it to the OtherUsers field.
func (o *UserResponseRelationships) SetOtherUsers(v RelationshipToUsers) {
	o.OtherUsers = &v
}

// GetRoles returns the Roles field value if set, zero value otherwise.
func (o *UserResponseRelationships) GetRoles() RelationshipToRoles {
	if o == nil || o.Roles == nil {
		var ret RelationshipToRoles
		return ret
	}
	return *o.Roles
}

// GetRolesOk returns a tuple with the Roles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserResponseRelationships) GetRolesOk() (*RelationshipToRoles, bool) {
	if o == nil || o.Roles == nil {
		return nil, false
	}
	return o.Roles, true
}

// HasRoles returns a boolean if a field has been set.
func (o *UserResponseRelationships) HasRoles() bool {
	return o != nil && o.Roles != nil
}

// SetRoles gets a reference to the given RelationshipToRoles and assigns it to the Roles field.
func (o *UserResponseRelationships) SetRoles(v RelationshipToRoles) {
	o.Roles = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserResponseRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Org != nil {
		toSerialize["org"] = o.Org
	}
	if o.OtherOrgs != nil {
		toSerialize["other_orgs"] = o.OtherOrgs
	}
	if o.OtherUsers != nil {
		toSerialize["other_users"] = o.OtherUsers
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
func (o *UserResponseRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Org        *RelationshipToOrganization  `json:"org,omitempty"`
		OtherOrgs  *RelationshipToOrganizations `json:"other_orgs,omitempty"`
		OtherUsers *RelationshipToUsers         `json:"other_users,omitempty"`
		Roles      *RelationshipToRoles         `json:"roles,omitempty"`
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
	if all.Org != nil && all.Org.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Org = all.Org
	if all.OtherOrgs != nil && all.OtherOrgs.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.OtherOrgs = all.OtherOrgs
	if all.OtherUsers != nil && all.OtherUsers.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.OtherUsers = all.OtherUsers
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
