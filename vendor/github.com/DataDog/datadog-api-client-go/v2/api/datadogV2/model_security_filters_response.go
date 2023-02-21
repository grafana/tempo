// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityFiltersResponse All the available security filters objects.
type SecurityFiltersResponse struct {
	// A list of security filters objects.
	Data []SecurityFilter `json:"data,omitempty"`
	// Optional metadata associated to the response.
	Meta *SecurityFilterMeta `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityFiltersResponse instantiates a new SecurityFiltersResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityFiltersResponse() *SecurityFiltersResponse {
	this := SecurityFiltersResponse{}
	return &this
}

// NewSecurityFiltersResponseWithDefaults instantiates a new SecurityFiltersResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityFiltersResponseWithDefaults() *SecurityFiltersResponse {
	this := SecurityFiltersResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *SecurityFiltersResponse) GetData() []SecurityFilter {
	if o == nil || o.Data == nil {
		var ret []SecurityFilter
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFiltersResponse) GetDataOk() (*[]SecurityFilter, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *SecurityFiltersResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []SecurityFilter and assigns it to the Data field.
func (o *SecurityFiltersResponse) SetData(v []SecurityFilter) {
	o.Data = v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *SecurityFiltersResponse) GetMeta() SecurityFilterMeta {
	if o == nil || o.Meta == nil {
		var ret SecurityFilterMeta
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFiltersResponse) GetMetaOk() (*SecurityFilterMeta, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *SecurityFiltersResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given SecurityFilterMeta and assigns it to the Meta field.
func (o *SecurityFiltersResponse) SetMeta(v SecurityFilterMeta) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityFiltersResponse) MarshalJSON() ([]byte, error) {
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
func (o *SecurityFiltersResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data []SecurityFilter    `json:"data,omitempty"`
		Meta *SecurityFilterMeta `json:"meta,omitempty"`
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
