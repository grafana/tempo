// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsResponseMetadata The metadata associated with a request
type LogsResponseMetadata struct {
	// The time elapsed in milliseconds
	Elapsed *int64 `json:"elapsed,omitempty"`
	// Paging attributes.
	Page *LogsResponseMetadataPage `json:"page,omitempty"`
	// The identifier of the request
	RequestId *string `json:"request_id,omitempty"`
	// The status of the response
	Status *LogsAggregateResponseStatus `json:"status,omitempty"`
	// A list of warnings (non fatal errors) encountered, partial results might be returned if
	// warnings are present in the response.
	Warnings []LogsWarning `json:"warnings,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsResponseMetadata instantiates a new LogsResponseMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsResponseMetadata() *LogsResponseMetadata {
	this := LogsResponseMetadata{}
	return &this
}

// NewLogsResponseMetadataWithDefaults instantiates a new LogsResponseMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsResponseMetadataWithDefaults() *LogsResponseMetadata {
	this := LogsResponseMetadata{}
	return &this
}

// GetElapsed returns the Elapsed field value if set, zero value otherwise.
func (o *LogsResponseMetadata) GetElapsed() int64 {
	if o == nil || o.Elapsed == nil {
		var ret int64
		return ret
	}
	return *o.Elapsed
}

// GetElapsedOk returns a tuple with the Elapsed field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsResponseMetadata) GetElapsedOk() (*int64, bool) {
	if o == nil || o.Elapsed == nil {
		return nil, false
	}
	return o.Elapsed, true
}

// HasElapsed returns a boolean if a field has been set.
func (o *LogsResponseMetadata) HasElapsed() bool {
	return o != nil && o.Elapsed != nil
}

// SetElapsed gets a reference to the given int64 and assigns it to the Elapsed field.
func (o *LogsResponseMetadata) SetElapsed(v int64) {
	o.Elapsed = &v
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *LogsResponseMetadata) GetPage() LogsResponseMetadataPage {
	if o == nil || o.Page == nil {
		var ret LogsResponseMetadataPage
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsResponseMetadata) GetPageOk() (*LogsResponseMetadataPage, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *LogsResponseMetadata) HasPage() bool {
	return o != nil && o.Page != nil
}

// SetPage gets a reference to the given LogsResponseMetadataPage and assigns it to the Page field.
func (o *LogsResponseMetadata) SetPage(v LogsResponseMetadataPage) {
	o.Page = &v
}

// GetRequestId returns the RequestId field value if set, zero value otherwise.
func (o *LogsResponseMetadata) GetRequestId() string {
	if o == nil || o.RequestId == nil {
		var ret string
		return ret
	}
	return *o.RequestId
}

// GetRequestIdOk returns a tuple with the RequestId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsResponseMetadata) GetRequestIdOk() (*string, bool) {
	if o == nil || o.RequestId == nil {
		return nil, false
	}
	return o.RequestId, true
}

// HasRequestId returns a boolean if a field has been set.
func (o *LogsResponseMetadata) HasRequestId() bool {
	return o != nil && o.RequestId != nil
}

// SetRequestId gets a reference to the given string and assigns it to the RequestId field.
func (o *LogsResponseMetadata) SetRequestId(v string) {
	o.RequestId = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *LogsResponseMetadata) GetStatus() LogsAggregateResponseStatus {
	if o == nil || o.Status == nil {
		var ret LogsAggregateResponseStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsResponseMetadata) GetStatusOk() (*LogsAggregateResponseStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *LogsResponseMetadata) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given LogsAggregateResponseStatus and assigns it to the Status field.
func (o *LogsResponseMetadata) SetStatus(v LogsAggregateResponseStatus) {
	o.Status = &v
}

// GetWarnings returns the Warnings field value if set, zero value otherwise.
func (o *LogsResponseMetadata) GetWarnings() []LogsWarning {
	if o == nil || o.Warnings == nil {
		var ret []LogsWarning
		return ret
	}
	return o.Warnings
}

// GetWarningsOk returns a tuple with the Warnings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsResponseMetadata) GetWarningsOk() (*[]LogsWarning, bool) {
	if o == nil || o.Warnings == nil {
		return nil, false
	}
	return &o.Warnings, true
}

// HasWarnings returns a boolean if a field has been set.
func (o *LogsResponseMetadata) HasWarnings() bool {
	return o != nil && o.Warnings != nil
}

// SetWarnings gets a reference to the given []LogsWarning and assigns it to the Warnings field.
func (o *LogsResponseMetadata) SetWarnings(v []LogsWarning) {
	o.Warnings = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsResponseMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Elapsed != nil {
		toSerialize["elapsed"] = o.Elapsed
	}
	if o.Page != nil {
		toSerialize["page"] = o.Page
	}
	if o.RequestId != nil {
		toSerialize["request_id"] = o.RequestId
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.Warnings != nil {
		toSerialize["warnings"] = o.Warnings
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsResponseMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Elapsed   *int64                       `json:"elapsed,omitempty"`
		Page      *LogsResponseMetadataPage    `json:"page,omitempty"`
		RequestId *string                      `json:"request_id,omitempty"`
		Status    *LogsAggregateResponseStatus `json:"status,omitempty"`
		Warnings  []LogsWarning                `json:"warnings,omitempty"`
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
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Elapsed = all.Elapsed
	if all.Page != nil && all.Page.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Page = all.Page
	o.RequestId = all.RequestId
	o.Status = all.Status
	o.Warnings = all.Warnings
	return nil
}
