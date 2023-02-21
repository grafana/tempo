// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// EventsResponseMetadata The metadata associated with a request.
type EventsResponseMetadata struct {
	// The time elapsed in milliseconds.
	Elapsed *int64 `json:"elapsed,omitempty"`
	// Pagination attributes.
	Page *EventsResponseMetadataPage `json:"page,omitempty"`
	// The identifier of the request.
	RequestId *string `json:"request_id,omitempty"`
	// A list of warnings (non-fatal errors) encountered. Partial results might be returned if
	// warnings are present in the response.
	Warnings []EventsWarning `json:"warnings,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewEventsResponseMetadata instantiates a new EventsResponseMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewEventsResponseMetadata() *EventsResponseMetadata {
	this := EventsResponseMetadata{}
	return &this
}

// NewEventsResponseMetadataWithDefaults instantiates a new EventsResponseMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewEventsResponseMetadataWithDefaults() *EventsResponseMetadata {
	this := EventsResponseMetadata{}
	return &this
}

// GetElapsed returns the Elapsed field value if set, zero value otherwise.
func (o *EventsResponseMetadata) GetElapsed() int64 {
	if o == nil || o.Elapsed == nil {
		var ret int64
		return ret
	}
	return *o.Elapsed
}

// GetElapsedOk returns a tuple with the Elapsed field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsResponseMetadata) GetElapsedOk() (*int64, bool) {
	if o == nil || o.Elapsed == nil {
		return nil, false
	}
	return o.Elapsed, true
}

// HasElapsed returns a boolean if a field has been set.
func (o *EventsResponseMetadata) HasElapsed() bool {
	return o != nil && o.Elapsed != nil
}

// SetElapsed gets a reference to the given int64 and assigns it to the Elapsed field.
func (o *EventsResponseMetadata) SetElapsed(v int64) {
	o.Elapsed = &v
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *EventsResponseMetadata) GetPage() EventsResponseMetadataPage {
	if o == nil || o.Page == nil {
		var ret EventsResponseMetadataPage
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsResponseMetadata) GetPageOk() (*EventsResponseMetadataPage, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *EventsResponseMetadata) HasPage() bool {
	return o != nil && o.Page != nil
}

// SetPage gets a reference to the given EventsResponseMetadataPage and assigns it to the Page field.
func (o *EventsResponseMetadata) SetPage(v EventsResponseMetadataPage) {
	o.Page = &v
}

// GetRequestId returns the RequestId field value if set, zero value otherwise.
func (o *EventsResponseMetadata) GetRequestId() string {
	if o == nil || o.RequestId == nil {
		var ret string
		return ret
	}
	return *o.RequestId
}

// GetRequestIdOk returns a tuple with the RequestId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsResponseMetadata) GetRequestIdOk() (*string, bool) {
	if o == nil || o.RequestId == nil {
		return nil, false
	}
	return o.RequestId, true
}

// HasRequestId returns a boolean if a field has been set.
func (o *EventsResponseMetadata) HasRequestId() bool {
	return o != nil && o.RequestId != nil
}

// SetRequestId gets a reference to the given string and assigns it to the RequestId field.
func (o *EventsResponseMetadata) SetRequestId(v string) {
	o.RequestId = &v
}

// GetWarnings returns the Warnings field value if set, zero value otherwise.
func (o *EventsResponseMetadata) GetWarnings() []EventsWarning {
	if o == nil || o.Warnings == nil {
		var ret []EventsWarning
		return ret
	}
	return o.Warnings
}

// GetWarningsOk returns a tuple with the Warnings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsResponseMetadata) GetWarningsOk() (*[]EventsWarning, bool) {
	if o == nil || o.Warnings == nil {
		return nil, false
	}
	return &o.Warnings, true
}

// HasWarnings returns a boolean if a field has been set.
func (o *EventsResponseMetadata) HasWarnings() bool {
	return o != nil && o.Warnings != nil
}

// SetWarnings gets a reference to the given []EventsWarning and assigns it to the Warnings field.
func (o *EventsResponseMetadata) SetWarnings(v []EventsWarning) {
	o.Warnings = v
}

// MarshalJSON serializes the struct using spec logic.
func (o EventsResponseMetadata) MarshalJSON() ([]byte, error) {
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
	if o.Warnings != nil {
		toSerialize["warnings"] = o.Warnings
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *EventsResponseMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Elapsed   *int64                      `json:"elapsed,omitempty"`
		Page      *EventsResponseMetadataPage `json:"page,omitempty"`
		RequestId *string                     `json:"request_id,omitempty"`
		Warnings  []EventsWarning             `json:"warnings,omitempty"`
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
	o.Warnings = all.Warnings
	return nil
}
