// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RelationshipToRoleData Relationship to role object.
type RelationshipToRoleData struct {
	// The unique identifier of the role.
	Id *string `json:"id,omitempty"`
	// Roles type.
	Type *RolesType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRelationshipToRoleData instantiates a new RelationshipToRoleData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRelationshipToRoleData() *RelationshipToRoleData {
	this := RelationshipToRoleData{}
	var typeVar RolesType = ROLESTYPE_ROLES
	this.Type = &typeVar
	return &this
}

// NewRelationshipToRoleDataWithDefaults instantiates a new RelationshipToRoleData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRelationshipToRoleDataWithDefaults() *RelationshipToRoleData {
	this := RelationshipToRoleData{}
	var typeVar RolesType = ROLESTYPE_ROLES
	this.Type = &typeVar
	return &this
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *RelationshipToRoleData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RelationshipToRoleData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *RelationshipToRoleData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *RelationshipToRoleData) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *RelationshipToRoleData) GetType() RolesType {
	if o == nil || o.Type == nil {
		var ret RolesType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RelationshipToRoleData) GetTypeOk() (*RolesType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *RelationshipToRoleData) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given RolesType and assigns it to the Type field.
func (o *RelationshipToRoleData) SetType(v RolesType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RelationshipToRoleData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
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
func (o *RelationshipToRoleData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Id   *string    `json:"id,omitempty"`
		Type *RolesType `json:"type,omitempty"`
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
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
