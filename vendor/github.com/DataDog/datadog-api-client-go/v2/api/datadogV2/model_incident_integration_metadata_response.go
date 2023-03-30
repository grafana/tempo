// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentIntegrationMetadataResponse Response with an incident integration metadata.
type IncidentIntegrationMetadataResponse struct {
	// Incident integration metadata from a response.
	Data IncidentIntegrationMetadataResponseData `json:"data"`
	// Included related resources that the user requested.
	Included []IncidentIntegrationMetadataResponseIncludedItem `json:"included,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentIntegrationMetadataResponse instantiates a new IncidentIntegrationMetadataResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentIntegrationMetadataResponse(data IncidentIntegrationMetadataResponseData) *IncidentIntegrationMetadataResponse {
	this := IncidentIntegrationMetadataResponse{}
	this.Data = data
	return &this
}

// NewIncidentIntegrationMetadataResponseWithDefaults instantiates a new IncidentIntegrationMetadataResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentIntegrationMetadataResponseWithDefaults() *IncidentIntegrationMetadataResponse {
	this := IncidentIntegrationMetadataResponse{}
	return &this
}

// GetData returns the Data field value.
func (o *IncidentIntegrationMetadataResponse) GetData() IncidentIntegrationMetadataResponseData {
	if o == nil {
		var ret IncidentIntegrationMetadataResponseData
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *IncidentIntegrationMetadataResponse) GetDataOk() (*IncidentIntegrationMetadataResponseData, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *IncidentIntegrationMetadataResponse) SetData(v IncidentIntegrationMetadataResponseData) {
	o.Data = v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *IncidentIntegrationMetadataResponse) GetIncluded() []IncidentIntegrationMetadataResponseIncludedItem {
	if o == nil || o.Included == nil {
		var ret []IncidentIntegrationMetadataResponseIncludedItem
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentIntegrationMetadataResponse) GetIncludedOk() (*[]IncidentIntegrationMetadataResponseIncludedItem, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *IncidentIntegrationMetadataResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []IncidentIntegrationMetadataResponseIncludedItem and assigns it to the Included field.
func (o *IncidentIntegrationMetadataResponse) SetIncluded(v []IncidentIntegrationMetadataResponseIncludedItem) {
	o.Included = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentIntegrationMetadataResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data"] = o.Data
	if o.Included != nil {
		toSerialize["included"] = o.Included
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentIntegrationMetadataResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *IncidentIntegrationMetadataResponseData `json:"data"`
	}{}
	all := struct {
		Data     IncidentIntegrationMetadataResponseData           `json:"data"`
		Included []IncidentIntegrationMetadataResponseIncludedItem `json:"included,omitempty"`
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
	o.Included = all.Included
	return nil
}
