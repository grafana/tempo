// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// UserResponse Response containing information about a single user.
type UserResponse struct {
	// User object returned by the API.
	Data *User `json:"data,omitempty"`
	// Array of objects related to the user.
	Included []UserResponseIncludedItem `json:"included,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserResponse instantiates a new UserResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserResponse() *UserResponse {
	this := UserResponse{}
	return &this
}

// NewUserResponseWithDefaults instantiates a new UserResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserResponseWithDefaults() *UserResponse {
	this := UserResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *UserResponse) GetData() User {
	if o == nil || o.Data == nil {
		var ret User
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserResponse) GetDataOk() (*User, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *UserResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given User and assigns it to the Data field.
func (o *UserResponse) SetData(v User) {
	o.Data = &v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *UserResponse) GetIncluded() []UserResponseIncludedItem {
	if o == nil || o.Included == nil {
		var ret []UserResponseIncludedItem
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserResponse) GetIncludedOk() (*[]UserResponseIncludedItem, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *UserResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []UserResponseIncludedItem and assigns it to the Included field.
func (o *UserResponse) SetIncluded(v []UserResponseIncludedItem) {
	o.Included = v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	if o.Included != nil {
		toSerialize["included"] = o.Included
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UserResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data     *User                      `json:"data,omitempty"`
		Included []UserResponseIncludedItem `json:"included,omitempty"`
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
	if all.Data != nil && all.Data.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Data = all.Data
	o.Included = all.Included
	return nil
}
