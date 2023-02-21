// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsListResponse Response object with all logs matching the request and pagination information.
type LogsListResponse struct {
	// Array of logs matching the request.
	Data []Log `json:"data,omitempty"`
	// Links attributes.
	Links *LogsListResponseLinks `json:"links,omitempty"`
	// The metadata associated with a request
	Meta *LogsResponseMetadata `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsListResponse instantiates a new LogsListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsListResponse() *LogsListResponse {
	this := LogsListResponse{}
	return &this
}

// NewLogsListResponseWithDefaults instantiates a new LogsListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsListResponseWithDefaults() *LogsListResponse {
	this := LogsListResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *LogsListResponse) GetData() []Log {
	if o == nil || o.Data == nil {
		var ret []Log
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListResponse) GetDataOk() (*[]Log, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *LogsListResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []Log and assigns it to the Data field.
func (o *LogsListResponse) SetData(v []Log) {
	o.Data = v
}

// GetLinks returns the Links field value if set, zero value otherwise.
func (o *LogsListResponse) GetLinks() LogsListResponseLinks {
	if o == nil || o.Links == nil {
		var ret LogsListResponseLinks
		return ret
	}
	return *o.Links
}

// GetLinksOk returns a tuple with the Links field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListResponse) GetLinksOk() (*LogsListResponseLinks, bool) {
	if o == nil || o.Links == nil {
		return nil, false
	}
	return o.Links, true
}

// HasLinks returns a boolean if a field has been set.
func (o *LogsListResponse) HasLinks() bool {
	return o != nil && o.Links != nil
}

// SetLinks gets a reference to the given LogsListResponseLinks and assigns it to the Links field.
func (o *LogsListResponse) SetLinks(v LogsListResponseLinks) {
	o.Links = &v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *LogsListResponse) GetMeta() LogsResponseMetadata {
	if o == nil || o.Meta == nil {
		var ret LogsResponseMetadata
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListResponse) GetMetaOk() (*LogsResponseMetadata, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *LogsListResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given LogsResponseMetadata and assigns it to the Meta field.
func (o *LogsListResponse) SetMeta(v LogsResponseMetadata) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsListResponse) MarshalJSON() ([]byte, error) {
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
func (o *LogsListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data  []Log                  `json:"data,omitempty"`
		Links *LogsListResponseLinks `json:"links,omitempty"`
		Meta  *LogsResponseMetadata  `json:"meta,omitempty"`
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
