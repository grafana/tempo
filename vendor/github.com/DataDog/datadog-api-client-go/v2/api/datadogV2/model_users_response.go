// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// UsersResponse Response containing information about multiple users.
type UsersResponse struct {
	// Array of returned users.
	Data []User `json:"data,omitempty"`
	// Array of objects related to the users.
	Included []UserResponseIncludedItem `json:"included,omitempty"`
	// Object describing meta attributes of response.
	Meta *ResponseMetaAttributes `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUsersResponse instantiates a new UsersResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsersResponse() *UsersResponse {
	this := UsersResponse{}
	return &this
}

// NewUsersResponseWithDefaults instantiates a new UsersResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsersResponseWithDefaults() *UsersResponse {
	this := UsersResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *UsersResponse) GetData() []User {
	if o == nil || o.Data == nil {
		var ret []User
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsersResponse) GetDataOk() (*[]User, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *UsersResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []User and assigns it to the Data field.
func (o *UsersResponse) SetData(v []User) {
	o.Data = v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *UsersResponse) GetIncluded() []UserResponseIncludedItem {
	if o == nil || o.Included == nil {
		var ret []UserResponseIncludedItem
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsersResponse) GetIncludedOk() (*[]UserResponseIncludedItem, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *UsersResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []UserResponseIncludedItem and assigns it to the Included field.
func (o *UsersResponse) SetIncluded(v []UserResponseIncludedItem) {
	o.Included = v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *UsersResponse) GetMeta() ResponseMetaAttributes {
	if o == nil || o.Meta == nil {
		var ret ResponseMetaAttributes
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsersResponse) GetMetaOk() (*ResponseMetaAttributes, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *UsersResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given ResponseMetaAttributes and assigns it to the Meta field.
func (o *UsersResponse) SetMeta(v ResponseMetaAttributes) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsersResponse) MarshalJSON() ([]byte, error) {
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
	if o.Meta != nil {
		toSerialize["meta"] = o.Meta
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsersResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data     []User                     `json:"data,omitempty"`
		Included []UserResponseIncludedItem `json:"included,omitempty"`
		Meta     *ResponseMetaAttributes    `json:"meta,omitempty"`
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
	o.Data = all.Data
	o.Included = all.Included
	if all.Meta != nil && all.Meta.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Meta = all.Meta
	return nil
}
