// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentTeamsResponse Response with a list of incident team payloads.
type IncidentTeamsResponse struct {
	// An array of incident teams.
	Data []IncidentTeamResponseData `json:"data"`
	// Included related resources which the user requested.
	Included []IncidentTeamIncludedItems `json:"included,omitempty"`
	// The metadata object containing pagination metadata.
	Meta *IncidentResponseMeta `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentTeamsResponse instantiates a new IncidentTeamsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentTeamsResponse(data []IncidentTeamResponseData) *IncidentTeamsResponse {
	this := IncidentTeamsResponse{}
	this.Data = data
	return &this
}

// NewIncidentTeamsResponseWithDefaults instantiates a new IncidentTeamsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentTeamsResponseWithDefaults() *IncidentTeamsResponse {
	this := IncidentTeamsResponse{}
	return &this
}

// GetData returns the Data field value.
func (o *IncidentTeamsResponse) GetData() []IncidentTeamResponseData {
	if o == nil {
		var ret []IncidentTeamResponseData
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *IncidentTeamsResponse) GetDataOk() (*[]IncidentTeamResponseData, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *IncidentTeamsResponse) SetData(v []IncidentTeamResponseData) {
	o.Data = v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *IncidentTeamsResponse) GetIncluded() []IncidentTeamIncludedItems {
	if o == nil || o.Included == nil {
		var ret []IncidentTeamIncludedItems
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentTeamsResponse) GetIncludedOk() (*[]IncidentTeamIncludedItems, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *IncidentTeamsResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []IncidentTeamIncludedItems and assigns it to the Included field.
func (o *IncidentTeamsResponse) SetIncluded(v []IncidentTeamIncludedItems) {
	o.Included = v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *IncidentTeamsResponse) GetMeta() IncidentResponseMeta {
	if o == nil || o.Meta == nil {
		var ret IncidentResponseMeta
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentTeamsResponse) GetMetaOk() (*IncidentResponseMeta, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *IncidentTeamsResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given IncidentResponseMeta and assigns it to the Meta field.
func (o *IncidentTeamsResponse) SetMeta(v IncidentResponseMeta) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentTeamsResponse) MarshalJSON() ([]byte, error) {
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
func (o *IncidentTeamsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *[]IncidentTeamResponseData `json:"data"`
	}{}
	all := struct {
		Data     []IncidentTeamResponseData  `json:"data"`
		Included []IncidentTeamIncludedItems `json:"included,omitempty"`
		Meta     *IncidentResponseMeta       `json:"meta,omitempty"`
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
