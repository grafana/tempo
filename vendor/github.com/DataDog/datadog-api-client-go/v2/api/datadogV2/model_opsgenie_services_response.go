// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// OpsgenieServicesResponse Response with a list of Opsgenie services.
type OpsgenieServicesResponse struct {
	// An array of Opsgenie services.
	Data []OpsgenieServiceResponseData `json:"data"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewOpsgenieServicesResponse instantiates a new OpsgenieServicesResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOpsgenieServicesResponse(data []OpsgenieServiceResponseData) *OpsgenieServicesResponse {
	this := OpsgenieServicesResponse{}
	this.Data = data
	return &this
}

// NewOpsgenieServicesResponseWithDefaults instantiates a new OpsgenieServicesResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOpsgenieServicesResponseWithDefaults() *OpsgenieServicesResponse {
	this := OpsgenieServicesResponse{}
	return &this
}

// GetData returns the Data field value.
func (o *OpsgenieServicesResponse) GetData() []OpsgenieServiceResponseData {
	if o == nil {
		var ret []OpsgenieServiceResponseData
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *OpsgenieServicesResponse) GetDataOk() (*[]OpsgenieServiceResponseData, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *OpsgenieServicesResponse) SetData(v []OpsgenieServiceResponseData) {
	o.Data = v
}

// MarshalJSON serializes the struct using spec logic.
func (o OpsgenieServicesResponse) MarshalJSON() ([]byte, error) {
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
func (o *OpsgenieServicesResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *[]OpsgenieServiceResponseData `json:"data"`
	}{}
	all := struct {
		Data []OpsgenieServiceResponseData `json:"data"`
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
	o.Data = all.Data
	return nil
}
