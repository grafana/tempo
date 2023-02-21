// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentAttachmentsResponse The response object containing an incident's attachments.
type IncidentAttachmentsResponse struct {
	// An array of incident attachments.
	Data []IncidentAttachmentData `json:"data"`
	// Included related resources that the user requested.
	Included []IncidentAttachmentsResponseIncludedItem `json:"included,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentAttachmentsResponse instantiates a new IncidentAttachmentsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentAttachmentsResponse(data []IncidentAttachmentData) *IncidentAttachmentsResponse {
	this := IncidentAttachmentsResponse{}
	this.Data = data
	return &this
}

// NewIncidentAttachmentsResponseWithDefaults instantiates a new IncidentAttachmentsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentAttachmentsResponseWithDefaults() *IncidentAttachmentsResponse {
	this := IncidentAttachmentsResponse{}
	return &this
}

// GetData returns the Data field value.
func (o *IncidentAttachmentsResponse) GetData() []IncidentAttachmentData {
	if o == nil {
		var ret []IncidentAttachmentData
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentsResponse) GetDataOk() (*[]IncidentAttachmentData, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *IncidentAttachmentsResponse) SetData(v []IncidentAttachmentData) {
	o.Data = v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *IncidentAttachmentsResponse) GetIncluded() []IncidentAttachmentsResponseIncludedItem {
	if o == nil || o.Included == nil {
		var ret []IncidentAttachmentsResponseIncludedItem
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentsResponse) GetIncludedOk() (*[]IncidentAttachmentsResponseIncludedItem, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *IncidentAttachmentsResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []IncidentAttachmentsResponseIncludedItem and assigns it to the Included field.
func (o *IncidentAttachmentsResponse) SetIncluded(v []IncidentAttachmentsResponseIncludedItem) {
	o.Included = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentAttachmentsResponse) MarshalJSON() ([]byte, error) {
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
func (o *IncidentAttachmentsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *[]IncidentAttachmentData `json:"data"`
	}{}
	all := struct {
		Data     []IncidentAttachmentData                  `json:"data"`
		Included []IncidentAttachmentsResponseIncludedItem `json:"included,omitempty"`
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
	return nil
}
