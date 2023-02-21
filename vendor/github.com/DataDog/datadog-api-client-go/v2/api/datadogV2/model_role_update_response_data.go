// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RoleUpdateResponseData Role object returned by the API.
type RoleUpdateResponseData struct {
	// Attributes of the role.
	Attributes *RoleUpdateAttributes `json:"attributes,omitempty"`
	// The unique identifier of the role.
	Id *string `json:"id,omitempty"`
	// Relationships of the role object returned by the API.
	Relationships *RoleResponseRelationships `json:"relationships,omitempty"`
	// Roles type.
	Type RolesType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRoleUpdateResponseData instantiates a new RoleUpdateResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRoleUpdateResponseData(typeVar RolesType) *RoleUpdateResponseData {
	this := RoleUpdateResponseData{}
	this.Type = typeVar
	return &this
}

// NewRoleUpdateResponseDataWithDefaults instantiates a new RoleUpdateResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRoleUpdateResponseDataWithDefaults() *RoleUpdateResponseData {
	this := RoleUpdateResponseData{}
	var typeVar RolesType = ROLESTYPE_ROLES
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *RoleUpdateResponseData) GetAttributes() RoleUpdateAttributes {
	if o == nil || o.Attributes == nil {
		var ret RoleUpdateAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleUpdateResponseData) GetAttributesOk() (*RoleUpdateAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *RoleUpdateResponseData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given RoleUpdateAttributes and assigns it to the Attributes field.
func (o *RoleUpdateResponseData) SetAttributes(v RoleUpdateAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *RoleUpdateResponseData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleUpdateResponseData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *RoleUpdateResponseData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *RoleUpdateResponseData) SetId(v string) {
	o.Id = &v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *RoleUpdateResponseData) GetRelationships() RoleResponseRelationships {
	if o == nil || o.Relationships == nil {
		var ret RoleResponseRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleUpdateResponseData) GetRelationshipsOk() (*RoleResponseRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *RoleUpdateResponseData) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given RoleResponseRelationships and assigns it to the Relationships field.
func (o *RoleUpdateResponseData) SetRelationships(v RoleResponseRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *RoleUpdateResponseData) GetType() RolesType {
	if o == nil {
		var ret RolesType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *RoleUpdateResponseData) GetTypeOk() (*RolesType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *RoleUpdateResponseData) SetType(v RolesType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RoleUpdateResponseData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
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
func (o *RoleUpdateResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *RolesType `json:"type"`
	}{}
	all := struct {
		Attributes    *RoleUpdateAttributes      `json:"attributes,omitempty"`
		Id            *string                    `json:"id,omitempty"`
		Relationships *RoleResponseRelationships `json:"relationships,omitempty"`
		Type          RolesType                  `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
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
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
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
