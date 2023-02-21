// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentsResponse Response with a list of incidents.
type IncidentsResponse struct {
	// An array of incidents.
	Data []IncidentResponseData `json:"data"`
	// Included related resources that the user requested.
	Included []IncidentResponseIncludedItem `json:"included,omitempty"`
	// The metadata object containing pagination metadata.
	Meta *IncidentResponseMeta `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentsResponse instantiates a new IncidentsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentsResponse(data []IncidentResponseData) *IncidentsResponse {
	this := IncidentsResponse{}
	this.Data = data
	return &this
}

// NewIncidentsResponseWithDefaults instantiates a new IncidentsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentsResponseWithDefaults() *IncidentsResponse {
	this := IncidentsResponse{}
	return &this
}

// GetData returns the Data field value.
func (o *IncidentsResponse) GetData() []IncidentResponseData {
	if o == nil {
		var ret []IncidentResponseData
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *IncidentsResponse) GetDataOk() (*[]IncidentResponseData, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *IncidentsResponse) SetData(v []IncidentResponseData) {
	o.Data = v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *IncidentsResponse) GetIncluded() []IncidentResponseIncludedItem {
	if o == nil || o.Included == nil {
		var ret []IncidentResponseIncludedItem
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentsResponse) GetIncludedOk() (*[]IncidentResponseIncludedItem, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *IncidentsResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []IncidentResponseIncludedItem and assigns it to the Included field.
func (o *IncidentsResponse) SetIncluded(v []IncidentResponseIncludedItem) {
	o.Included = v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *IncidentsResponse) GetMeta() IncidentResponseMeta {
	if o == nil || o.Meta == nil {
		var ret IncidentResponseMeta
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentsResponse) GetMetaOk() (*IncidentResponseMeta, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *IncidentsResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given IncidentResponseMeta and assigns it to the Meta field.
func (o *IncidentsResponse) SetMeta(v IncidentResponseMeta) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentsResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data"] = o.Data
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
func (o *IncidentsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *[]IncidentResponseData `json:"data"`
	}{}
	all := struct {
		Data     []IncidentResponseData         `json:"data"`
		Included []IncidentResponseIncludedItem `json:"included,omitempty"`
		Meta     *IncidentResponseMeta          `json:"meta,omitempty"`
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
