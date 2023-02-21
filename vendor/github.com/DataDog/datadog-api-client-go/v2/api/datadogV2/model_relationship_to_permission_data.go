// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RelationshipToPermissionData Relationship to permission object.
type RelationshipToPermissionData struct {
	// ID of the permission.
	Id *string `json:"id,omitempty"`
	// Permissions resource type.
	Type *PermissionsType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRelationshipToPermissionData instantiates a new RelationshipToPermissionData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRelationshipToPermissionData() *RelationshipToPermissionData {
	this := RelationshipToPermissionData{}
	var typeVar PermissionsType = PERMISSIONSTYPE_PERMISSIONS
	this.Type = &typeVar
	return &this
}

// NewRelationshipToPermissionDataWithDefaults instantiates a new RelationshipToPermissionData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRelationshipToPermissionDataWithDefaults() *RelationshipToPermissionData {
	this := RelationshipToPermissionData{}
	var typeVar PermissionsType = PERMISSIONSTYPE_PERMISSIONS
	this.Type = &typeVar
	return &this
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *RelationshipToPermissionData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RelationshipToPermissionData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *RelationshipToPermissionData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *RelationshipToPermissionData) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *RelationshipToPermissionData) GetType() PermissionsType {
	if o == nil || o.Type == nil {
		var ret PermissionsType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RelationshipToPermissionData) GetTypeOk() (*PermissionsType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *RelationshipToPermissionData) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given PermissionsType and assigns it to the Type field.
func (o *RelationshipToPermissionData) SetType(v PermissionsType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RelationshipToPermissionData) MarshalJSON() ([]byte, error) {
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
func (o *RelationshipToPermissionData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Id   *string          `json:"id,omitempty"`
		Type *PermissionsType `json:"type,omitempty"`
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
