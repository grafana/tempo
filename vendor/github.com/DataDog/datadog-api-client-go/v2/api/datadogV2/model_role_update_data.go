// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RoleUpdateData Data related to the update of a role.
type RoleUpdateData struct {
	// Attributes of the role.
	Attributes RoleUpdateAttributes `json:"attributes"`
	// The unique identifier of the role.
	Id string `json:"id"`
	// Relationships of the role object.
	Relationships *RoleRelationships `json:"relationships,omitempty"`
	// Roles type.
	Type RolesType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRoleUpdateData instantiates a new RoleUpdateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRoleUpdateData(attributes RoleUpdateAttributes, id string, typeVar RolesType) *RoleUpdateData {
	this := RoleUpdateData{}
	this.Attributes = attributes
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewRoleUpdateDataWithDefaults instantiates a new RoleUpdateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRoleUpdateDataWithDefaults() *RoleUpdateData {
	this := RoleUpdateData{}
	var typeVar RolesType = ROLESTYPE_ROLES
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *RoleUpdateData) GetAttributes() RoleUpdateAttributes {
	if o == nil {
		var ret RoleUpdateAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *RoleUpdateData) GetAttributesOk() (*RoleUpdateAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *RoleUpdateData) SetAttributes(v RoleUpdateAttributes) {
	o.Attributes = v
}

// GetId returns the Id field value.
func (o *RoleUpdateData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *RoleUpdateData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *RoleUpdateData) SetId(v string) {
	o.Id = v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *RoleUpdateData) GetRelationships() RoleRelationships {
	if o == nil || o.Relationships == nil {
		var ret RoleRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleUpdateData) GetRelationshipsOk() (*RoleRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *RoleUpdateData) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given RoleRelationships and assigns it to the Relationships field.
func (o *RoleUpdateData) SetRelationships(v RoleRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *RoleUpdateData) GetType() RolesType {
	if o == nil {
		var ret RolesType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *RoleUpdateData) GetTypeOk() (*RolesType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *RoleUpdateData) SetType(v RolesType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RoleUpdateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["id"] = o.Id
	if o.Relationships != nil {
		toSerialize["relationships"] = o.Relationships
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RoleUpdateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *RoleUpdateAttributes `json:"attributes"`
		Id         *string               `json:"id"`
		Type       *RolesType            `json:"type"`
	}{}
	all := struct {
		Attributes    RoleUpdateAttributes `json:"attributes"`
		Id            string               `json:"id"`
		Relationships *RoleRelationships   `json:"relationships,omitempty"`
		Type          RolesType            `json:"type"`
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
	if all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Id = all.Id
	if all.Relationships != nil && all.Relationships.UnparsedObject != nil && o.UnparsedObject == nil {
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
