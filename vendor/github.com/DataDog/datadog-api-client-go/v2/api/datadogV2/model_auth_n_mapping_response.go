// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// AuthNMappingResponse AuthN Mapping response from the API.
type AuthNMappingResponse struct {
	// The AuthN Mapping object returned by API.
	Data *AuthNMapping `json:"data,omitempty"`
	// Included data in the AuthN Mapping response.
	Included []AuthNMappingIncluded `json:"included,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAuthNMappingResponse instantiates a new AuthNMappingResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAuthNMappingResponse() *AuthNMappingResponse {
	this := AuthNMappingResponse{}
	return &this
}

// NewAuthNMappingResponseWithDefaults instantiates a new AuthNMappingResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAuthNMappingResponseWithDefaults() *AuthNMappingResponse {
	this := AuthNMappingResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *AuthNMappingResponse) GetData() AuthNMapping {
	if o == nil || o.Data == nil {
		var ret AuthNMapping
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingResponse) GetDataOk() (*AuthNMapping, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *AuthNMappingResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given AuthNMapping and assigns it to the Data field.
func (o *AuthNMappingResponse) SetData(v AuthNMapping) {
	o.Data = &v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *AuthNMappingResponse) GetIncluded() []AuthNMappingIncluded {
	if o == nil || o.Included == nil {
		var ret []AuthNMappingIncluded
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingResponse) GetIncludedOk() (*[]AuthNMappingIncluded, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *AuthNMappingResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []AuthNMappingIncluded and assigns it to the Included field.
func (o *AuthNMappingResponse) SetIncluded(v []AuthNMappingIncluded) {
	o.Included = v
}

// MarshalJSON serializes the struct using spec logic.
func (o AuthNMappingResponse) MarshalJSON() ([]byte, error) {
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
func (o *AuthNMappingResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data     *AuthNMapping          `json:"data,omitempty"`
		Included []AuthNMappingIncluded `json:"included,omitempty"`
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
