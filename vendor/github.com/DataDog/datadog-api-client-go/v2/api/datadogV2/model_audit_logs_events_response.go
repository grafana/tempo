// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// AuditLogsEventsResponse Response object with all events matching the request and pagination information.
type AuditLogsEventsResponse struct {
	// Array of events matching the request.
	Data []AuditLogsEvent `json:"data,omitempty"`
	// Links attributes.
	Links *AuditLogsResponseLinks `json:"links,omitempty"`
	// The metadata associated with a request.
	Meta *AuditLogsResponseMetadata `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAuditLogsEventsResponse instantiates a new AuditLogsEventsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAuditLogsEventsResponse() *AuditLogsEventsResponse {
	this := AuditLogsEventsResponse{}
	return &this
}

// NewAuditLogsEventsResponseWithDefaults instantiates a new AuditLogsEventsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAuditLogsEventsResponseWithDefaults() *AuditLogsEventsResponse {
	this := AuditLogsEventsResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *AuditLogsEventsResponse) GetData() []AuditLogsEvent {
	if o == nil || o.Data == nil {
		var ret []AuditLogsEvent
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuditLogsEventsResponse) GetDataOk() (*[]AuditLogsEvent, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *AuditLogsEventsResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []AuditLogsEvent and assigns it to the Data field.
func (o *AuditLogsEventsResponse) SetData(v []AuditLogsEvent) {
	o.Data = v
}

// GetLinks returns the Links field value if set, zero value otherwise.
func (o *AuditLogsEventsResponse) GetLinks() AuditLogsResponseLinks {
	if o == nil || o.Links == nil {
		var ret AuditLogsResponseLinks
		return ret
	}
	return *o.Links
}

// GetLinksOk returns a tuple with the Links field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuditLogsEventsResponse) GetLinksOk() (*AuditLogsResponseLinks, bool) {
	if o == nil || o.Links == nil {
		return nil, false
	}
	return o.Links, true
}

// HasLinks returns a boolean if a field has been set.
func (o *AuditLogsEventsResponse) HasLinks() bool {
	return o != nil && o.Links != nil
}

// SetLinks gets a reference to the given AuditLogsResponseLinks and assigns it to the Links field.
func (o *AuditLogsEventsResponse) SetLinks(v AuditLogsResponseLinks) {
	o.Links = &v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *AuditLogsEventsResponse) GetMeta() AuditLogsResponseMetadata {
	if o == nil || o.Meta == nil {
		var ret AuditLogsResponseMetadata
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuditLogsEventsResponse) GetMetaOk() (*AuditLogsResponseMetadata, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *AuditLogsEventsResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given AuditLogsResponseMetadata and assigns it to the Meta field.
func (o *AuditLogsEventsResponse) SetMeta(v AuditLogsResponseMetadata) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AuditLogsEventsResponse) MarshalJSON() ([]byte, error) {
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
func (o *AuditLogsEventsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data  []AuditLogsEvent           `json:"data,omitempty"`
		Links *AuditLogsResponseLinks    `json:"links,omitempty"`
		Meta  *AuditLogsResponseMetadata `json:"meta,omitempty"`
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
