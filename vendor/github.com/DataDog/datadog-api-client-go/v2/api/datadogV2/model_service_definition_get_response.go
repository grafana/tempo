// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionGetResponse Get service definition response.
type ServiceDefinitionGetResponse struct {
	// Service definition data.
	Data *ServiceDefinitionData `json:"data,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionGetResponse instantiates a new ServiceDefinitionGetResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionGetResponse() *ServiceDefinitionGetResponse {
	this := ServiceDefinitionGetResponse{}
	return &this
}

// NewServiceDefinitionGetResponseWithDefaults instantiates a new ServiceDefinitionGetResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionGetResponseWithDefaults() *ServiceDefinitionGetResponse {
	this := ServiceDefinitionGetResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *ServiceDefinitionGetResponse) GetData() ServiceDefinitionData {
	if o == nil || o.Data == nil {
		var ret ServiceDefinitionData
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionGetResponse) GetDataOk() (*ServiceDefinitionData, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *ServiceDefinitionGetResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given ServiceDefinitionData and assigns it to the Data field.
func (o *ServiceDefinitionGetResponse) SetData(v ServiceDefinitionData) {
	o.Data = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionGetResponse) MarshalJSON() ([]byte, error) {
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
func (o *ServiceDefinitionGetResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data *ServiceDefinitionData `json:"data,omitempty"`
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
