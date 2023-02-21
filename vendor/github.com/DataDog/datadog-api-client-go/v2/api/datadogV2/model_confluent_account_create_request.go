// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ConfluentAccountCreateRequest Payload schema when adding a Confluent account.
type ConfluentAccountCreateRequest struct {
	// The data body for adding a Confluent account.
	Data ConfluentAccountCreateRequestData `json:"data"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewConfluentAccountCreateRequest instantiates a new ConfluentAccountCreateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewConfluentAccountCreateRequest(data ConfluentAccountCreateRequestData) *ConfluentAccountCreateRequest {
	this := ConfluentAccountCreateRequest{}
	this.Data = data
	return &this
}

// NewConfluentAccountCreateRequestWithDefaults instantiates a new ConfluentAccountCreateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewConfluentAccountCreateRequestWithDefaults() *ConfluentAccountCreateRequest {
	this := ConfluentAccountCreateRequest{}
	return &this
}

// GetData returns the Data field value.
func (o *ConfluentAccountCreateRequest) GetData() ConfluentAccountCreateRequestData {
	if o == nil {
		var ret ConfluentAccountCreateRequestData
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *ConfluentAccountCreateRequest) GetDataOk() (*ConfluentAccountCreateRequestData, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *ConfluentAccountCreateRequest) SetData(v ConfluentAccountCreateRequestData) {
	o.Data = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ConfluentAccountCreateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data"] = o.Data

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ConfluentAccountCreateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *ConfluentAccountCreateRequestData `json:"data"`
	}{}
	all := struct {
		Data ConfluentAccountCreateRequestData `json:"data"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Data == nil {
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
	if all.Data.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Data = all.Data
	return nil
}
