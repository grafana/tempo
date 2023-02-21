// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RUMEventsResponse Response object with all events matching the request and pagination information.
type RUMEventsResponse struct {
	// Array of events matching the request.
	Data []RUMEvent `json:"data,omitempty"`
	// Links attributes.
	Links *RUMResponseLinks `json:"links,omitempty"`
	// The metadata associated with a request.
	Meta *RUMResponseMetadata `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRUMEventsResponse instantiates a new RUMEventsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRUMEventsResponse() *RUMEventsResponse {
	this := RUMEventsResponse{}
	return &this
}

// NewRUMEventsResponseWithDefaults instantiates a new RUMEventsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRUMEventsResponseWithDefaults() *RUMEventsResponse {
	this := RUMEventsResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *RUMEventsResponse) GetData() []RUMEvent {
	if o == nil || o.Data == nil {
		var ret []RUMEvent
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMEventsResponse) GetDataOk() (*[]RUMEvent, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *RUMEventsResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []RUMEvent and assigns it to the Data field.
func (o *RUMEventsResponse) SetData(v []RUMEvent) {
	o.Data = v
}

// GetLinks returns the Links field value if set, zero value otherwise.
func (o *RUMEventsResponse) GetLinks() RUMResponseLinks {
	if o == nil || o.Links == nil {
		var ret RUMResponseLinks
		return ret
	}
	return *o.Links
}

// GetLinksOk returns a tuple with the Links field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMEventsResponse) GetLinksOk() (*RUMResponseLinks, bool) {
	if o == nil || o.Links == nil {
		return nil, false
	}
	return o.Links, true
}

// HasLinks returns a boolean if a field has been set.
func (o *RUMEventsResponse) HasLinks() bool {
	return o != nil && o.Links != nil
}

// SetLinks gets a reference to the given RUMResponseLinks and assigns it to the Links field.
func (o *RUMEventsResponse) SetLinks(v RUMResponseLinks) {
	o.Links = &v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *RUMEventsResponse) GetMeta() RUMResponseMetadata {
	if o == nil || o.Meta == nil {
		var ret RUMResponseMetadata
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMEventsResponse) GetMetaOk() (*RUMResponseMetadata, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *RUMEventsResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given RUMResponseMetadata and assigns it to the Meta field.
func (o *RUMEventsResponse) SetMeta(v RUMResponseMetadata) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RUMEventsResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	if o.Links != nil {
		toSerialize["links"] = o.Links
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
func (o *RUMEventsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data  []RUMEvent           `json:"data,omitempty"`
		Links *RUMResponseLinks    `json:"links,omitempty"`
		Meta  *RUMResponseMetadata `json:"meta,omitempty"`
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
	o.Data = all.Data
	if all.Links != nil && all.Links.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Links = all.Links
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
