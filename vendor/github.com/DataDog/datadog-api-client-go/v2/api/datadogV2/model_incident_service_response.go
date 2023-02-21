// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentServiceResponse Response with an incident service payload.
type IncidentServiceResponse struct {
	// Incident Service data from responses.
	Data IncidentServiceResponseData `json:"data"`
	// Included objects from relationships.
	Included []IncidentServiceIncludedItems `json:"included,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentServiceResponse instantiates a new IncidentServiceResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentServiceResponse(data IncidentServiceResponseData) *IncidentServiceResponse {
	this := IncidentServiceResponse{}
	this.Data = data
	return &this
}

// NewIncidentServiceResponseWithDefaults instantiates a new IncidentServiceResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentServiceResponseWithDefaults() *IncidentServiceResponse {
	this := IncidentServiceResponse{}
	return &this
}

// GetData returns the Data field value.
func (o *IncidentServiceResponse) GetData() IncidentServiceResponseData {
	if o == nil {
		var ret IncidentServiceResponseData
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *IncidentServiceResponse) GetDataOk() (*IncidentServiceResponseData, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *IncidentServiceResponse) SetData(v IncidentServiceResponseData) {
	o.Data = v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *IncidentServiceResponse) GetIncluded() []IncidentServiceIncludedItems {
	if o == nil || o.Included == nil {
		var ret []IncidentServiceIncludedItems
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentServiceResponse) GetIncludedOk() (*[]IncidentServiceIncludedItems, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *IncidentServiceResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []IncidentServiceIncludedItems and assigns it to the Included field.
func (o *IncidentServiceResponse) SetIncluded(v []IncidentServiceIncludedItems) {
	o.Included = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentServiceResponse) MarshalJSON() ([]byte, error) {
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
func (o *IncidentServiceResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *IncidentServiceResponseData `json:"data"`
	}{}
	all := struct {
		Data     IncidentServiceResponseData    `json:"data"`
		Included []IncidentServiceIncludedItems `json:"included,omitempty"`
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
