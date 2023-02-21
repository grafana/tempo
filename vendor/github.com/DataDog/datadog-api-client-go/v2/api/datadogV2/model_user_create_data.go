// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// UserCreateData Object to create a user.
type UserCreateData struct {
	// Attributes of the created user.
	Attributes UserCreateAttributes `json:"attributes"`
	// Relationships of the user object.
	Relationships *UserRelationships `json:"relationships,omitempty"`
	// Users resource type.
	Type UsersType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserCreateData instantiates a new UserCreateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserCreateData(attributes UserCreateAttributes, typeVar UsersType) *UserCreateData {
	this := UserCreateData{}
	this.Attributes = attributes
	this.Type = typeVar
	return &this
}

// NewUserCreateDataWithDefaults instantiates a new UserCreateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserCreateDataWithDefaults() *UserCreateData {
	this := UserCreateData{}
	var typeVar UsersType = USERSTYPE_USERS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *UserCreateData) GetAttributes() UserCreateAttributes {
	if o == nil {
		var ret UserCreateAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *UserCreateData) GetAttributesOk() (*UserCreateAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *UserCreateData) SetAttributes(v UserCreateAttributes) {
	o.Attributes = v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *UserCreateData) GetRelationships() UserRelationships {
	if o == nil || o.Relationships == nil {
		var ret UserRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserCreateData) GetRelationshipsOk() (*UserRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *UserCreateData) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given UserRelationships and assigns it to the Relationships field.
func (o *UserCreateData) SetRelationships(v UserRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *UserCreateData) GetType() UsersType {
	if o == nil {
		var ret UsersType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *UserCreateData) GetTypeOk() (*UsersType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *UserCreateData) SetType(v UsersType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserCreateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
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
func (o *UserCreateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *UserCreateAttributes `json:"attributes"`
		Type       *UsersType            `json:"type"`
	}{}
	all := struct {
		Attributes    UserCreateAttributes `json:"attributes"`
		Relationships *UserRelationships   `json:"relationships,omitempty"`
		Type          UsersType            `json:"type"`
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
