// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsAggregateResponse The response object for the logs aggregate API endpoint
type LogsAggregateResponse struct {
	// The query results
	Data *LogsAggregateResponseData `json:"data,omitempty"`
	// The metadata associated with a request
	Meta *LogsResponseMetadata `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsAggregateResponse instantiates a new LogsAggregateResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsAggregateResponse() *LogsAggregateResponse {
	this := LogsAggregateResponse{}
	return &this
}

// NewLogsAggregateResponseWithDefaults instantiates a new LogsAggregateResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsAggregateResponseWithDefaults() *LogsAggregateResponse {
	this := LogsAggregateResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *LogsAggregateResponse) GetData() LogsAggregateResponseData {
	if o == nil || o.Data == nil {
		var ret LogsAggregateResponseData
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAggregateResponse) GetDataOk() (*LogsAggregateResponseData, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *LogsAggregateResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given LogsAggregateResponseData and assigns it to the Data field.
func (o *LogsAggregateResponse) SetData(v LogsAggregateResponseData) {
	o.Data = &v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *LogsAggregateResponse) GetMeta() LogsResponseMetadata {
	if o == nil || o.Meta == nil {
		var ret LogsResponseMetadata
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAggregateResponse) GetMetaOk() (*LogsResponseMetadata, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *LogsAggregateResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given LogsResponseMetadata and assigns it to the Meta field.
func (o *LogsAggregateResponse) SetMeta(v LogsResponseMetadata) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsAggregateResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
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
func (o *LogsAggregateResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data *LogsAggregateResponseData `json:"data,omitempty"`
		Meta *LogsResponseMetadata      `json:"meta,omitempty"`
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
