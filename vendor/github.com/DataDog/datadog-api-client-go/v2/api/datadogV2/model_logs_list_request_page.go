// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsListRequestPage Paging attributes for listing logs.
type LogsListRequestPage struct {
	// List following results with a cursor provided in the previous query.
	Cursor *string `json:"cursor,omitempty"`
	// Maximum number of logs in the response.
	Limit *int32 `json:"limit,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsListRequestPage instantiates a new LogsListRequestPage object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsListRequestPage() *LogsListRequestPage {
	this := LogsListRequestPage{}
	var limit int32 = 10
	this.Limit = &limit
	return &this
}

// NewLogsListRequestPageWithDefaults instantiates a new LogsListRequestPage object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsListRequestPageWithDefaults() *LogsListRequestPage {
	this := LogsListRequestPage{}
	var limit int32 = 10
	this.Limit = &limit
	return &this
}

// GetCursor returns the Cursor field value if set, zero value otherwise.
func (o *LogsListRequestPage) GetCursor() string {
	if o == nil || o.Cursor == nil {
		var ret string
		return ret
	}
	return *o.Cursor
}

// GetCursorOk returns a tuple with the Cursor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListRequestPage) GetCursorOk() (*string, bool) {
	if o == nil || o.Cursor == nil {
		return nil, false
	}
	return o.Cursor, true
}

// HasCursor returns a boolean if a field has been set.
func (o *LogsListRequestPage) HasCursor() bool {
	return o != nil && o.Cursor != nil
}

// SetCursor gets a reference to the given string and assigns it to the Cursor field.
func (o *LogsListRequestPage) SetCursor(v string) {
	o.Cursor = &v
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *LogsListRequestPage) GetLimit() int32 {
	if o == nil || o.Limit == nil {
		var ret int32
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListRequestPage) GetLimitOk() (*int32, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *LogsListRequestPage) HasLimit() bool {
	return o != nil && o.Limit != nil
}

// SetLimit gets a reference to the given int32 and assigns it to the Limit field.
func (o *LogsListRequestPage) SetLimit(v int32) {
	o.Limit = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsListRequestPage) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Cursor != nil {
		toSerialize["cursor"] = o.Cursor
	}
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsListRequestPage) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cursor *string `json:"cursor,omitempty"`
		Limit  *int32  `json:"limit,omitempty"`
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
	o.Cursor = all.Cursor
	o.Limit = all.Limit
	return nil
}
