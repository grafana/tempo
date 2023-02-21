// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// NullableRelationshipToUserData Relationship to user object.
type NullableRelationshipToUserData struct {
	// A unique identifier that represents the user.
	Id string `json:"id"`
	// Users resource type.
	Type UsersType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewNullableRelationshipToUserData instantiates a new NullableRelationshipToUserData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNullableRelationshipToUserData(id string, typeVar UsersType) *NullableRelationshipToUserData {
	this := NullableRelationshipToUserData{}
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewNullableRelationshipToUserDataWithDefaults instantiates a new NullableRelationshipToUserData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNullableRelationshipToUserDataWithDefaults() *NullableRelationshipToUserData {
	this := NullableRelationshipToUserData{}
	var typeVar UsersType = USERSTYPE_USERS
	this.Type = typeVar
	return &this
}

// GetId returns the Id field value.
func (o *NullableRelationshipToUserData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *NullableRelationshipToUserData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *NullableRelationshipToUserData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *NullableRelationshipToUserData) GetType() UsersType {
	if o == nil {
		var ret UsersType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *NullableRelationshipToUserData) GetTypeOk() (*UsersType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *NullableRelationshipToUserData) SetType(v UsersType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o NullableRelationshipToUserData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["id"] = o.Id
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NullableRelationshipToUserData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Id   *string    `json:"id"`
		Type *UsersType `json:"type"`
	}{}
	all := struct {
		Id   string    `json:"id"`
		Type UsersType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
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
	o.Id = all.Id
	o.Type = all.Type
	return nil
}

// NullableNullableRelationshipToUserData handles when a null is used for NullableRelationshipToUserData.
type NullableNullableRelationshipToUserData struct {
	value *NullableRelationshipToUserData
	isSet bool
}

// Get returns the associated value.
func (v NullableNullableRelationshipToUserData) Get() *NullableRelationshipToUserData {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNullableRelationshipToUserData) Set(val *NullableRelationshipToUserData) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNullableRelationshipToUserData) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableNullableRelationshipToUserData) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNullableRelationshipToUserData initializes the struct as if Set has been called.
func NewNullableNullableRelationshipToUserData(val *NullableRelationshipToUserData) *NullableNullableRelationshipToUserData {
	return &NullableNullableRelationshipToUserData{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNullableRelationshipToUserData) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNullableRelationshipToUserData) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
