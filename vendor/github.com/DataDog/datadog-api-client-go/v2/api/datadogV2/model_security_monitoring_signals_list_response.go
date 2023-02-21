// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringSignalsListResponse The response object with all security signals matching the request
// and pagination information.
type SecurityMonitoringSignalsListResponse struct {
	// An array of security signals matching the request.
	Data []SecurityMonitoringSignal `json:"data,omitempty"`
	// Links attributes.
	Links *SecurityMonitoringSignalsListResponseLinks `json:"links,omitempty"`
	// Meta attributes.
	Meta *SecurityMonitoringSignalsListResponseMeta `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalsListResponse instantiates a new SecurityMonitoringSignalsListResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalsListResponse() *SecurityMonitoringSignalsListResponse {
	this := SecurityMonitoringSignalsListResponse{}
	return &this
}

// NewSecurityMonitoringSignalsListResponseWithDefaults instantiates a new SecurityMonitoringSignalsListResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalsListResponseWithDefaults() *SecurityMonitoringSignalsListResponse {
	this := SecurityMonitoringSignalsListResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalsListResponse) GetData() []SecurityMonitoringSignal {
	if o == nil || o.Data == nil {
		var ret []SecurityMonitoringSignal
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalsListResponse) GetDataOk() (*[]SecurityMonitoringSignal, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalsListResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []SecurityMonitoringSignal and assigns it to the Data field.
func (o *SecurityMonitoringSignalsListResponse) SetData(v []SecurityMonitoringSignal) {
	o.Data = v
}

// GetLinks returns the Links field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalsListResponse) GetLinks() SecurityMonitoringSignalsListResponseLinks {
	if o == nil || o.Links == nil {
		var ret SecurityMonitoringSignalsListResponseLinks
		return ret
	}
	return *o.Links
}

// GetLinksOk returns a tuple with the Links field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalsListResponse) GetLinksOk() (*SecurityMonitoringSignalsListResponseLinks, bool) {
	if o == nil || o.Links == nil {
		return nil, false
	}
	return o.Links, true
}

// HasLinks returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalsListResponse) HasLinks() bool {
	return o != nil && o.Links != nil
}

// SetLinks gets a reference to the given SecurityMonitoringSignalsListResponseLinks and assigns it to the Links field.
func (o *SecurityMonitoringSignalsListResponse) SetLinks(v SecurityMonitoringSignalsListResponseLinks) {
	o.Links = &v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalsListResponse) GetMeta() SecurityMonitoringSignalsListResponseMeta {
	if o == nil || o.Meta == nil {
		var ret SecurityMonitoringSignalsListResponseMeta
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalsListResponse) GetMetaOk() (*SecurityMonitoringSignalsListResponseMeta, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalsListResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given SecurityMonitoringSignalsListResponseMeta and assigns it to the Meta field.
func (o *SecurityMonitoringSignalsListResponse) SetMeta(v SecurityMonitoringSignalsListResponseMeta) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalsListResponse) MarshalJSON() ([]byte, error) {
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
func (o *SecurityMonitoringSignalsListResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data  []SecurityMonitoringSignal                  `json:"data,omitempty"`
		Links *SecurityMonitoringSignalsListResponseLinks `json:"links,omitempty"`
		Meta  *SecurityMonitoringSignalsListResponseMeta  `json:"meta,omitempty"`
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
