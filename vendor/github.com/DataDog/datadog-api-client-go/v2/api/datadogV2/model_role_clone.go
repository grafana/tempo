// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RoleClone Data for the clone role request.
type RoleClone struct {
	// Attributes required to create a new role by cloning an existing one.
	Attributes RoleCloneAttributes `json:"attributes"`
	// Roles type.
	Type RolesType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRoleClone instantiates a new RoleClone object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRoleClone(attributes RoleCloneAttributes, typeVar RolesType) *RoleClone {
	this := RoleClone{}
	this.Attributes = attributes
	this.Type = typeVar
	return &this
}

// NewRoleCloneWithDefaults instantiates a new RoleClone object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRoleCloneWithDefaults() *RoleClone {
	this := RoleClone{}
	var typeVar RolesType = ROLESTYPE_ROLES
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *RoleClone) GetAttributes() RoleCloneAttributes {
	if o == nil {
		var ret RoleCloneAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *RoleClone) GetAttributesOk() (*RoleCloneAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *RoleClone) SetAttributes(v RoleCloneAttributes) {
	o.Attributes = v
}

// GetType returns the Type field value.
func (o *RoleClone) GetType() RolesType {
	if o == nil {
		var ret RolesType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *RoleClone) GetTypeOk() (*RolesType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *RoleClone) SetType(v RolesType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RoleClone) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RoleClone) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *RoleCloneAttributes `json:"attributes"`
		Type       *RolesType           `json:"type"`
	}{}
	all := struct {
		Attributes RoleCloneAttributes `json:"attributes"`
		Type       RolesType           `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("required field attributes missing")
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
	o.Type = all.Type
	return nil
}
