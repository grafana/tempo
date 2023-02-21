// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RoleUpdateResponse Response containing information about an updated role.
type RoleUpdateResponse struct {
	// Role object returned by the API.
	Data *RoleUpdateResponseData `json:"data,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRoleUpdateResponse instantiates a new RoleUpdateResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRoleUpdateResponse() *RoleUpdateResponse {
	this := RoleUpdateResponse{}
	return &this
}

// NewRoleUpdateResponseWithDefaults instantiates a new RoleUpdateResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRoleUpdateResponseWithDefaults() *RoleUpdateResponse {
	this := RoleUpdateResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *RoleUpdateResponse) GetData() RoleUpdateResponseData {
	if o == nil || o.Data == nil {
		var ret RoleUpdateResponseData
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleUpdateResponse) GetDataOk() (*RoleUpdateResponseData, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *RoleUpdateResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given RoleUpdateResponseData and assigns it to the Data field.
func (o *RoleUpdateResponse) SetData(v RoleUpdateResponseData) {
	o.Data = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RoleUpdateResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RoleUpdateResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data *RoleUpdateResponseData `json:"data,omitempty"`
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
	return nil
}
