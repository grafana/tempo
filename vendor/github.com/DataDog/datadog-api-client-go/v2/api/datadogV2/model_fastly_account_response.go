// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// FastlyAccountResponse The expected response schema when getting a Fastly account.
type FastlyAccountResponse struct {
	// Data object of a Fastly account.
	Data *FastlyAccountResponseData `json:"data,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewFastlyAccountResponse instantiates a new FastlyAccountResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFastlyAccountResponse() *FastlyAccountResponse {
	this := FastlyAccountResponse{}
	return &this
}

// NewFastlyAccountResponseWithDefaults instantiates a new FastlyAccountResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFastlyAccountResponseWithDefaults() *FastlyAccountResponse {
	this := FastlyAccountResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *FastlyAccountResponse) GetData() FastlyAccountResponseData {
	if o == nil || o.Data == nil {
		var ret FastlyAccountResponseData
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FastlyAccountResponse) GetDataOk() (*FastlyAccountResponseData, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *FastlyAccountResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given FastlyAccountResponseData and assigns it to the Data field.
func (o *FastlyAccountResponse) SetData(v FastlyAccountResponseData) {
	o.Data = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o FastlyAccountResponse) MarshalJSON() ([]byte, error) {
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
func (o *FastlyAccountResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data *FastlyAccountResponseData `json:"data,omitempty"`
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
