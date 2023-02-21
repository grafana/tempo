// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// NullableRelationshipToUser Relationship to user.
type NullableRelationshipToUser struct {
	// Relationship to user object.
	Data NullableNullableRelationshipToUserData `json:"data"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewNullableRelationshipToUser instantiates a new NullableRelationshipToUser object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNullableRelationshipToUser(data NullableNullableRelationshipToUserData) *NullableRelationshipToUser {
	this := NullableRelationshipToUser{}
	this.Data = data
	return &this
}

// NewNullableRelationshipToUserWithDefaults instantiates a new NullableRelationshipToUser object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNullableRelationshipToUserWithDefaults() *NullableRelationshipToUser {
	this := NullableRelationshipToUser{}
	return &this
}

// GetData returns the Data field value.
// If the value is explicit nil, the zero value for NullableRelationshipToUserData will be returned.
func (o *NullableRelationshipToUser) GetData() NullableRelationshipToUserData {
	if o == nil || o.Data.Get() == nil {
		var ret NullableRelationshipToUserData
		return ret
	}
	return *o.Data.Get()
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *NullableRelationshipToUser) GetDataOk() (*NullableRelationshipToUserData, bool) {
	if o == nil {
		return nil, false
	}
	return o.Data.Get(), o.Data.IsSet()
}

// SetData sets field value.
func (o *NullableRelationshipToUser) SetData(v NullableRelationshipToUserData) {
	o.Data.Set(&v)
}

// MarshalJSON serializes the struct using spec logic.
func (o NullableRelationshipToUser) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data"] = o.Data.Get()

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NullableRelationshipToUser) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data NullableNullableRelationshipToUserData `json:"data"`
	}{}
	all := struct {
		Data NullableNullableRelationshipToUserData `json:"data"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if !required.Data.IsSet() {
		return fmt.Errorf("required field data missing")
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
	o.Data = all.Data
	return nil
}
